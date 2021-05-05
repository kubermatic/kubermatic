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

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-12-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/dc"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// https://docs.microsoft.com/en-us/azure/virtual-machines/sizes-gpu
var gpuInstanceFamilies = map[string]int32{"Standard_NC6": 1, "Standard_NC12": 2, "Standard_NC24": 4, "Standard_NC24r": 4,
	"Standard_NC6s_v2": 1, "Standard_NC12s_v2": 2, "Standard_NC24s_v2": 4, "Standard_NC24rs_v2": 4, "Standard_NC6s_v3": 1,
	"Standard_NC12s_v3": 2, "Standard_NC24s_v3": 4, "Standard_NC24rs_v3": 4, "Standard_NC4as_T4_v3": 1, "Standard_NC8as_T4_v3": 1,
	"Standard_NC16as_T4_v3": 1, "Standard_NC64as_T4_v3": 4, "Standard_ND6s": 1, "Standard_ND12s": 2, "Standard_ND24s": 4, "Standard_ND24rs": 4,
	"Standard_ND40rs_v2": 8, "Standard_NV6": 1, "Standard_NV12": 2, "Standard_NV24": 4, "Standard_NV12s_v3": 1, "Standard_NV24s_v3": 2, "Standard_NV48s_v3": 4,
	"Standard_NV32as_v4": 1}

var NewAzureClientSet = func(subscriptionID, clientID, clientSecret, tenantID string) (AzureClientSet, error) {
	var err error
	sizesClient := compute.NewVirtualMachineSizesClient(subscriptionID)
	sizesClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	skusClient := compute.NewResourceSkusClient(subscriptionID)
	skusClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	securityGroupsClient := network.NewSecurityGroupsClient(subscriptionID)
	securityGroupsClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	resourceGroupsClient := resources.NewGroupsClient(subscriptionID)
	resourceGroupsClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	routeTablesClient := network.NewRouteTablesClient(subscriptionID)
	routeTablesClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	subnetsClient := network.NewSubnetsClient(subscriptionID)
	subnetsClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	vnetClient := network.NewVirtualNetworksClient(subscriptionID)
	vnetClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}

	return &azureClientSetImpl{
		vmSizeClient:         sizesClient,
		skusClient:           skusClient,
		securityGroupsClient: securityGroupsClient,
		resourceGroupsClient: resourceGroupsClient,
		subnetsClient:        subnetsClient,
		vnetClient:           vnetClient,
		routeTablesClient:    routeTablesClient,
	}, nil
}

type azureClientSetImpl struct {
	vmSizeClient         compute.VirtualMachineSizesClient
	skusClient           compute.ResourceSkusClient
	securityGroupsClient network.SecurityGroupsClient
	routeTablesClient    network.RouteTablesClient
	resourceGroupsClient resources.GroupsClient
	subnetsClient        network.SubnetsClient
	vnetClient           network.VirtualNetworksClient
}

type AzureClientSet interface {
	ListVMSize(ctx context.Context, location string) ([]compute.VirtualMachineSize, error)
	ListSKU(ctx context.Context, location string) ([]compute.ResourceSku, error)
	ListSecurityGroups(ctx context.Context, resourceGroupName string) ([]network.SecurityGroup, error)
	ListResourceGroups(ctx context.Context) ([]resources.Group, error)
	ListRouteTables(ctx context.Context, resourceGroupName string) ([]network.RouteTable, error)
	ListVnets(ctx context.Context, resourceGroupName string) ([]network.VirtualNetwork, error)
	ListSubnets(ctx context.Context, resourceGroupName, virtualNetworkName string) ([]network.Subnet, error)
}

func (s *azureClientSetImpl) ListSKU(ctx context.Context, location string) ([]compute.ResourceSku, error) {
	skuList, err := s.skusClient.List(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list SKU resource: %v", err)
	}
	return skuList.Values(), nil
}

func (s *azureClientSetImpl) ListVMSize(ctx context.Context, location string) ([]compute.VirtualMachineSize, error) {
	sizesResult, err := s.vmSizeClient.List(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %v", err)
	}
	return *sizesResult.Value, nil
}

func (s *azureClientSetImpl) ListSecurityGroups(ctx context.Context, resourceGroupName string) ([]network.SecurityGroup, error) {
	securityGroups, err := s.securityGroupsClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %v", err)
	}
	return securityGroups.Values(), nil
}

func (s *azureClientSetImpl) ListRouteTables(ctx context.Context, resourceGroupName string) ([]network.RouteTable, error) {
	routeTables, err := s.routeTablesClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource groups: %v", err)
	}
	return routeTables.Values(), nil

}

