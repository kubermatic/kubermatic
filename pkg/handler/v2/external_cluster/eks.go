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
	"strings"

	"github.com/aws/aws-sdk-go/service/eks"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

const (
	EKSNodeGroupStatus    = "ACTIVE"
	EKSNodeGroupNameLabel = "eks.amazonaws.com/nodegroup"
)

func createEKSCluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticapiv1.ExternalCluster, error) {
	if cloud.EKS.Name == "" || cloud.EKS.Region == "" || cloud.EKS.AccessKeyID == "" || cloud.EKS.SecretAccessKey == "" {
		return nil, errors.NewBadRequest("the EKS cluster name, region or credentials can not be empty")
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticapiv1.ExternalClusterCloudSpec{
		EKS: &kubermaticapiv1.ExternalClusterEKSCloudSpec{
			Name:   cloud.EKS.Name,
			Region: cloud.EKS.Region,
		},
	}
	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.CredentialsSecretsCleanupFinalizer)
	newCluster.Spec.CloudSpec.EKS.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchEKSCluster(old, new *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec) (*apiv2.Update, error) {

	accessKeyID, secretAccessKey, err := awsprovider.GetCredentialsForEKSCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	newVersion := new.Spec.Version.Semver()
	newVersionString := strings.TrimSuffix(newVersion.String(), ".0")

	updateInput := eks.UpdateClusterVersionInput{
		Name:    &cloudSpec.EKS.Name,
		Version: &newVersionString,
	}
	updateOutput, err := client.EKS.UpdateClusterVersion(&updateInput)
	if err != nil {
		return nil, err
	}

	update := apiv2.Update{
		CreatedAt: updateOutput.Update.CreatedAt,
		Id:        updateOutput.Update.Id,
		Status:    updateOutput.Update.Status,
		Type:      updateOutput.Update.Status,
	}
	var params []*apiv2.UpdateParam
	for _, param := range updateOutput.Update.Params {
		params = append(params, &apiv2.UpdateParam{
			Type:  param.Type,
			Value: param.Value,
		})
		update.Params = params
	}

	return &update, nil
}

func getEKSNodePools(cluster *kubermaticapiv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	accessKeyID, secretAccessKey, err := awsprovider.GetCredentialsForEKSCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	clusterName := cloudSpec.EKS.Name
	nodeInput := &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	}
	nodeOutput, err := client.EKS.ListNodegroups(nodeInput)
	if err != nil {
		return nil, err
	}
	nodeGroups := nodeOutput.Nodegroups

	machineDeployments := make([]apiv2.ExternalClusterMachineDeployment, 0, len(nodeGroups))

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, nodeGroupName := range nodeGroups {
		var readyReplicas int32
		for _, n := range nodes.Items {
			if n.Labels != nil {
				if n.Labels[EKSNodeGroupNameLabel] == *nodeGroupName {
					readyReplicas++
				}
			}
		}

		nodeGroupInput := &eks.DescribeNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: nodeGroupName,
		}

		nodeGroupOutput, err := client.EKS.DescribeNodegroup(nodeGroupInput)
		if err != nil {
			return nil, err
		}
		nodeGroup := nodeGroupOutput.Nodegroup
		machineDeployments = append(machineDeployments, nodeGroupToMD(nodeGroup, readyReplicas))

	}

	return machineDeployments, err
}

func getEKSNodePool(cluster *kubermaticapiv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	accessKeyID, secretAccessKey, err := awsprovider.GetCredentialsForEKSCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	clusterName := cloudSpec.EKS.Name

	nodeGroupInput := &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(nodeGroupInput)
	if err != nil {
		return nil, err
	}
	nodeGroup := nodeGroupOutput.Nodegroup

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var readyReplicas int32
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[GKENodepoolNameLabel] == nodeGroupName {
				readyReplicas++
			}
		}
	}
	machineDeployment := nodeGroupToMD(nodeGroup, readyReplicas)

	return &machineDeployment, err
}

