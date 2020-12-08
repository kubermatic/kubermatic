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

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/dc"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

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

	return &azureClientSetImpl{
		vmSizeClient: sizesClient,
		skusClient:   skusClient,
	}, nil
}

type azureClientSetImpl struct {
	vmSizeClient compute.VirtualMachineSizesClient
	skusClient   compute.ResourceSkusClient
}

type AzureClientSet interface {
	ListVMSize(ctx context.Context, location string) ([]compute.VirtualMachineSize, error)
	ListSKU(ctx context.Context, location string) ([]compute.ResourceSku, error)
}

func (s *azureClientSetImpl) ListSKU(ctx context.Context, _ string) ([]compute.ResourceSku, error) {
	skuList, err := s.skusClient.List(ctx)
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

func AzureSizeWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
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
	return AzureSize(ctx, creds.SubscriptionID, creds.ClientID, creds.ClientSecret, creds.TenantID, azureLocation)
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

func AzureSize(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location string) (apiv1.AzureSizeList, error) {
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

	// prepare set of valid VM size types for container purpose
	validVMSizeList := compute.PossibleContainerServiceVMSizeTypesValues()
	validVMContainerSet := make(map[string]struct{}, len(validVMSizeList))
	for _, s := range validVMSizeList {
		validVMContainerSet[string(s)] = struct{}{}
	}

	// get all available VM size types for given location
	listVMSize, err := sizesClient.ListVMSize(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %v", err)
	}

	var sizeList apiv1.AzureSizeList
	for _, v := range listVMSize {
		if v.Name != nil {
			_, okSKU := validSKUSet[*v.Name]
			_, okVMContainer := validVMContainerSet[*v.Name]

			if okSKU && okVMContainer {
				s := apiv1.AzureSize{
					Name:          *v.Name,
					NumberOfCores: *v.NumberOfCores,
					// TODO: Use this to validate user-defined disk size.
					OsDiskSizeInMB:       *v.OsDiskSizeInMB,
					ResourceDiskSizeInMB: *v.ResourceDiskSizeInMB,
					MemoryInMB:           *v.MemoryInMB,
					MaxDataDiskCount:     *v.MaxDataDiskCount,
				}
				sizeList = append(sizeList, s)
			}
		}
	}

	return sizeList, nil
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
