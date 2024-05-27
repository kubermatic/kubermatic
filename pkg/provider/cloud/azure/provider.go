/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

const (
	resourceNamePrefix = "kubernetes-"

	clusterTagKey = "cluster"

	// FinalizerSecurityGroup will instruct the deletion of the security group.
	FinalizerSecurityGroup = "kubermatic.k8c.io/cleanup-azure-security-group"
	// FinalizerRouteTable will instruct the deletion of the route table.
	FinalizerRouteTable = "kubermatic.k8c.io/cleanup-azure-route-table"
	// FinalizerSubnet will instruct the deletion of the subnet.
	FinalizerSubnet = "kubermatic.k8c.io/cleanup-azure-subnet"
	// FinalizerVNet will instruct the deletion of the virtual network.
	FinalizerVNet = "kubermatic.k8c.io/cleanup-azure-vnet"
	// FinalizerResourceGroup will instruct the deletion of the resource group.
	FinalizerResourceGroup = "kubermatic.k8c.io/cleanup-azure-resource-group"
	// FinalizerAvailabilitySet will instruct the deletion of the availability set.
	FinalizerAvailabilitySet = "kubermatic.k8c.io/cleanup-azure-availability-set"

	denyAllTCPSecGroupRuleName   = "deny_all_tcp"
	denyAllUDPSecGroupRuleName   = "deny_all_udp"
	allowAllICMPSecGroupRuleName = "icmp_by_allow_all"
)

type Azure struct {
	dc                *kubermaticv1.DatacenterSpecAzure
	log               *zap.SugaredLogger
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// New returns a new Azure provider.
func New(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Azure, error) {
	if dc.Spec.Azure == nil {
		return nil, errors.New("datacenter is not an Azure datacenter")
	}
	return &Azure{
		dc:                dc.Spec.Azure,
		log:               log.Logger,
		secretKeySelector: secretKeyGetter,
	}, nil
}

var _ provider.ReconcilingCloudProvider = &Azure{}

func getRegionFaultDomainCount(location string) (int32, error) {
	// Azure API doesn't allow programmatically getting the number of available fault domains in a given region.
	// We must therefore hardcode these based on https://docs.microsoft.com/en-us/azure/virtual-machines/windows/manage-availability
	//
	// The list of region codes was generated by `az vm list-skus --resource-type availabilitySets --query '[?name==`Aligned`].{Location:locationInfo[0].location, MaximumFaultDomainCount:capabilities[0].value}'`.
	var faultDomainsPerRegion = map[string]int32{
		"AustraliaCentral":   2,
		"AustraliaCentral2":  2,
		"australiaeast":      2,
		"australiasoutheast": 2,
		"brazilsouth":        3,
		"BrazilSoutheast":    2,
		"CanadaCentral":      3,
		"CanadaEast":         2,
		"CentralIndia":       3,
		"centralus":          3,
		"CentralUSEUAP":      1,
		"eastasia":           2,
		"eastus":             3,
		"eastus2":            3,
		"EastUS2EUAP":        2,
		"EastUSSTG":          1,
		"FranceCentral":      3,
		"FranceSouth":        2,
		"GermanyNorth":       2,
		"GermanyWestCentral": 2,
		"IsraelCentral":      2,
		"ItalyNorth":         2,
		"japaneast":          3,
		"japanwest":          2,
		"JioIndiaCentral":    2,
		"JioIndiaWest":       2,
		"KoreaCentral":       2,
		"KoreaSouth":         2,
		"MalaysiaSouth":      2,
		"MexicoCentral":      2,
		"NewZealandNorth":    1,
		"northcentralus":     3,
		"northeurope":        3,
		"NorwayEast":         2,
		"NorwayWest":         2,
		"PolandCentral":      2,
		"QatarCentral":       2,
		"SouthAfricaNorth":   2,
		"SouthAfricaWest":    2,
		"southcentralus":     3,
		"SouthCentralUSSTG":  2,
		"southeastasia":      2,
		"SouthIndia":         2,
		"SpainCentral":       2,
		"SwedenCentral":      3,
		"SwedenSouth":        2,
		"SwitzerlandNorth":   2,
		"SwitzerlandWest":    2,
		"TaiwanNorth":        2,
		"TaiwanNorthwest":    2,
		"UAECentral":         2,
		"UAENorth":           2,
		"uksouth":            2,
		"ukwest":             2,
		"westcentralus":      2,
		"westeurope":         3,
		"WestIndia":          2,
		"westus":             3,
		"westus2":            3,
		"WestUS3":            3,
	}
	for region, count := range faultDomainsPerRegion {
		if strings.EqualFold(location, region) {
			return count, nil
		}
	}
	return 0, fmt.Errorf("unknown region %q", location)
}

func (a *Azure) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error

	credentials, err := GetCredentialsForCluster(cluster.Spec.Cloud, a.secretKeySelector)
	if err != nil {
		return nil, err
	}

	clientSet, err := GetClientSet(credentials)
	if err != nil {
		return nil, err
	}

	logger := a.log.With("cluster", cluster.Name)
	if kuberneteshelper.HasFinalizer(cluster, FinalizerSecurityGroup) {
		logger.Infow("deleting security group", "group", cluster.Spec.Cloud.Azure.SecurityGroup)
		if err := deleteSecurityGroup(ctx, clientSet, cluster.Spec.Cloud); err != nil {
			return cluster, fmt.Errorf("failed to delete security group %q: %w", cluster.Spec.Cloud.Azure.SecurityGroup, err)
		}
		cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerSecurityGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerSubnet) {
		logger.Infow("deleting subnet", "subnet", cluster.Spec.Cloud.Azure.SubnetName)
		if err := deleteSubnet(ctx, clientSet, cluster.Spec.Cloud); err != nil {
			return cluster, fmt.Errorf("failed to delete sub-network %q: %w", cluster.Spec.Cloud.Azure.SubnetName, err)
		}
		cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerSubnet)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerRouteTable) {
		logger.Infow("deleting route table", "routeTableName", cluster.Spec.Cloud.Azure.RouteTableName)
		if err := deleteRouteTable(ctx, clientSet, cluster.Spec.Cloud); err != nil {
			return cluster, fmt.Errorf("failed to delete route table %q: %w", cluster.Spec.Cloud.Azure.RouteTableName, err)
		}
		cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerRouteTable)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerVNet) {
		logger.Infow("deleting vnet", "vnet", cluster.Spec.Cloud.Azure.VNetName)
		if err := deleteVNet(ctx, clientSet, cluster.Spec.Cloud); err != nil {
			return cluster, fmt.Errorf("failed to delete virtual network %q: %w", cluster.Spec.Cloud.Azure.VNetName, err)
		}

		cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerVNet)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerAvailabilitySet) {
		logger.Infow("deleting availability set", "availabilitySet", cluster.Spec.Cloud.Azure.AvailabilitySet)
		if err := deleteAvailabilitySet(ctx, clientSet, cluster.Spec.Cloud); err != nil {
			return cluster, fmt.Errorf("failed to delete availability set %q: %w", cluster.Spec.Cloud.Azure.AvailabilitySet, err)
		}

		cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerAvailabilitySet)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerResourceGroup) {
		logger.Infow("deleting resource group", "resourceGroup", cluster.Spec.Cloud.Azure.ResourceGroup)
		if err := deleteResourceGroup(ctx, clientSet, cluster.Spec.Cloud); err != nil {
			return cluster, fmt.Errorf("failed to delete resource group %q: %w", cluster.Spec.Cloud.Azure.ResourceGroup, err)
		}

		cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerResourceGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func (a *Azure) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return a.reconcileCluster(ctx, cluster, update, false, true)
}

