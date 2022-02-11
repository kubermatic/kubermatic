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
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	eksprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/eks"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

const (
	EKSNodeGroupStatus    = "ACTIVE"
	EKSNodeGroupNameLabel = "eks.amazonaws.com/nodegroup"
)

func createNewEKSCluster(ctx context.Context, eksCloudSpec *apiv2.EKSCloudSpec) error {
	client, err := awsprovider.GetClientSet(eksCloudSpec.AccessKeyID, eksCloudSpec.SecretAccessKey, "", "", eksCloudSpec.Region)
	if err != nil {
		return err
	}

	clusterSpec := eksCloudSpec.ClusterSpec

	fields := reflect.ValueOf(clusterSpec).Elem()
	for i := 0; i < fields.NumField(); i++ {
		yourjsonTags := fields.Type().Field(i).Tag.Get("required")
		if strings.Contains(yourjsonTags, "true") && fields.Field(i).IsZero() {
			return fmt.Errorf("required field is missing %v", fields.Type().Field(i).Tag)
		}
	}
	input := &eks.CreateClusterInput{
		Name: aws.String(eksCloudSpec.Name),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SecurityGroupIds: clusterSpec.ResourcesVpcConfig.SecurityGroupIds,
			SubnetIds:        clusterSpec.ResourcesVpcConfig.SubnetIds,
		},
		RoleArn: aws.String(clusterSpec.RoleArn),
		Version: aws.String(clusterSpec.Version),
	}
	_, err = client.EKS.CreateCluster(input)

	if err != nil {
		return err
	}

	return nil
}

func createOrImportEKSCluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticv1.ExternalCluster, error) {
	fields := reflect.ValueOf(cloud.EKS).Elem()
	for i := 0; i < fields.NumField(); i++ {
		yourjsonTags := fields.Type().Field(i).Tag.Get("required")
		if strings.Contains(yourjsonTags, "true") && fields.Field(i).IsZero() {
			return nil, errors.NewBadRequest("required field is missing: %v", fields.Type().Field(i).Name)
		}
	}

	if cloud.EKS.ClusterSpec != nil {
		if err := createNewEKSCluster(ctx, cloud.EKS); err != nil {
			return nil, err
		}
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticv1.ExternalClusterCloudSpec{
		EKS: &kubermaticv1.ExternalClusterEKSCloudSpec{
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

func patchEKSCluster(oldCluster, newCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalCluster, error) {
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	newVersion := newCluster.Spec.Version.Semver()
	newVersionString := strings.TrimSuffix(newVersion.String(), ".0")

	updateInput := eks.UpdateClusterVersionInput{
		Name:    &cloudSpec.EKS.Name,
		Version: &newVersionString,
	}
	_, err = client.EKS.UpdateClusterVersion(&updateInput)
	if err != nil {
		return nil, err
	}

	return newCluster, nil
}

func getEKSNodeGroups(cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	cloudSpec := cluster.Spec.CloudSpec
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
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
		machineDeployments = append(machineDeployments, createMachineDeploymentFromEKSNodePoll(nodeGroup, readyReplicas))
	}

	return machineDeployments, err
}

func getEKSNodeGroup(cluster *kubermaticv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	cloudSpec := cluster.Spec.CloudSpec

	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	return getEKSMachineDeployment(client, cluster, nodeGroupName, clusterProvider)
}

func getEKSMachineDeployment(client *awsprovider.ClientSet, cluster *kubermaticv1.ExternalCluster, nodeGroupName string, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	clusterName := cluster.Spec.CloudSpec.EKS.Name

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
			if n.Labels[EKSNodeGroupNameLabel] == nodeGroupName {
				readyReplicas++
			}
		}
	}
	machineDeployment := createMachineDeploymentFromEKSNodePoll(nodeGroup, readyReplicas)

	return &machineDeployment, err
}

func createMachineDeploymentFromEKSNodePoll(nodeGroup *eks.Nodegroup, readyReplicas int32) apiv2.ExternalClusterMachineDeployment {
	md := apiv2.ExternalClusterMachineDeployment{
		NodeDeployment: apiv1.NodeDeployment{
			ObjectMeta: apiv1.ObjectMeta{
				ID:   aws.StringValue(nodeGroup.NodegroupName),
				Name: aws.StringValue(nodeGroup.NodegroupName),
			},
			Spec: apiv1.NodeDeploymentSpec{
				Template: apiv1.NodeSpec{
					Versions: apiv1.NodeVersionInfo{
						Kubelet: aws.StringValue(nodeGroup.Version),
					},
				},
			},
			Status: clusterv1alpha1.MachineDeploymentStatus{
				ReadyReplicas: readyReplicas,
			},
		},
		Cloud: &apiv2.ExternalClusterMachineDeploymentCloudSpec{},
	}
	md.Cloud.EKS = &apiv2.EKSMachineDeploymentCloudSpec{
		Subnets:       nodeGroup.Subnets,
		NodeRole:      aws.StringValue(nodeGroup.NodeRole),
		AmiType:       aws.StringValue(nodeGroup.AmiType),
		CapacityType:  aws.StringValue(nodeGroup.CapacityType),
		DiskSize:      aws.Int64Value(nodeGroup.DiskSize),
		InstanceTypes: nodeGroup.InstanceTypes,
		Labels:        nodeGroup.Labels,
		Version:       aws.StringValue(nodeGroup.Version),
	}
	scalingConfig := nodeGroup.ScalingConfig
	if nodeGroup.ScalingConfig == nil {
		return md
	}
	md.Spec.Replicas = int32(aws.Int64Value(scalingConfig.DesiredSize))
	md.Status.Replicas = int32(aws.Int64Value(scalingConfig.DesiredSize))
	md.Cloud.EKS.ScalingConfig = apiv2.NodegroupScalingConfig{
		DesiredSize: aws.Int64Value(scalingConfig.DesiredSize),
		MaxSize:     aws.Int64Value(scalingConfig.MaxSize),
		MinSize:     aws.Int64Value(scalingConfig.MinSize),
	}

	return md
}

func patchEKSMachineDeployment(oldMD, newMD *apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, cluster *kubermaticv1.ExternalCluster) (*apiv2.ExternalClusterMachineDeployment, error) {
	cloudSpec := cluster.Spec.CloudSpec

	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	// The EKS can update Node Group size or NodeGroup object. Can't change both till one of the update is in progress.
	// It's required to update NodeGroup size separately.

	clusterName := cloudSpec.EKS.Name
	nodeGroupName := newMD.NodeDeployment.Name

	currentReplicas := oldMD.NodeDeployment.Spec.Replicas
	desiredReplicas := newMD.NodeDeployment.Spec.Replicas
	currentVersion := oldMD.NodeDeployment.Spec.Template.Versions.Kubelet
	desiredVersion := newMD.NodeDeployment.Spec.Template.Versions.Kubelet
	if desiredReplicas != currentReplicas {
		_, err = resizeEKSNodeGroup(client, clusterName, nodeGroupName, int64(currentReplicas), int64(desiredReplicas))
		if err != nil {
			return nil, err
		}
		newMD.NodeDeployment.Status.Replicas = desiredReplicas
		newMD.NodeDeployment.Spec.Template.Versions.Kubelet = currentVersion
		return newMD, nil
	}

	if desiredVersion != currentVersion {
		_, err = upgradeEKSNodeGroup(client, &clusterName, &nodeGroupName, &currentVersion, &desiredVersion)
		if err != nil {
			return nil, err
		}
		newMD.NodeDeployment.Spec.Replicas = currentReplicas
		return newMD, nil
	}

	return newMD, nil
}

func upgradeEKSNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName, currentVersion, desiredVersion *string) (*eks.UpdateNodegroupVersionOutput, error) {
	nodeGroupInput := eks.UpdateNodegroupVersionInput{
		ClusterName:   clusterName,
		NodegroupName: nodeGroupName,
		Version:       desiredVersion,
	}

	updateOutput, err := client.EKS.UpdateNodegroupVersion(&nodeGroupInput)
	if err != nil {
		return nil, err
	}

	return updateOutput, nil
}

func resizeEKSNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName string, currentSize, desiredSize int64) (*eks.UpdateNodegroupConfigOutput, error) {
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

	return updateOutput, nil
}

func getEKSNodes(cluster *kubermaticv1.ExternalCluster, nodeGroupName string, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterNode, error) {
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
					return nil, fmt.Errorf("failed to output node %s: %w", n.Name, err)
				}
				nodesV1 = append(nodesV1, *outNode)
			}
		}
	}

	return nodesV1, err
}

func deleteEKSNodeGroup(cluster *kubermaticv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cluster.Spec.CloudSpec, secretKeySelector)
	if err != nil {
		return err
	}

	cloudSpec := cluster.Spec.CloudSpec
	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return err
	}

	deleteNGInput := eks.DeleteNodegroupInput{
		ClusterName:   &cloudSpec.EKS.Name,
		NodegroupName: &nodeGroupName,
	}
	_, err = client.EKS.DeleteNodegroup(&deleteNGInput)
	if err != nil {
		return err
	}
	return nil
}

func checkCreatePoolReqValid(machineDeployment apiv2.ExternalClusterMachineDeployment) error {
	eksMD := machineDeployment.Cloud.EKS
	if eksMD == nil {
		return fmt.Errorf("EKS MachineDeployment Spec cannot be nil")
	}
	fields := reflect.ValueOf(eksMD).Elem()
	for i := 0; i < fields.NumField(); i++ {
		yourjsonTags := fields.Type().Field(i).Tag.Get("required")
		if strings.Contains(yourjsonTags, "true") && fields.Field(i).IsZero() {
			return errors.NewBadRequest("required field is missing: %v", fields.Type().Field(i).Name)
		}
	}
	return nil
}

func createEKSNodePool(cloudSpec *kubermaticv1.ExternalClusterCloudSpec, machineDeployment apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	if err := checkCreatePoolReqValid(machineDeployment); err != nil {
		return nil, err
	}
	eksMD := machineDeployment.Cloud.EKS

	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}
	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	createInput := &eks.CreateNodegroupInput{
		ClusterName:   aws.String(cloudSpec.EKS.Name),
		NodegroupName: aws.String(machineDeployment.Name),
		Subnets:       eksMD.Subnets,
		NodeRole:      aws.String(eksMD.NodeRole),
		AmiType:       aws.String(eksMD.AmiType),
		CapacityType:  aws.String(eksMD.CapacityType),
		DiskSize:      aws.Int64(eksMD.DiskSize),
		InstanceTypes: eksMD.InstanceTypes,
		Labels:        eksMD.Labels,
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(eksMD.ScalingConfig.DesiredSize),
			MaxSize:     aws.Int64(eksMD.ScalingConfig.MaxSize),
			MinSize:     aws.Int64(eksMD.ScalingConfig.MinSize),
		},
	}
	_, err = client.EKS.CreateNodegroup(createInput)
	if err != nil {
		return nil, err
	}

	return &machineDeployment, nil
}
