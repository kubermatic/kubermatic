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
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/dc"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// https://docs.microsoft.com/en-us/azure/virtual-machines/sizes-gpu
var gpuInstanceFamilies = map[string]int32{"Standard_NC6": 1, "Standard_NC12": 2, "Standard_NC24": 4, "Standard_NC24r": 4,
	"Standard_NC6s_v2": 1, "Standard_NC12s_v2": 2, "Standard_NC24s_v2": 4, "Standard_NC24rs_v2": 4, "Standard_NC6s_v3": 1,
	"Standard_NC12s_v3": 2, "Standard_NC24s_v3": 4, "Standard_NC24rs_v3": 4, "Standard_NC4as_T4_v3": 1, "Standard_NC8as_T4_v3": 1,
	"Standard_NC16as_T4_v3": 1, "Standard_NC64as_T4_v3": 4, "Standard_ND6s": 1, "Standard_ND12s": 2, "Standard_ND24s": 4, "Standard_ND24rs": 4,
	"Standard_ND40rs_v2": 8, "Standard_NV6": 1, "Standard_NV12": 2, "Standard_NV24": 4, "Standard_NV12s_v3": 1, "Standard_NV24s_v3": 2, "Standard_NV48s_v3": 4,
	"Standard_NV32as_v4": 1}

var NewAzureClientSet = func(subscriptionID, clientID, clientSecret, tenantID string) (AzureClientSet, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, err
	}

	sizesClient, err := armcompute.NewVirtualMachineSizesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	skusClient, err := armcompute.NewResourceSKUsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	securityGroupsClient, err := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	resourceGroupsClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	routeTablesClient, err := armnetwork.NewRouteTablesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	subnetsClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
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
	vmSizeClient         *armcompute.VirtualMachineSizesClient
	skusClient           *armcompute.ResourceSKUsClient
	securityGroupsClient *armnetwork.SecurityGroupsClient
	routeTablesClient    *armnetwork.RouteTablesClient
	resourceGroupsClient *armresources.ResourceGroupsClient
	subnetsClient        *armnetwork.SubnetsClient
	vnetClient           *armnetwork.VirtualNetworksClient
}

type AzureClientSet interface {
	ListVMSize(ctx context.Context, location string) ([]armcompute.VirtualMachineSize, error)
	ListSKU(ctx context.Context, location string) ([]armcompute.ResourceSKU, error)
	ListSecurityGroups(ctx context.Context, resourceGroupName string) ([]armnetwork.SecurityGroup, error)
	ListResourceGroups(ctx context.Context) ([]armresources.ResourceGroup, error)
	ListRouteTables(ctx context.Context, resourceGroupName string) ([]armnetwork.RouteTable, error)
	ListVnets(ctx context.Context, resourceGroupName string) ([]armnetwork.VirtualNetwork, error)
	ListSubnets(ctx context.Context, resourceGroupName, virtualNetworkName string) ([]armnetwork.Subnet, error)
}