func nodeGroupToMD(nodeGroup *eks.Nodegroup, readyReplicas int32) apiv2.ExternalClusterMachineDeployment {
	return apiv2.ExternalClusterMachineDeployment{
		NodeDeployment: apiv1.NodeDeployment{
			ObjectMeta: apiv1.ObjectMeta{
				ID:   *nodeGroup.NodegroupName,
				Name: *nodeGroup.NodegroupName,
			},
			Spec: apiv1.NodeDeploymentSpec{
				Replicas: int32(*nodeGroup.ScalingConfig.DesiredSize),
				Template: apiv1.NodeSpec{
					Versions: apiv1.NodeVersionInfo{
						Kubelet: *nodeGroup.Version,
					},
				},
			},
			Status: clusterv1alpha1.MachineDeploymentStatus{
				Replicas:      int32(*nodeGroup.ScalingConfig.DesiredSize),
				ReadyReplicas: readyReplicas,
			},
		},
	}
}

func patchEKSMD(old, new *apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec) (*apiv2.Update, error) {

	accessKeyID, secretAccessKey, err := awsprovider.GetCredentialsForEKSCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}
	clusterName := cloudSpec.EKS.Name
	nodeGroupName := new.NodeDeployment.Name

	desiredReplicas := new.NodeDeployment.Spec.Replicas
	updateOutput, err := resizeEKSNodeGroup(client, clusterName, nodeGroupName, int64(desiredReplicas))
	if err != nil {
		return nil, err
	}
	return updateOutput, err
}

func resizeEKSNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName string, desiredSize int64) (*apiv2.Update, error) {

	nodeGroupInput := eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(&nodeGroupInput)
	if err != nil {
		return nil, err
	}

	nodeGroup := nodeGroupOutput.Nodegroup
	if *nodeGroup.Status != EKSNodeGroupStatus {
		return nil, fmt.Errorf("cannot resize, cluster nodegroup not active")
	}

	scalingConfig := nodeGroup.ScalingConfig
	currentSize := *scalingConfig.DesiredSize
	maxSize := *scalingConfig.MaxSize
	minSize := *scalingConfig.MinSize

	var newScalingConfig eks.NodegroupScalingConfig
	newScalingConfig.DesiredSize = &desiredSize

	switch {
	case currentSize == desiredSize:
		return nil, fmt.Errorf("cluster nodes are already of size: %d", desiredSize)

	case desiredSize > maxSize:
		newScalingConfig.MaxSize = &desiredSize

	case desiredSize < minSize:
		newScalingConfig.MinSize = &desiredSize

	}

	configInput := eks.UpdateNodegroupConfigInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
		ScalingConfig: &newScalingConfig,
	}

	updateOutput, err := client.EKS.UpdateNodegroupConfig(&configInput)
	if err != nil {
		return nil, err
	}

	update := apiv2.Update{
		CreatedAt: updateOutput.Update.CreatedAt,
		Id:        updateOutput.Update.Id,
		Status:    updateOutput.Update.Status,
		Type:      updateOutput.Update.Status,
	}
	var params []*apiv2.UpdateParam
	for _, param := range updateOutput.Update.Params {
		params = append(params, &apiv2.UpdateParam{
			Type:  param.Type,
			Value: param.Value,
		})
		update.Params = params
	}

	return &update, nil
}

func getEKSNodes(cluster *kubermaticapiv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterNode, error) {

	var nodesV1 []apiv2.ExternalClusterNode

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[EKSNodeGroupNameLabel] == nodeGroupName {
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

func deleteEKSNodePool(cluster *kubermaticapiv1.ExternalCluster, nodePoolID string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	accessKeyID, secretAccessKey, err := awsprovider.GetCredentialsForEKSCluster(*cluster.Spec.CloudSpec, secretKeySelector)
	if err != nil {
		return err
	}

	cloudSpec := cluster.Spec.CloudSpec
	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, cloudSpec.EKS.Region)
	if err != nil {
		return err
	}

	deleteNGInput := eks.DeleteNodegroupInput{
		ClusterName:   &cloudSpec.EKS.Name,
		NodegroupName: &nodePoolID,
	}
	_, err = client.EKS.DeleteNodegroup(&deleteNGInput)
	if err != nil {
		return err
	}
	return nil
}
