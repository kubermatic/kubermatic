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
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	resourceNamePrefix = "kubernetes-"

	clusterTagKey = "cluster"

	// FinalizerSecurityGroup will instruct the deletion of the security group
	FinalizerSecurityGroup = "kubermatic.io/cleanup-azure-security-group"
	// FinalizerRouteTable will instruct the deletion of the route table
	FinalizerRouteTable = "kubermatic.io/cleanup-azure-route-table"
	// FinalizerSubnet will instruct the deletion of the subnet
	FinalizerSubnet = "kubermatic.io/cleanup-azure-subnet"
	// FinalizerVNet will instruct the deletion of the virtual network
	FinalizerVNet = "kubermatic.io/cleanup-azure-vnet"
	// FinalizerResourceGroup will instruct the deletion of the resource group
	FinalizerResourceGroup = "kubermatic.io/cleanup-azure-resource-group"
	// FinalizerAvailabilitySet will instruct the deletion of the availability set
	FinalizerAvailabilitySet = "kubermatic.io/cleanup-azure-availability-set"

	denyAllTCPSecGroupRuleName   = "deny_all_tcp"
	denyAllUDPSecGroupRuleName   = "deny_all_udp"
	allowAllICMPSecGroupRuleName = "icmp_by_allow_all"
)

type Azure struct {
	dc                *kubermaticv1.DatacenterSpecAzure
	log               *zap.SugaredLogger
	ctx               context.Context
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
		ctx:               context.TODO(),
		secretKeySelector: secretKeyGetter,
	}, nil
}

var _ provider.CloudProvider = &Azure{}

// Azure API doesn't allow programmatically getting the number of available fault domains in a given region.
// We must therefore hardcode these based on https://docs.microsoft.com/en-us/azure/virtual-machines/windows/manage-availability
//
// The list of region codes was generated by `az account list-locations | jq .[].id --raw-output | cut -d/ -f5 | sed -e 's/^/"/' -e 's/$/" : ,/'`
var faultDomainsPerRegion = map[string]int32{
	"eastasia":           2,
	"southeastasia":      2,
	"centralus":          3,
	"eastus":             3,
	"eastus2":            3,
	"westus":             3,
	"northcentralus":     3,
	"southcentralus":     3,
	"northeurope":        3,
	"westeurope":         3,
	"japanwest":          2,
	"japaneast":          2,
	"brazilsouth":        2,
	"australiaeast":      2,
	"australiasoutheast": 2,
	"southindia":         2,
	"centralindia":       2,
	"westindia":          2,
	"canadacentral":      3,
	"canadaeast":         2,
	"uksouth":            2,
	"ukwest":             2,
	"westcentralus":      2,
	"westus2":            2,
	"koreacentral":       2,
	"koreasouth":         2,
}

func (a *Azure) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error

	credentials, err := GetCredentialsForCluster(cluster.Spec.Cloud, a.secretKeySelector)
	if err != nil {
		return nil, err
	}

	clientSet, err := GetClientSet(cluster.Spec.Cloud, credentials)
	if err != nil {
		return nil, err
	}

	logger := a.log.With("cluster", cluster.Name)
	if kuberneteshelper.HasFinalizer(cluster, FinalizerSecurityGroup) {
		logger.Infow("deleting security group", "group", cluster.Spec.Cloud.Azure.SecurityGroup)
		if err := deleteSecurityGroup(a.ctx, clientSet, cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete security group %q: %v", cluster.Spec.Cloud.Azure.SecurityGroup, err)
			}
		}
		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerSecurityGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerRouteTable) {
		logger.Infow("deleting route table", "routeTableName", cluster.Spec.Cloud.Azure.RouteTableName)
		if err := deleteRouteTable(a.ctx, clientSet, cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete route table %q: %v", cluster.Spec.Cloud.Azure.RouteTableName, err)
			}
		}
		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerRouteTable)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerSubnet) {
		logger.Infow("deleting subnet", "subnet", cluster.Spec.Cloud.Azure.SubnetName)
		if err := deleteSubnet(a.ctx, clientSet, cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete sub-network %q: %v", cluster.Spec.Cloud.Azure.SubnetName, err)
			}
		}
		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerSubnet)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerVNet) {
		logger.Infow("deleting vnet", "vnet", cluster.Spec.Cloud.Azure.VNetName)
		if err := deleteVNet(a.ctx, clientSet, cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete virtual network %q: %v", cluster.Spec.Cloud.Azure.VNetName, err)
			}
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerVNet)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerResourceGroup) {
		logger.Infow("deleting resource group", "resourceGroup", cluster.Spec.Cloud.Azure.ResourceGroup)
		if err := deleteResourceGroup(a.ctx, clientSet, cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete resource group %q: %v", cluster.Spec.Cloud.Azure.ResourceGroup, err)
			}
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerResourceGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, FinalizerAvailabilitySet) {
		logger.Infow("deleting availability set", "availabilitySet", cluster.Spec.Cloud.Azure.AvailabilitySet)
		if err := deleteAvailabilitySet(a.ctx, clientSet, cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete availability set %q: %v", cluster.Spec.Cloud.Azure.AvailabilitySet, err)
			}
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerAvailabilitySet)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func (a *Azure) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return a.reconcileCluster(cluster, update, false, true)
}

