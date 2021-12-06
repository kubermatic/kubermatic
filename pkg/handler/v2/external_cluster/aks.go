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

package externalcluster

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerservice/mgmt/containerservice"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

const AKSNodepoolNameLabel = "kubernetes.azure.com/agentpool"

func createAKSCluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticapiv1.ExternalCluster, error) {
	if cloud.AKS.Name == "" || cloud.AKS.TenantID == "" || cloud.AKS.SubscriptionID == "" || cloud.AKS.ClientID == "" || cloud.AKS.ClientSecret == "" || cloud.AKS.ResourceGroup == "" {
		return nil, errors.NewBadRequest("the AKS cluster name, tenant id or subscription id or client id or client secret or resource group can not be empty")
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticapiv1.ExternalClusterCloudSpec{
		AKS: &kubermaticapiv1.ExternalClusterAKSCloudSpec{
			Name:          cloud.AKS.Name,
			ResourceGroup: cloud.AKS.ResourceGroup,
		},
	}
	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.CredentialsSecretsCleanupFinalizer)
	newCluster.Spec.CloudSpec.AKS.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchAKSCluster(ctx context.Context, old, new *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticapiv1.ExternalClusterCloudSpec) (*apiv2.ExternalCluster, error) {
	clusterName := cloud.AKS.Name
	resourceGroup := cloud.AKS.ResourceGroup

	newVersion := new.Spec.Version.Semver().String()

	cred, err := azure.GetCredentialsForAKSCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := getAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := getAKSCluster(ctx, aksClient, cloud)
	if err != nil {
		return nil, err
	}

	location := aksCluster.Location

	updateCluster := containerservice.ManagedCluster{
		Location: location,
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			KubernetesVersion: &newVersion,
		},
	}
	_, err = aksClient.CreateOrUpdate(ctx, resourceGroup, clusterName, updateCluster)
	if err != nil {
		return nil, err
	}

	return new, nil
}

func getAKSNodePools(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	cloud := cluster.Spec.CloudSpec

	cred, err := azure.GetCredentialsForAKSCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := getAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := getAKSCluster(ctx, aksClient, cloud)
	if err != nil {
		return nil, err
	}

	poolProfiles := *aksCluster.ManagedClusterProperties.AgentPoolProfiles

	return getAKSMachineDeployments(poolProfiles, cluster, clusterProvider)
}

func getAKSMachineDeployments(poolProfiles []containerservice.ManagedClusterAgentPoolProfile, cluster *kubermaticapiv1.ExternalCluster, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	machineDeployments := make([]apiv2.ExternalClusterMachineDeployment, 0, len(poolProfiles))

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, poolProfile := range poolProfiles {
		var readyReplicas int32
		for _, n := range nodes.Items {
			if n.Labels != nil {
				if n.Labels[AKSNodepoolNameLabel] == *poolProfile.Name {
					readyReplicas++
				}
			}
		}
		machineDeployments = append(machineDeployments, createMachineDeploymentFromAKSNodePoll(poolProfile, readyReplicas))
	}

	return machineDeployments, nil
}

func getAKSNodePool(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, nodePoolName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	cloud := cluster.Spec.CloudSpec

	cred, err := azure.GetCredentialsForAKSCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := getAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := getAKSCluster(ctx, aksClient, cloud)
	if err != nil {
		return nil, err
	}

	var poolProfile containerservice.ManagedClusterAgentPoolProfile
	var flag bool = false
	for _, agentPoolProperty := range *aksCluster.ManagedClusterProperties.AgentPoolProfiles {
		if *agentPoolProperty.Name == nodePoolName {
			poolProfile = agentPoolProperty
			flag = true
			break
		}
	}
	if !flag {
		return nil, fmt.Errorf("no nodePool found with the name: %v", nodePoolName)
	}

	return getAKSMachineDeployment(poolProfile, cluster, clusterProvider)
}

func getAKSMachineDeployment(poolProfile containerservice.ManagedClusterAgentPoolProfile, cluster *kubermaticapiv1.ExternalCluster, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var readyReplicas int32
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[AKSNodepoolNameLabel] == *poolProfile.Name {
				readyReplicas++
			}
		}
	}
	md := createMachineDeploymentFromAKSNodePoll(poolProfile, readyReplicas)
	return &md, nil
}

func createMachineDeploymentFromAKSNodePoll(nodePool containerservice.ManagedClusterAgentPoolProfile, readyReplicas int32) apiv2.ExternalClusterMachineDeployment {
	return apiv2.ExternalClusterMachineDeployment{
		NodeDeployment: apiv1.NodeDeployment{
			ObjectMeta: apiv1.ObjectMeta{
				ID:   *nodePool.Name,
				Name: *nodePool.Name,
			},
			Spec: apiv1.NodeDeploymentSpec{
				Replicas: *nodePool.Count,
				Template: apiv1.NodeSpec{
					Versions: apiv1.NodeVersionInfo{
						Kubelet: *nodePool.OrchestratorVersion,
					},
				},
			},
			Status: clusterv1alpha1.MachineDeploymentStatus{
				Replicas:      *nodePool.Count,
				ReadyReplicas: readyReplicas,
			},
		},
	}
}

func getAKSNodes(cluster *kubermaticapiv1.ExternalCluster, nodePoolName string, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterNode, error) {

	var nodesV1 []apiv2.ExternalClusterNode

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[AKSNodepoolNameLabel] == nodePoolName {
				outNode, err := outputNode(n)
				if err != nil {
					return nil, fmt.Errorf("failed to output node %s: %v", n.Name, err)
				}
				nodesV1 = append(nodesV1, *outNode)
			}
		}
	}

	return nodesV1, err
}

