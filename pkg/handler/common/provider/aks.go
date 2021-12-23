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

	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerservice/mgmt/containerservice"
	"github.com/Azure/go-autorest/autorest/azure/auth"
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
	var err error
	clusters := apiv2.AKSClusterList{}

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterList, err := clusterProvider.List(project)
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
	aksClient := containerservice.NewManagedClustersClient(cred.SubscriptionID)
	aksClient.Authorizer, err = auth.NewClientCredentialsConfig(cred.ClientID, cred.ClientSecret, cred.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}
	clusterListResult, err := aksClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list AKS clusters: %v", err)
	}

	for _, f := range clusterListResult.Values() {
		var imported bool
		resourceGroup := strings.Split(strings.SplitAfter(*f.ID, "resourcegroups/")[1], "/")[0]
		if clusterSet, ok := aksExternalCluster[resourceGroup]; ok {
			if clusterSet.Has(*f.Name) {
				imported = true
			}
		}
		clusters = append(clusters, apiv2.AKSCluster{Name: *f.Name, ResourceGroup: resourceGroup, IsImported: imported})
	}
	return clusters, nil
}

func ListAKSUpgrades(ctx context.Context, cred resources.AKSCredentials, resourceGroupName, resourceName string) ([]*apiv1.MasterVersion, error) {
	var err error
	upgrades := make([]*apiv1.MasterVersion, 0)

	aksClient := containerservice.NewManagedClustersClient(cred.SubscriptionID)
	aksClient.Authorizer, err = auth.NewClientCredentialsConfig(cred.ClientID, cred.ClientSecret, cred.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %v", err.Error())
	}
	clusterUpgradeProfile, err := aksClient.GetUpgradeProfile(ctx, resourceGroupName, resourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade cluster: %v", err.Error())
	}
	upgradesItems := *clusterUpgradeProfile.ManagedClusterUpgradeProfileProperties.ControlPlaneProfile.Upgrades
	for _, upgradesItem := range upgradesItems {
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
	var err error
	upgrades := make([]*apiv1.MasterVersion, 0)
	agentPoolClient := containerservice.NewAgentPoolsClient(cred.SubscriptionID)
	agentPoolClient.Authorizer, err = auth.NewClientCredentialsConfig(cred.ClientID, cred.ClientSecret, cred.TenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	profile, err := agentPoolClient.GetUpgradeProfile(ctx, resourceGroupName, clusterName, machineDeployment)
	if err != nil {
		return nil, err
	}
	if profile.AgentPoolUpgradeProfileProperties.Upgrades == nil {
		return nil, nil
	}
	poolUpgrades := *profile.AgentPoolUpgradeProfileProperties.Upgrades
	for _, poolUpgrade := range poolUpgrades {
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
	var err error

	aksClient := containerservice.NewManagedClustersClient(cred.SubscriptionID)
	aksClient.Authorizer, err = auth.NewClientCredentialsConfig(cred.ClientID, cred.ClientSecret, cred.TenantID).Authorizer()
	if err != nil {
		return fmt.Errorf("failed to create authorizer: %s", err.Error())
	}
	_, err = aksClient.List(ctx)

	return err
}
