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
	semverlib "github.com/Masterminds/semver/v3"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	"k8s.io/apimachinery/pkg/util/sets"
)

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
		if cloud != nil && cloud.AKS != nil {
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
		return nil, err
	}

	pager := aksClient.NewListPager(nil)

	result := []armcontainerservice.ManagedCluster{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list AKS clusters: %w", err)
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

func ListAKSUpgrades(ctx context.Context, cred resources.AKSCredentials, resourceGroupName, resourceName string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)

	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	aksClient, err := armcontainerservice.NewManagedClustersClient(cred.SubscriptionID, azcred, nil)
	if err != nil {
		return nil, err
	}

	clusterUpgradeProfile, err := aksClient.GetUpgradeProfile(ctx, resourceGroupName, resourceName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade cluster: %w", err)
	}

	upgradeProperties := clusterUpgradeProfile.Properties
	if upgradeProperties == nil || upgradeProperties.ControlPlaneProfile == nil || upgradeProperties.ControlPlaneProfile.Upgrades == nil {
		return upgrades, nil
	}

	for _, upgradesItem := range upgradeProperties.ControlPlaneProfile.Upgrades {
		v, err := ksemver.NewSemver(*upgradesItem.KubernetesVersion)
		if err != nil {
			return nil, err
		}
		upgrades = append(upgrades, &apiv1.MasterVersion{
			Version: v.Semver(),
		})
	}
	return upgrades, nil
}

func ListAKSMachineDeploymentUpgrades(ctx context.Context, cred resources.AKSCredentials, clusterName, resourceGroupName, machineDeployment string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)

	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	agentPoolClient, err := armcontainerservice.NewAgentPoolsClient(cred.SubscriptionID, azcred, nil)
	if err != nil {
		return nil, err
	}

	profile, err := agentPoolClient.GetUpgradeProfile(ctx, resourceGroupName, clusterName, machineDeployment, nil)
	if err != nil {
		return nil, err
	}

	poolUpgradeProperties := profile.Properties
	if poolUpgradeProperties == nil || poolUpgradeProperties.Upgrades == nil {
		return upgrades, nil
	}

	for _, poolUpgrade := range poolUpgradeProperties.Upgrades {
		if poolUpgrade.KubernetesVersion != nil {
			upgradeMachineDeploymentVer, err := semverlib.NewVersion(*poolUpgrade.KubernetesVersion)
			if err != nil {
				return nil, err
			}
			upgrades = append(upgrades, &apiv1.MasterVersion{Version: upgradeMachineDeploymentVer})
		}
	}

	return upgrades, nil
}

func ValidateAKSCredentials(ctx context.Context, cred resources.AKSCredentials) error {
	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return err
	}

	aksClient, err := armcontainerservice.NewManagedClustersClient(cred.SubscriptionID, azcred, nil)
	if err != nil {
		return err
	}

	_, err = aksClient.NewListPager(nil).NextPage(ctx)

	return err
}

func ListAKSVMSizes(ctx context.Context, cred resources.AKSCredentials, location string) (apiv2.AKSVMSizeList, error) {
	vmSizes, err := AKSAzureSize(ctx, cred.SubscriptionID, cred.ClientID, cred.ClientSecret, cred.TenantID, location)
	if err != nil {
		return nil, fmt.Errorf("couldn't get vmsizes: %w", err)
	}

	return vmSizes, nil
}

func AKSAzureSize(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location string) (apiv2.AKSVMSizeList, error) {
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

	var sizeList apiv2.AKSVMSizeList
	for _, v := range listVMSize {
		if v.Name != nil {
			vmName := *v.Name
			_, okSKU := validSKUSet[vmName]
			if okSKU {
				sizeList = append(sizeList, apiv2.AKSVMSize(vmName))
			}
		}
	}

	return sizeList, nil
}
