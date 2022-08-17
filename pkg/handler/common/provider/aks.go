/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
	"github.com/Azure/go-autorest/autorest/to"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aks"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/util/sets"
)

const MinimumVMCores = 2

func ListAKSClusters(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, cred resources.AKSCredentials, projectID string) (apiv2.AKSClusterList, error) {
	clusters := apiv2.AKSClusterList{}

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterList, err := clusterProvider.List(ctx, project)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	aksExternalCluster := make(map[string]sets.String)
	for _, externalCluster := range clusterList.Items {
		cloud := externalCluster.Spec.CloudSpec
		if cloud.AKS != nil {
			resourceGroup := cloud.AKS.ResourceGroup
			if _, ok := aksExternalCluster[resourceGroup]; !ok {
				aksExternalCluster[resourceGroup] = make(sets.String)
			}
			aksExternalCluster[resourceGroup] = aksExternalCluster[resourceGroup].Insert(cloud.AKS.Name)
		}
	}

	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	aksClient, err := armcontainerservice.NewManagedClustersClient(cred.SubscriptionID, azcred, nil)
	if err != nil {
		return nil, aks.DecodeError(err)
	}

	pager := aksClient.NewListPager(nil)

	result := []armcontainerservice.ManagedCluster{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, aks.DecodeError(err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}
	for _, cluster := range result {
		if cluster.ID == nil || cluster.Name == nil {
			continue
		}

		var imported bool
		resourceGroup := strings.Split(strings.SplitAfter(*cluster.ID, "resourcegroups/")[1], "/")[0]
		if clusterSet, ok := aksExternalCluster[resourceGroup]; ok {
			if clusterSet.Has(*cluster.Name) {
				imported = true
			}
		}
		clusters = append(clusters, apiv2.AKSCluster{Name: *cluster.Name, ResourceGroup: resourceGroup, IsImported: imported})
	}
	return clusters, nil
}

func ListAKSVMSizes(ctx context.Context, cred resources.AKSCredentials, location string) (apiv2.AKSVMSizeList, error) {
	vmSizes, err := AKSAzureSize(ctx, cred, location)
	if err != nil {
		return nil, fmt.Errorf("couldn't get vmsizes: %w", err)
	}

	return vmSizes, nil
}

func AKSAzureSize(ctx context.Context, cred resources.AKSCredentials, location string) (apiv2.AKSVMSizeList, error) {
	sizesClient, err := NewAzureClientSet(cred.SubscriptionID, cred.ClientID, cred.ClientSecret, cred.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %w", err)
	}

	skuList, err := sizesClient.ListSKU(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list SKU resource: %w", err)
	}

	// prepare set of valid VM size types from SKU resources
	validSKUSet := make(map[string]struct{}, len(skuList))
	for _, sku := range skuList {
		if isValidVM(sku, location) {
			validSKUSet[*sku.Name] = struct{}{}
		}
	}

	// get all available VM size types for given location
	listVMSize, err := sizesClient.ListVMSize(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list vmsizes: %w", err)
	}

	var sizeList apiv2.AKSVMSizeList
	for _, vm := range listVMSize {
		// VM sizes with less than 2 CPUs may not be used with AKS.
		if vm.Name != nil && vm.NumberOfCores != nil {
			if *vm.NumberOfCores >= MinimumVMCores {
				vmName := *vm.Name

				if _, okSKU := validSKUSet[vmName]; okSKU {
					s := apiv2.AKSVMSize{
						Name:                 vmName,
						NumberOfCores:        to.Int32(vm.NumberOfCores),
						MemoryInMB:           to.Int32(vm.MemoryInMB),
						MaxDataDiskCount:     to.Int32(vm.MaxDataDiskCount),
						OsDiskSizeInMB:       to.Int32(vm.OSDiskSizeInMB),
						ResourceDiskSizeInMB: to.Int32(vm.ResourceDiskSizeInMB),
					}
					if gpus, okGPU := gpuInstanceFamilies[vmName]; okGPU {
						s.NumberOfGPUs = gpus
					}

					sizeList = append(sizeList, s)
				}
			}
		}
	}

	return sizeList, nil
}
