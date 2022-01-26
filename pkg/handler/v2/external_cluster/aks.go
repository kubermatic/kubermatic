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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aks"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

const AKSNodepoolNameLabel = "kubernetes.azure.com/agentpool"

func createAKSCluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticv1.ExternalCluster, error) {
	if cloud.AKS.Name == "" || cloud.AKS.TenantID == "" || cloud.AKS.SubscriptionID == "" || cloud.AKS.ClientID == "" || cloud.AKS.ClientSecret == "" || cloud.AKS.ResourceGroup == "" {
		return nil, errors.NewBadRequest("the AKS cluster name, tenant id or subscription id or client id or client secret or resource group can not be empty")
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticv1.ExternalClusterCloudSpec{
		AKS: &kubermaticv1.ExternalClusterAKSCloudSpec{
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

func patchAKSCluster(ctx context.Context, oldCluster, newCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalCluster, error) {
	clusterName := cloud.AKS.Name
	resourceGroup := cloud.AKS.ResourceGroup

	newVersion := newCluster.Spec.Version.Semver().String()

	cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := aks.GetAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := aks.GetAKSCluster(ctx, aksClient, cloud)
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

	return newCluster, nil
}

func getAKSNodePools(ctx context.Context, cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	cloud := cluster.Spec.CloudSpec

	cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := aks.GetAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := aks.GetAKSCluster(ctx, aksClient, cloud)
	if err != nil {
		return nil, err
	}

	poolProfiles := *aksCluster.ManagedClusterProperties.AgentPoolProfiles

	return getAKSMachineDeployments(poolProfiles, cluster, clusterProvider)
}

func getAKSMachineDeployments(poolProfiles []containerservice.ManagedClusterAgentPoolProfile, cluster *kubermaticv1.ExternalCluster, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
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

func getAKSNodePool(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodePoolName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	cloud := cluster.Spec.CloudSpec

	cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := aks.GetAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := aks.GetAKSCluster(ctx, aksClient, cloud)
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

func getAKSMachineDeployment(poolProfile containerservice.ManagedClusterAgentPoolProfile, cluster *kubermaticv1.ExternalCluster, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
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
	md := apiv2.ExternalClusterMachineDeployment{
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
		Cloud: &apiv2.ExternalClusterMachineDeploymentCloudSpec{
			AKS: &apiv2.AKSMachineDeploymentCloudSpec{},
		},
	}

	md.Cloud.AKS.Basics = &apiv2.AgentPoolBasics{
		Mode:                string(nodePool.Mode),
		OsType:              string(nodePool.OsType),
		AvailabilityZones:   nodePool.AvailabilityZones,
		OrchestratorVersion: nodePool.OrchestratorVersion,
		VMSize:              nodePool.VMSize,
		EnableAutoScaling:   nodePool.EnableAutoScaling,
		MaxCount:            nodePool.MaxCount,
		MinCount:            nodePool.MinCount,
		Count:               nodePool.Count,
	}
	md.Cloud.AKS.OptionalSettings = &apiv2.AgentPoolOptionalSettings{
		MaxPods:            nodePool.MaxPods,
		EnableNodePublicIP: nodePool.EnableNodePublicIP,
		UpgradeSettings:    (*apiv2.AgentPoolUpgradeSettings)(nodePool.UpgradeSettings),
		NodeLabels:         nodePool.NodeLabels,
		NodeTaints:         nodePool.NodeTaints,
	}
	md.Cloud.AKS.Tags = nodePool.Tags

	return md
}

func getAKSNodes(cluster *kubermaticv1.ExternalCluster, nodePoolName string, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterNode, error) {
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
					return nil, fmt.Errorf("failed to output node %s: %w", n.Name, err)
				}
				nodesV1 = append(nodesV1, *outNode)
			}
		}
	}

	return nodesV1, err
}

func patchAKSMachineDeployment(ctx context.Context, oldCluster, newCluster *apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterMachineDeployment, error) {
	cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	agentPoolClient, err := getAKSNodePoolClient(cred)
	if err != nil {
		return nil, err
	}

	nodePoolName := newCluster.NodeDeployment.Name
	currentReplicas := oldCluster.NodeDeployment.Spec.Replicas
	desiredReplicas := newCluster.NodeDeployment.Spec.Replicas
	currentVersion := oldCluster.NodeDeployment.Spec.Template.Versions.Kubelet
	desiredVersion := newCluster.NodeDeployment.Spec.Template.Versions.Kubelet
	if desiredReplicas != currentReplicas {
		_, err = resizeAKSNodePool(ctx, *agentPoolClient, cloud, nodePoolName, desiredReplicas)
		if err != nil {
			return nil, err
		}
		newCluster.NodeDeployment.Status.Replicas = desiredReplicas
		return newCluster, nil
	}
	if desiredVersion != currentVersion {
		_, err = upgradeNodePool(ctx, *agentPoolClient, cloud, nodePoolName, desiredVersion)
		if err != nil {
			return nil, err
		}
		newCluster.NodeDeployment.Spec.Replicas = currentReplicas
		return newCluster, nil
	}

	return newCluster, nil
}

func resizeAKSNodePool(ctx context.Context, agentPoolClient containerservice.AgentPoolsClient, cloud *kubermaticv1.ExternalClusterCloudSpec, nodePoolName string, desiredSize int32) (*containerservice.AgentPoolsCreateOrUpdateFuture, error) {
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

func upgradeNodePool(ctx context.Context, agentPoolClient containerservice.AgentPoolsClient, cloud *kubermaticv1.ExternalClusterCloudSpec, nodePoolName string, desiredVersion string) (*containerservice.AgentPoolsCreateOrUpdateFuture, error) {
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

func getAKSNodePoolClient(cred resources.AKSCredentials) (*containerservice.AgentPoolsClient, error) {
	var err error

	agentPoolClient := containerservice.NewAgentPoolsClient(cred.SubscriptionID)
	agentPoolClient.Authorizer, err = auth.NewClientCredentialsConfig(cred.ClientID, cred.ClientSecret, cred.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %w", err)
	}
	return &agentPoolClient, nil
}

func updateAKSNodePool(ctx context.Context, agentPoolClient containerservice.AgentPoolsClient, cloud *kubermaticv1.ExternalClusterCloudSpec, nodePoolName string, nodePool containerservice.AgentPool) (*containerservice.AgentPoolsCreateOrUpdateFuture, error) {
	result, err := agentPoolClient.CreateOrUpdate(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, nodePoolName, nodePool)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func deleteAKSNodeGroup(ctx context.Context, cloud *kubermaticv1.ExternalClusterCloudSpec, nodePoolName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
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

func createAKSNodePool(ctx context.Context, cloud *kubermaticv1.ExternalClusterCloudSpec, machineDeployment apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	cred, err := aks.GetCredentialsForCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	agentPoolClient, err := getAKSNodePoolClient(cred)
	if err != nil {
		return nil, err
	}

	if machineDeployment.Cloud.AKS == nil {
		return nil, fmt.Errorf("AKS cloud spec cannot be empty")
	}

	aks := machineDeployment.Cloud.AKS

	nodePool := &containerservice.AgentPool{
		Name: &machineDeployment.Name,
	}
	property := containerservice.ManagedClusterAgentPoolProfileProperties{
		Count:               &machineDeployment.Spec.Replicas,
		OrchestratorVersion: &machineDeployment.Spec.Template.Versions.Kubelet,
	}

	basics := aks.Basics
	if basics != nil {
		property.Mode = (containerservice.AgentPoolMode)(basics.Mode)
		property.OsType = (containerservice.OSType)(basics.OsType)
		if basics.AvailabilityZones != nil {
			property.AvailabilityZones = basics.AvailabilityZones
		}
		if basics.OrchestratorVersion != nil {
			property.OrchestratorVersion = basics.OrchestratorVersion
		}
		if basics.VMSize != nil {
			property.VMSize = basics.VMSize
		}
		if basics.EnableAutoScaling != nil {
			property.EnableAutoScaling = basics.EnableAutoScaling
			if *property.EnableAutoScaling {
				if basics.MaxCount != nil {
					property.MaxCount = basics.MaxCount
				}
				if basics.MaxCount != nil {
					property.MinCount = basics.MinCount
				}
			}
		}
		if basics.Count != nil {
			property.Count = basics.Count
		}
	}

	optional := aks.OptionalSettings

	if optional != nil {
		if optional.MaxPods != nil {
			property.MaxPods = optional.MaxPods
		}
		if optional.EnableNodePublicIP != nil {
			property.EnableNodePublicIP = optional.EnableNodePublicIP
		}
		if optional.UpgradeSettings != nil {
			property.UpgradeSettings = &containerservice.AgentPoolUpgradeSettings{
				MaxSurge: optional.UpgradeSettings.MaxSurge,
			}
		}
		if optional.NodeLabels != nil {
			property.NodeLabels = optional.NodeLabels
		}
		if optional.NodeTaints != nil {
			property.NodeTaints = optional.NodeTaints
		}
	}
	property.Tags = aks.Tags
	nodePool.ManagedClusterAgentPoolProfileProperties = &property

	_, err = agentPoolClient.CreateOrUpdate(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, *nodePool.Name, *nodePool)
	if err != nil {
		return nil, err
	}

	return &machineDeployment, nil
}