func (a *Azure) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return a.reconcileCluster(ctx, cluster, update, true, true)
}

func (*Azure) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

func (a *Azure) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool, setTags bool) (*kubermaticv1.Cluster, error) {
	var err error
	logger := a.log.With("cluster", cluster.Name)
	location := a.dc.Location

	credentials, err := GetCredentialsForCluster(cluster.Spec.Cloud, a.secretKeySelector)
	if err != nil {
		return nil, err
	}

	clientSet, err := GetClientSet(credentials)
	if err != nil {
		return nil, err
	}

	if force || cluster.Spec.Cloud.Azure.ResourceGroup == "" {
		logger.Infow("reconciling resource group", "resourceGroup", cluster.Spec.Cloud.Azure.ResourceGroup)
		cluster, err = reconcileResourceGroup(ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.VNetName == "" {
		logger.Infow("reconciling vnet", "vnet", vnetName(cluster))
		cluster, err = reconcileVNet(ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.RouteTableName == "" {
		logger.Infow("reconciling route table", "routeTableName", routeTableName(cluster))
		cluster, err = reconcileRouteTable(ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.SubnetName == "" {
		logger.Infow("reconciling subnet", "subnet", subnetName(cluster))
		cluster, err = reconcileSubnet(ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.SecurityGroup == "" {
		logger.Infow("reconciling security group", "securityGroup", securityGroupName(cluster))
		cluster, err = reconcileSecurityGroup(ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.AvailabilitySet == "" {
		if cluster.Spec.Cloud.Azure.AssignAvailabilitySet == nil ||
			*cluster.Spec.Cloud.Azure.AssignAvailabilitySet {
			logger.Infow("reconciling AvailabilitySet", "availabilitySet", availabilitySetName(cluster))
			cluster, err = reconcileAvailabilitySet(ctx, clientSet, location, cluster, update)
			if err != nil {
				return nil, err
			}
		}
	}

	return cluster, nil
}

func (a *Azure) DefaultCloudSpec(ctx context.Context, clusterSpec *kubermaticv1.ClusterSpec) error {
	if clusterSpec.Cloud.Azure == nil {
		return errors.New("no Azure cloud spec found")
	}

	if clusterSpec.Cloud.Azure.LoadBalancerSKU == "" {
		clusterSpec.Cloud.Azure.LoadBalancerSKU = kubermaticv1.AzureStandardLBSKU
	}

	if clusterSpec.Cloud.Azure.NodePortsAllowedIPRanges == nil {
		switch clusterSpec.ClusterNetwork.IPFamily {
		case kubermaticv1.IPFamilyIPv4:
			clusterSpec.Cloud.Azure.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{resources.IPv4MatchAnyCIDR},
			}
		case kubermaticv1.IPFamilyDualStack:
			clusterSpec.Cloud.Azure.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{resources.IPv4MatchAnyCIDR, resources.IPv6MatchAnyCIDR},
			}
		}
	}

	return nil
}

func (a *Azure) ValidateCloudSpec(ctx context.Context, cloud kubermaticv1.CloudSpec) error {
	credentials, err := GetCredentialsForCluster(cloud, a.secretKeySelector)
	if err != nil {
		return err
	}

	credential, err := credentials.ToAzureCredential()
	if err != nil {
		return err
	}

	if cloud.Azure.ResourceGroup != "" {
		rgClient, err := getGroupsClient(credential, credentials.SubscriptionID)
		if err != nil {
			return err
		}

		if _, err = rgClient.Get(ctx, cloud.Azure.ResourceGroup, nil); err != nil {
			return err
		}
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}

	if cloud.Azure.VNetName != "" {
		vnetClient, err := getNetworksClient(credential, credentials.SubscriptionID)
		if err != nil {
			return err
		}

		if _, err = vnetClient.Get(ctx, resourceGroup, cloud.Azure.VNetName, nil); err != nil {
			return err
		}
	}

	if cloud.Azure.SubnetName != "" {
		subnetClient, err := getSubnetsClient(credential, credentials.SubscriptionID)
		if err != nil {
			return err
		}

		if _, err = subnetClient.Get(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, nil); err != nil {
			return err
		}
	}

	if cloud.Azure.RouteTableName != "" {
		routeTablesClient, err := getRouteTablesClient(credential, credentials.SubscriptionID)
		if err != nil {
			return err
		}

		if _, err = routeTablesClient.Get(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, nil); err != nil {
			return err
		}
	}

	if cloud.Azure.SecurityGroup != "" {
		sgClient, err := getSecurityGroupsClient(credential, credentials.SubscriptionID)
		if err != nil {
			return err
		}

		if _, err = sgClient.Get(ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, nil); err != nil {
			return err
		}
	}

	return nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (a *Azure) ValidateCloudSpecUpdate(_ context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	if oldSpec.Azure == nil || newSpec.Azure == nil {
		return errors.New("'azure' spec is empty")
	}

	// we validate that a couple of resources are not changed.
	// the exception being the provider itself updating it in case the field
	// was left empty to dynamically generate resources.

	if oldSpec.Azure.ResourceGroup != "" && oldSpec.Azure.ResourceGroup != newSpec.Azure.ResourceGroup {
		return fmt.Errorf("updating Azure resource group is not supported (was %s, updated to %s)", oldSpec.Azure.ResourceGroup, newSpec.Azure.ResourceGroup)
	}

	if oldSpec.Azure.VNetResourceGroup != "" && oldSpec.Azure.VNetResourceGroup != newSpec.Azure.VNetResourceGroup {
		return fmt.Errorf("updating Azure vnet resource group is not supported (was %s, updated to %s)", oldSpec.Azure.VNetResourceGroup, newSpec.Azure.VNetResourceGroup)
	}

	if oldSpec.Azure.VNetName != "" && oldSpec.Azure.VNetName != newSpec.Azure.VNetName {
		return fmt.Errorf("updating Azure vnet name is not supported (was %s, updated to %s)", oldSpec.Azure.VNetName, newSpec.Azure.VNetName)
	}

	if oldSpec.Azure.SubnetName != "" && oldSpec.Azure.SubnetName != newSpec.Azure.SubnetName {
		return fmt.Errorf("updating Azure subnet name is not supported (was %s, updated to %s)", oldSpec.Azure.SubnetName, newSpec.Azure.SubnetName)
	}

	if oldSpec.Azure.RouteTableName != "" && oldSpec.Azure.RouteTableName != newSpec.Azure.RouteTableName {
		return fmt.Errorf("updating Azure route table name is not supported (was %s, updated to %s)", oldSpec.Azure.RouteTableName, newSpec.Azure.RouteTableName)
	}

	if oldSpec.Azure.SecurityGroup != "" && oldSpec.Azure.SecurityGroup != newSpec.Azure.SecurityGroup {
		return fmt.Errorf("updating Azure security group is not supported (was %s, updated to %s)", oldSpec.Azure.SecurityGroup, newSpec.Azure.SecurityGroup)
	}

	if oldSpec.Azure.AvailabilitySet != "" && oldSpec.Azure.AvailabilitySet != newSpec.Azure.AvailabilitySet {
		return fmt.Errorf("updating Azure availability set is not supported (was %s, updated to %s)", oldSpec.Azure.AvailabilitySet, newSpec.Azure.AvailabilitySet)
	}

	return nil
}

type Credentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

func (c Credentials) ToAzureCredential() (*azidentity.ClientSecretCredential, error) {
	return azidentity.NewClientSecretCredential(c.TenantID, c.ClientID, c.ClientSecret, nil)
}