func (s *azureClientSetImpl) ListSKU(ctx context.Context, location string) ([]armcompute.ResourceSKU, error) {
	pager := s.skusClient.NewListPager(&armcompute.ResourceSKUsClientListOptions{
		Filter: &location,
	})

	result := []armcompute.ResourceSKU{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list SKU resource: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func (s *azureClientSetImpl) ListVMSize(ctx context.Context, location string) ([]armcompute.VirtualMachineSize, error) {
	pager := s.vmSizeClient.NewListPager(location, nil)

	result := []armcompute.VirtualMachineSize{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list sizes: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func (s *azureClientSetImpl) ListSecurityGroups(ctx context.Context, resourceGroupName string) ([]armnetwork.SecurityGroup, error) {
	pager := s.securityGroupsClient.NewListPager(resourceGroupName, nil)

	result := []armnetwork.SecurityGroup{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list security groups: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func (s *azureClientSetImpl) ListRouteTables(ctx context.Context, resourceGroupName string) ([]armnetwork.RouteTable, error) {
	pager := s.routeTablesClient.NewListPager(resourceGroupName, nil)

	result := []armnetwork.RouteTable{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list route tables: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func (s *azureClientSetImpl) ListResourceGroups(ctx context.Context) ([]armresources.ResourceGroup, error) {
	pager := s.resourceGroupsClient.NewListPager(nil)

	result := []armresources.ResourceGroup{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list resource groups: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func (s *azureClientSetImpl) ListSubnets(ctx context.Context, resourceGroupName, virtualNetworkName string) ([]armnetwork.Subnet, error) {
	pager := s.subnetsClient.NewListPager(resourceGroupName, virtualNetworkName, nil)

	result := []armnetwork.Subnet{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list subnets: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func (s *azureClientSetImpl) ListVnets(ctx context.Context, resourceGroupName string) ([]armnetwork.VirtualNetwork, error) {
	pager := s.vnetClient.NewListPager(resourceGroupName, nil)

	result := []armnetwork.VirtualNetwork{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list vnets: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func AzureSizeWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Azure == nil {
		return nil, utilerrors.NewNotFound("cloud spec for ", clusterID)
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	datacenter, err := dc.GetDatacenter(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
	}

	if datacenter.Spec.Azure == nil {
		return nil, utilerrors.NewNotFound("cloud spec (dc) for ", clusterID)
	}

	azureLocation := datacenter.Spec.Azure.Location
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	creds, err := azure.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	settings, err := settingsProvider.GetGlobalSettings(ctx)
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
		return nil, utilerrors.NewNotFound("cloud spec for ", clusterID)
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	datacenter, err := dc.GetDatacenter(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
	}

	if datacenter.Spec.Azure == nil {
		return nil, utilerrors.NewNotFound("cloud spec (dc) for ", clusterID)
	}

	azureLocation := datacenter.Spec.Azure.Location
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	creds, err := azure.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	return AzureSKUAvailabilityZones(ctx, creds.SubscriptionID, creds.ClientID, creds.ClientSecret, creds.TenantID, azureLocation, skuName)
}

func isVirtualMachinesType(sku armcompute.ResourceSKU) bool {
	resourceType := sku.ResourceType

	return resourceType != nil && *resourceType == "virtualMachines"
}

func isLocation(sku armcompute.ResourceSKU, location string) bool {
	if sku.Locations != nil {
		for _, l := range sku.Locations {
			if *l == location {
				return true
			}
		}
	}
	return false
}

// isValidVM checks all constrains for VM.
func isValidVM(sku armcompute.ResourceSKU, location string) bool {
	if !isLocation(sku, location) {
		return false
	}

	if !isVirtualMachinesType(sku) {
		return false
	}

	// check restricted locations
	if restrictions := sku.Restrictions; restrictions != nil {
		for _, r := range restrictions {
			if info := r.RestrictionInfo; info != nil && info.Locations != nil {
				for _, l := range info.Locations {
					if *l == location {
						return false
					}
				}
			}
		}
	}

	return true
}

func GetAzureVMSize(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, vmName string) (*apiv1.AzureSize, error) {
	sizesClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %w", err)
	}

	// get all available VM size types for given location
	listVMSize, err := sizesClient.ListVMSize(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %w", err)
	}

	for _, vm := range listVMSize {
		if strings.EqualFold(*vm.Name, vmName) {
			return &apiv1.AzureSize{
				NumberOfCores:        *vm.NumberOfCores,
				ResourceDiskSizeInMB: *vm.ResourceDiskSizeInMB,
				MemoryInMB:           *vm.MemoryInMB,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not find Azure VM Size named %q", vmName)
}

func AzureSize(ctx context.Context, quota kubermaticv1.MachineDeploymentVMResourceQuota, subscriptionID, clientID, clientSecret, tenantID, location string) (apiv1.AzureSizeList, error) {
	sizesClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %w", err)
	}

	skuList, err := sizesClient.ListSKU(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list SKU resource: %w", err)
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
		return nil, fmt.Errorf("failed to list sizes: %w", err)
	}

	var sizeList apiv1.AzureSizeList
	for _, v := range listVMSize {
		if v.Name != nil {
			vmName := *v.Name

			if _, okSKU := validSKUSet[vmName]; okSKU {
				s := apiv1.AzureSize{
					Name:          vmName,
					NumberOfCores: *v.NumberOfCores,
					// TODO: Use this to validate user-defined disk size.
					OsDiskSizeInMB:       *v.OSDiskSizeInMB,
					ResourceDiskSizeInMB: *v.ResourceDiskSizeInMB,
					MemoryInMB:           *v.MemoryInMB,
					MaxDataDiskCount:     *v.MaxDataDiskCount,
				}

				if gpus, okGPU := gpuInstanceFamilies[vmName]; okGPU {
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
		return nil, fmt.Errorf("failed to create authorizer for sku client: %w", err)
	}

	skuList, err := azSKUClient.ListSKU(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sku resource: %w", err)
	}

	var azZones = &apiv1.AzureAvailabilityZonesList{}
	for _, sku := range skuList {
		if skuName == *sku.Name {
			for _, l := range sku.LocationInfo {
				if location == *l.Location && l.Zones != nil && len(l.Zones) > 0 {
					azZones.Zones = []string{}
					for _, z := range l.Zones {
						azZones.Zones = append(azZones.Zones, *z)
					}

					return azZones, nil
				}
			}
		}
	}

	return nil, nil
}

func AzureSecurityGroupEndpoint(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, resourceGroup string) (*apiv1.AzureSecurityGroupsList, error) {
	securityGroupsClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for security groups client: %w", err)
	}

	securityGroupList, err := securityGroupsClient.ListSecurityGroups(ctx, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to list security group resources: %w", err)
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
		return nil, fmt.Errorf("failed to create authorizer for security groups client: %w", err)
	}

	resourceGroupList, err := securityGroupsClient.ListResourceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list security group resources: %w", err)
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
		return nil, fmt.Errorf("failed to create authorizer for security groups client: %w", err)
	}

	routeTableList, err := routeTableClient.ListRouteTables(ctx, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to list route table resources: %w", err)
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
		return nil, fmt.Errorf("failed to create authorizer for virtual network client: %w", err)
	}

	vnetList, err := vnetClient.ListVnets(ctx, resourceGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual network resources: %w", err)
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
		return nil, fmt.Errorf("failed to create authorizer for subnet client: %w", err)
	}

	subnetList, err := subnetClient.ListSubnets(ctx, resourceGroup, virtualNetwork)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual network resources: %w", err)
	}

	subnets := &apiv1.AzureSubnetsList{}
	for _, sb := range subnetList {
		subnets.Subnets = append(subnets.Subnets, *sb.Name)
	}

	return subnets, nil
}