func (s *azureClientSetImpl) ListResourceGroups(ctx context.Context) ([]resources.Group, error) {
	resourceGroups, err := s.resourceGroupsClient.List(ctx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource groups: %v", err)
	}
	return resourceGroups.Values(), nil

}

func (s *azureClientSetImpl) ListSubnets(ctx context.Context, resourceGroupName, virtualNetworkName string) ([]network.Subnet, error) {
	subnets, err := s.subnetsClient.List(ctx, resourceGroupName, virtualNetworkName)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %v", err)
	}
	return subnets.Values(), nil

}

func (s *azureClientSetImpl) ListVnets(ctx context.Context, resourceGroupName string) ([]network.VirtualNetwork, error) {
	vnets, err := s.vnetClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("failed to list vnets: %v", err)
	}
	return vnets.Values(), nil

}

func AzureSizeWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Azure == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	datacenter, err := dc.GetDatacenter(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}

	if datacenter.Spec.Azure == nil {
		return nil, errors.NewNotFound("cloud spec (dc) for ", clusterID)
	}

	azureLocation := datacenter.Spec.Azure.Location
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	creds, err := azure.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	settings, err := settingsProvider.GetGlobalSettings()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return AzureSize(ctx, settings.Spec.MachineDeploymentVMResourceQuota, creds.SubscriptionID, creds.ClientID, creds.ClientSecret, creds.TenantID, azureLocation)
}

func AzureAvailabilityZonesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID, skuName string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Azure == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	datacenter, err := dc.GetDatacenter(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, err.Error())
	}

	if datacenter.Spec.Azure == nil {
		return nil, errors.NewNotFound("cloud spec (dc) for ", clusterID)
	}

	azureLocation := datacenter.Spec.Azure.Location
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	creds, err := azure.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	return AzureSKUAvailabilityZones(ctx, creds.SubscriptionID, creds.ClientID, creds.ClientSecret, creds.TenantID, azureLocation, skuName)
}

func isVirtualMachinesType(sku compute.ResourceSku) bool {
	resourceType := sku.ResourceType
	if resourceType != nil {
		if *resourceType == "virtualMachines" {
			return true
		}
	}
	return false
}

func isLocation(sku compute.ResourceSku, location string) bool {
	if sku.Locations != nil {
		for _, l := range *sku.Locations {
			if l == location {
				return true
			}
		}
	}
	return false
}

// isValidVM checks all constrains for VM
func isValidVM(sku compute.ResourceSku, location string) bool {

	if !isLocation(sku, location) {
		return false
	}

	if !isVirtualMachinesType(sku) {
		return false
	}

	// check restricted locations
	restrictions := sku.Restrictions
	if restrictions != nil {
		for _, r := range *restrictions {
			restrictionInfo := r.RestrictionInfo
			if restrictionInfo != nil {
				if restrictionInfo.Locations != nil {
					for _, l := range *restrictionInfo.Locations {
						if l == location {
							return false
						}
					}
				}
			}
		}
	}

	return true
}

func AzureSize(ctx context.Context, quota kubermaticv1.MachineDeploymentVMResourceQuota, subscriptionID, clientID, clientSecret, tenantID, location string) (apiv1.AzureSizeList, error) {
	sizesClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %v", err)
	}

	skuList, err := sizesClient.ListSKU(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list SKU resource: %v", err)
	}

	// prepare set of valid VM size types from SKU resources
	validSKUSet := make(map[string]struct{}, len(skuList))
	for _, v := range skuList {
		if isValidVM(v, location) {
			validSKUSet[*v.Name] = struct{}{}
		}
	}

	// get all available VM size types for given location
	listVMSize, err := sizesClient.ListVMSize(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %v", err)
	}

	var sizeList apiv1.AzureSizeList
	for _, v := range listVMSize {
		if v.Name != nil {
			vmName := *v.Name
			_, okSKU := validSKUSet[vmName]
			gpus, okGPU := gpuInstanceFamilies[vmName]
			if okSKU {
				s := apiv1.AzureSize{
					Name:          vmName,
					NumberOfCores: *v.NumberOfCores,
					// TODO: Use this to validate user-defined disk size.
					OsDiskSizeInMB:       *v.OsDiskSizeInMB,
					ResourceDiskSizeInMB: *v.ResourceDiskSizeInMB,
					MemoryInMB:           *v.MemoryInMB,
					MaxDataDiskCount:     *v.MaxDataDiskCount,
				}
				if okGPU {
					s.NumberOfGPUs = gpus
				}
				sizeList = append(sizeList, s)
			}
		}
	}

	return filterAzureByQuota(sizeList, quota), nil
}

