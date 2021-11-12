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
	"strings"

	"github.com/aws/aws-sdk-go/service/eks"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

const EKSNodeGroupNameLabel = "eks.amazonaws.com/nodegroup"

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

func patchEKSCluster(ctx context.Context, old, new *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec) (*string, error) {

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

	status := "Cluster Upgrade " + *updateOutput.Update.Status

	return &status, nil
}

func getEKSNodePools(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
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
		machineDeployments = append(machineDeployments, apiv2.ExternalClusterMachineDeployment{
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
		})

	}

	return machineDeployments, err
}