func patchAKSMachineDeployment(ctx context.Context, old, new *apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticapiv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterMachineDeployment, error) {
	cred, err := azure.GetCredentialsForAKSCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	agentPoolClient, err := getAKSNodePoolClient(cred)
	if err != nil {
		return nil, err
	}

	nodePoolName := new.NodeDeployment.Name
	currentReplicas := old.NodeDeployment.Spec.Replicas
	desiredReplicas := new.NodeDeployment.Spec.Replicas
	currentVersion := old.NodeDeployment.Spec.Template.Versions.Kubelet
	desiredVersion := new.NodeDeployment.Spec.Template.Versions.Kubelet
	if desiredReplicas != currentReplicas {
		_, err = resizeAKSNodePool(ctx, *agentPoolClient, cloud, nodePoolName, desiredReplicas)
		if err != nil {
			return nil, err
		}
		new.NodeDeployment.Status.Replicas = desiredReplicas
		return new, nil
	}
	if desiredVersion != currentVersion {
		_, err = upgradeNodePool(ctx, *agentPoolClient, cloud, nodePoolName, desiredVersion)
		if err != nil {
			return nil, err
		}
		new.NodeDeployment.Spec.Replicas = currentReplicas
		return new, nil
	}

	return new, nil
}

func resizeAKSNodePool(ctx context.Context, agentPoolClient containerservice.AgentPoolsClient, cloud *kubermaticapiv1.ExternalClusterCloudSpec, nodePoolName string, desiredSize int32) (*containerservice.AgentPoolsCreateOrUpdateFuture, error) {
	pool, err := agentPoolClient.Get(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, nodePoolName)
	if err != nil {
		return nil, err
	}
	nodePool := containerservice.AgentPool{
		Name: &nodePoolName,
		ManagedClusterAgentPoolProfileProperties: &containerservice.ManagedClusterAgentPoolProfileProperties{
			Count: &desiredSize,
		},
	}
	if pool.ManagedClusterAgentPoolProfileProperties.Mode == containerservice.AgentPoolModeSystem {
		nodePool.ManagedClusterAgentPoolProfileProperties.Mode = containerservice.AgentPoolModeSystem
	}
	update, err := updateAKSNodePool(ctx, agentPoolClient, cloud, nodePoolName, nodePool)
	if err != nil {
		return nil, err
	}

	return update, nil
}

func upgradeNodePool(ctx context.Context, agentPoolClient containerservice.AgentPoolsClient, cloud *kubermaticapiv1.ExternalClusterCloudSpec, nodePoolName string, desiredVersion string) (*containerservice.AgentPoolsCreateOrUpdateFuture, error) {
	pool, err := agentPoolClient.Get(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, nodePoolName)
	if err != nil {
		return nil, err
	}
	nodePool := containerservice.AgentPool{
		Name: &nodePoolName,
		ManagedClusterAgentPoolProfileProperties: &containerservice.ManagedClusterAgentPoolProfileProperties{
			OrchestratorVersion: &desiredVersion,
		},
	}
	if pool.ManagedClusterAgentPoolProfileProperties.Mode == containerservice.AgentPoolModeSystem {
		nodePool.ManagedClusterAgentPoolProfileProperties.Mode = containerservice.AgentPoolModeSystem
	}
	update, err := updateAKSNodePool(ctx, agentPoolClient, cloud, nodePoolName, nodePool)
	if err != nil {
		return nil, err
	}

	return update, nil
}

func getAKSClusterClient(cred azure.Credentials) (*containerservice.ManagedClustersClient, error) {
	var err error

	aksClient := containerservice.NewManagedClustersClient(cred.SubscriptionID)
	aksClient.Authorizer, err = auth.NewClientCredentialsConfig(cred.ClientID, cred.ClientSecret, cred.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %v", err.Error())
	}
	return &aksClient, nil
}

func getAKSNodePoolClient(cred azure.Credentials) (*containerservice.AgentPoolsClient, error) {
	var err error

	agentPoolClient := containerservice.NewAgentPoolsClient(cred.SubscriptionID)
	agentPoolClient.Authorizer, err = auth.NewClientCredentialsConfig(cred.ClientID, cred.ClientSecret, cred.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %v", err.Error())
	}
	return &agentPoolClient, nil
}

func getAKSCluster(ctx context.Context, aksClient *containerservice.ManagedClustersClient, cloud *kubermaticapiv1.ExternalClusterCloudSpec) (*containerservice.ManagedCluster, error) {
	resourceGroup := cloud.AKS.ResourceGroup
	clusterName := cloud.AKS.Name

	aksCluster, err := aksClient.Get(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get AKS managed cluster %v from resource group %v: %v", clusterName, resourceGroup, err)
	}

	return &aksCluster, nil
}

func updateAKSNodePool(ctx context.Context, agentPoolClient containerservice.AgentPoolsClient, cloud *kubermaticapiv1.ExternalClusterCloudSpec, nodePoolName string, nodePool containerservice.AgentPool) (*containerservice.AgentPoolsCreateOrUpdateFuture, error) {
	result, err := agentPoolClient.CreateOrUpdate(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, nodePoolName, nodePool)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func deleteAKSNodeGroup(ctx context.Context, cloud *kubermaticapiv1.ExternalClusterCloudSpec, nodePoolName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	cred, err := azure.GetCredentialsForAKSCluster(*cloud, secretKeySelector)
	if err != nil {
		return err
	}

	agentPoolClient, err := getAKSNodePoolClient(cred)
	if err != nil {
		return err
	}

	_, err = agentPoolClient.Delete(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, nodePoolName)
	if err != nil {
		return err
	}
	return nil
}