func filterAzureByQuota(instances apiv1.AzureSizeList, quota kubermaticv1.MachineDeploymentVMResourceQuota) apiv1.AzureSizeList {
	filteredRecords := apiv1.AzureSizeList{}

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, r := range instances {
		keep := true

		if !handlercommon.FilterGPU(int(r.NumberOfGPUs), quota.EnableGPU) {
			keep = false
		}

		if !handlercommon.FilterCPU(int(r.NumberOfCores), quota.MinCPU, quota.MaxCPU) {
			keep = false
		}
		if !handlercommon.FilterMemory(int(r.MemoryInMB/1024), quota.MinRAM, quota.MaxRAM) {
			keep = false
		}

		if keep {
			filteredRecords = append(filteredRecords, r)
		}
	}

	return filteredRecords
}

func AzureSKUAvailabilityZones(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, skuName string) (*apiv1.AzureAvailabilityZonesList, error) {
	azSKUClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for sku client: %v", err)
	}

	skuList, err := azSKUClient.ListSKU(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sku resource: %v", err)
	}

	var azZones = &apiv1.AzureAvailabilityZonesList{}
	for _, sku := range skuList {
		if skuName == *sku.Name {
			for _, l := range *sku.LocationInfo {
				if location == *l.Location {
					if *l.Zones != nil && len(*l.Zones) > 0 {
						azZones.Zones = *l.Zones
						return azZones, nil
					}
				}
			}
		}
	}

	return nil, nil
}

func AzureSecurityGroupEndpoint(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, resourceGroup string) (*apiv1.AzureSecurityGroupsList, error) {
	securityGroupsClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for security groups client: %v", err)
	}

	securityGroupList, err := securityGroupsClient.ListSecurityGroups(ctx, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to list security group resources: %v", err)
	}

	apiSecurityGroups := &apiv1.AzureSecurityGroupsList{}
	for _, sg := range securityGroupList {
		if location == *sg.Location {
			apiSecurityGroups.SecurityGroups = append(apiSecurityGroups.SecurityGroups, *sg.Name)
		}
	}

	return apiSecurityGroups, nil
}

func AzureResourceGroupEndpoint(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location string) (*apiv1.AzureResourceGroupsList, error) {
	securityGroupsClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for security groups client: %v", err)
	}

	resourceGroupList, err := securityGroupsClient.ListResourceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list security group resources: %v", err)
	}

	apiResourceGroups := &apiv1.AzureResourceGroupsList{}
	for _, rg := range resourceGroupList {
		if location == *rg.Location {
			apiResourceGroups.ResourceGroups = append(apiResourceGroups.ResourceGroups, *rg.Name)
		}
	}

	return apiResourceGroups, nil
}

func AzureRouteTableEndpoint(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, resourceGroup string) (*apiv1.AzureRouteTablesList, error) {
	routeTableClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for security groups client: %v", err)
	}

	routeTableList, err := routeTableClient.ListRouteTables(ctx, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to list route table resources: %v", err)
	}

	apiRouteTables := &apiv1.AzureRouteTablesList{}
	for _, rt := range routeTableList {
		if location == *rt.Location {
			apiRouteTables.RouteTables = append(apiRouteTables.RouteTables, *rt.Name)
		}
	}

	return apiRouteTables, nil
}

func AzureVnetEndpoint(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, resourceGroup string) (*apiv1.AzureVirtualNetworksList, error) {
	vnetClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for virtual network client: %v", err)
	}

	vnetList, err := vnetClient.ListVnets(ctx, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual network resources: %v", err)
	}

	vnets := &apiv1.AzureVirtualNetworksList{}
	for _, vn := range vnetList {
		if location == *vn.Location {
			vnets.VirtualNetworks = append(vnets.VirtualNetworks, *vn.Name)
		}
	}

	return vnets, nil
}

func AzureSubnetEndpoint(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, resourceGroup, virtualNetwork string) (*apiv1.AzureSubnetsList, error) {
	subnetClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for subnet client: %v", err)
	}

	subnetList, err := subnetClient.ListSubnets(ctx, resourceGroup, virtualNetwork)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual network resources: %v", err)
	}

	subnets := &apiv1.AzureSubnetsList{}
	for _, sb := range subnetList {
		subnets.Subnets = append(subnets.Subnets, *sb.Name)
	}

	return subnets, nil
}