func (a *Azure) ReconcileCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return a.reconcileCluster(cluster, update, true, true)
}

func (a *Azure) reconcileCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool, setTags bool) (*kubermaticv1.Cluster, error) {
	var err error
	logger := a.log.With("cluster", cluster.Name)
	location := a.dc.Location

	credentials, err := GetCredentialsForCluster(cluster.Spec.Cloud, a.secretKeySelector)
	if err != nil {
		return nil, err
	}

	clientSet, err := GetClientSet(cluster.Spec.Cloud, credentials)
	if err != nil {
		return nil, err
	}

	if force || cluster.Spec.Cloud.Azure.ResourceGroup == "" {
		logger.Infow("reconciling resource group", "resourceGroup", cluster.Spec.Cloud.Azure.ResourceGroup)
		cluster, err = reconcileResourceGroup(a.ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.VNetName == "" {
		logger.Infow("reconciling vnet", "vnet", vnetName(cluster))
		cluster, err = reconcileVNet(a.ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.SubnetName == "" {
		logger.Infow("reconciling subnet", "subnet", subnetName(cluster))
		cluster, err = reconcileSubnet(a.ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.RouteTableName == "" {
		logger.Infow("reconciling route table", "routeTableName", routeTableName(cluster))
		cluster, err = reconcileRouteTable(a.ctx, clientSet, location, cluster, update)
		if err != nil {
			return nil, err
		}
	}

<<<<<<< HEAD
	if force || cluster.Spec.Cloud.Azure.SecurityGroup == "" {
		logger.Infow("reconciling security group", "securityGroup", securityGroupName(cluster))
		cluster, err = reconcileSecurityGroup(a.ctx, clientSet, location, cluster, update)
=======
	if cluster.Spec.Cloud.Azure.SecurityGroup == "" {
		cluster.Spec.Cloud.Azure.SecurityGroup = resourceNamePrefix + cluster.Name

		lowPort, highPort := kubermaticresources.NewTemplateDataBuilder().
			WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
			WithCluster(cluster).
			Build().
			NodePorts()

		nodePortsAllowedIPRange := cluster.Spec.Cloud.Azure.NodePortsAllowedIPRange
		if nodePortsAllowedIPRange == "" {
			nodePortsAllowedIPRange = "0.0.0.0/0"
		}

		logger.Infow("ensuring security group", "securityGroup", cluster.Spec.Cloud.Azure.SecurityGroup)
		if err = a.ensureSecurityGroup(cluster.Spec.Cloud, location, cluster.Name, lowPort, highPort, nodePortsAllowedIPRange, credentials); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.SecurityGroup = cluster.Spec.Cloud.Azure.SecurityGroup
			kuberneteshelper.AddFinalizer(updatedCluster, FinalizerSecurityGroup)
		})
>>>>>>> AllowedIPRange field renamed
		if err != nil {
			return nil, err
		}
	}

	if force || cluster.Spec.Cloud.Azure.AvailabilitySet == "" {
		if cluster.Spec.Cloud.Azure.AssignAvailabilitySet == nil ||
			*cluster.Spec.Cloud.Azure.AssignAvailabilitySet {
			logger.Infow("reconciling AvailabilitySet", "availabilitySet", availabilitySetName(cluster))
			cluster, err = reconcileAvailabilitySet(a.ctx, clientSet, location, cluster, update)
			if err != nil {
				return nil, err
			}
		}
	}

	return cluster, nil
}

func (a *Azure) DefaultCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	return nil
}

func (a *Azure) ValidateCloudSpec(cloud kubermaticv1.CloudSpec) error {
	credentials, err := GetCredentialsForCluster(cloud, a.secretKeySelector)
	if err != nil {
		return err
	}

	if cloud.Azure.ResourceGroup != "" {
		rgClient, err := getGroupsClient(cloud, credentials)
		if err != nil {
			return err
		}

		if _, err = rgClient.Get(a.ctx, cloud.Azure.ResourceGroup); err != nil {
			return err
		}
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}

	if cloud.Azure.VNetName != "" {
		vnetClient, err := getNetworksClient(cloud, credentials)
		if err != nil {
			return err
		}

		if _, err = vnetClient.Get(a.ctx, resourceGroup, cloud.Azure.VNetName, ""); err != nil {
			return err
		}
	}

	if cloud.Azure.SubnetName != "" {
		subnetClient, err := getSubnetsClient(cloud, credentials)
		if err != nil {
			return err
		}

		if _, err = subnetClient.Get(a.ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, ""); err != nil {
			return err
		}
	}

	if cloud.Azure.RouteTableName != "" {
		routeTablesClient, err := getRouteTablesClient(cloud, credentials)
		if err != nil {
			return err
		}

		if _, err = routeTablesClient.Get(a.ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, ""); err != nil {
			return err
		}
	}

	if cloud.Azure.SecurityGroup != "" {
		sgClient, err := getSecurityGroupsClient(cloud, credentials)
		if err != nil {
			return err
		}

		if _, err = sgClient.Get(a.ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, ""); err != nil {
			return err
		}
	}

	return nil
}

func (a *Azure) AddICMPRulesIfRequired(cluster *kubermaticv1.Cluster) error {
	credentials, err := GetCredentialsForCluster(cluster.Spec.Cloud, a.secretKeySelector)
	if err != nil {
		return err
	}

	azure := cluster.Spec.Cloud.Azure
	if azure.SecurityGroup == "" {
		return nil
	}
	sgClient, err := getSecurityGroupsClient(cluster.Spec.Cloud, credentials)
	if err != nil {
		return fmt.Errorf("failed to get security group client: %v", err)
	}
	sg, err := sgClient.Get(a.ctx, azure.ResourceGroup, azure.SecurityGroup, "")
	if err != nil {
		return fmt.Errorf("failed to get security group %q: %v", azure.SecurityGroup, err)
	}

	var hasDenyAllTCPRule, hasDenyAllUDPRule, hasICMPAllowAllRule bool
	if sg.SecurityRules != nil {
		for _, rule := range *sg.SecurityRules {
			if rule.Name == nil {
				continue
			}
			// We trust that no one will alter the content of the rules
			switch *rule.Name {
			case denyAllTCPSecGroupRuleName:
				hasDenyAllTCPRule = true
			case denyAllUDPSecGroupRuleName:
				hasDenyAllUDPRule = true
			case allowAllICMPSecGroupRuleName:
				hasICMPAllowAllRule = true
			}
		}
	}

	var newSecurityRules []network.SecurityRule
	if !hasDenyAllTCPRule {
		a.log.With("cluster", cluster.Name).Info("Creating TCP deny all rule")
		newSecurityRules = append(newSecurityRules, tcpDenyAllRule())
	}
	if !hasDenyAllUDPRule {
		a.log.With("cluster", cluster.Name).Info("Creating UDP deny all rule")
		newSecurityRules = append(newSecurityRules, udpDenyAllRule())
	}
	if !hasICMPAllowAllRule {
		a.log.With("cluster", cluster.Name).Info("Creating ICMP allow all rule")
		newSecurityRules = append(newSecurityRules, icmpAllowAllRule())
	}

	if len(newSecurityRules) > 0 {
		newSecurityGroupRules := append(*sg.SecurityRules, newSecurityRules...)
		sg.SecurityRules = &newSecurityGroupRules
		_, err := sgClient.CreateOrUpdate(a.ctx, azure.ResourceGroup, azure.SecurityGroup, sg)
		if err != nil {
			return fmt.Errorf("failed to add new rules to security group %q: %v", *sg.Name, err)
		}
	}
	return nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (a *Azure) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

type Credentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

func GetAKSCLusterConfig(ctx context.Context, tenantID, subscriptionID, clientID, clientSecret, clusterName, resourceGroupName string) (*api.Config, error) {

	var err error
	aksClient := containerservice.NewManagedClustersClient(subscriptionID)
	aksClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	credResult, err := aksClient.ListClusterAdminCredentials(ctx, resourceGroupName, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cannot get azure cluster config %w", err)
	}

	data := (*credResult.Kubeconfigs)[0].Value
	config, err := clientcmd.Load(*data)
	if err != nil {
		return nil, fmt.Errorf("cannot get azure cluster config %w", err)
	}
	return config, nil
}

func GetCredentialsForAKSCluster(cloud kubermaticv1.ExternalClusterCloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (Credentials, error) {
	tenantID := cloud.AKS.TenantID
	subscriptionID := cloud.AKS.SubscriptionID
	clientID := cloud.AKS.ClientID
	clientSecret := cloud.AKS.ClientSecret
	var err error

	if tenantID == "" {
		if cloud.AKS.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		tenantID, err = secretKeySelector(cloud.AKS.CredentialsReference, kubermaticresources.AzureTenantID)
		if err != nil {
			return Credentials{}, err
		}
	}

	if subscriptionID == "" {
		if cloud.AKS.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		subscriptionID, err = secretKeySelector(cloud.AKS.CredentialsReference, kubermaticresources.AzureSubscriptionID)
		if err != nil {
			return Credentials{}, err
		}
	}

	if clientID == "" {
		if cloud.AKS.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		clientID, err = secretKeySelector(cloud.AKS.CredentialsReference, kubermaticresources.AzureClientID)
		if err != nil {
			return Credentials{}, err
		}
	}

	if clientSecret == "" {
		if cloud.AKS.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		clientSecret, err = secretKeySelector(cloud.AKS.CredentialsReference, kubermaticresources.AzureClientSecret)
		if err != nil {
			return Credentials{}, err
		}
	}

	return Credentials{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}, nil
}
