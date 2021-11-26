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

package externalcluster

import (
	"context"
	"fmt"

	"google.golang.org/api/container/v1"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

const GKENodepoolNameLabel = "cloud.google.com/gke-nodepool"

func createGKECluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticapiv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticapiv1.ExternalCluster, error) {
	if cloud.GKE.Name == "" || cloud.GKE.Zone == "" || cloud.GKE.ServiceAccount == "" {
		return nil, errors.NewBadRequest("the GKE cluster name, zone or service account can not be empty")
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticapiv1.ExternalClusterCloudSpec{
		GKE: &kubermaticapiv1.ExternalClusterGKECloudSpec{
			Name: cloud.GKE.Name,
			Zone: cloud.GKE.Zone,
		},
	}
	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.CredentialsSecretsCleanupFinalizer)
	newCluster.Spec.CloudSpec.GKE.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchGKECluster(ctx context.Context, old, new *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalCluster, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gcp.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	updateclusterrequest := &container.UpdateClusterRequest{}
	newVersion := new.Spec.Version.Semver()
	if !old.Spec.Version.Semver().Equal(newVersion) {
		updateclusterrequest.Update = &container.ClusterUpdate{
			DesiredMasterVersion: newVersion.String(),
		}
	}

	req := svc.Projects.Zones.Clusters.Update(project, old.Cloud.GKE.Zone, old.Cloud.GKE.Name, updateclusterrequest)
	_, err = req.Context(ctx).Do()

	return new, err
}

func getGKENodePools(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gcp.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.NodePools.List(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	machineDeployments := make([]apiv2.ExternalClusterMachineDeployment, 0, len(resp.NodePools))

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, md := range resp.NodePools {
		var readyReplicas int32
		for _, n := range nodes.Items {
			if n.Labels != nil {
				if n.Labels[GKENodepoolNameLabel] == md.Name {
					readyReplicas++
				}
			}
		}

		machineDeployments = append(machineDeployments, createMachineDeploymentFromGKENodePoll(md, readyReplicas))
	}

	return machineDeployments, err
}

func createMachineDeploymentFromGKENodePoll(np *container.NodePool, readyReplicas int32) apiv2.ExternalClusterMachineDeployment {
	md := apiv2.ExternalClusterMachineDeployment{
		NodeDeployment: apiv1.NodeDeployment{
			ObjectMeta: apiv1.ObjectMeta{
				ID:   np.Name,
				Name: np.Name,
			},
			Spec: apiv1.NodeDeploymentSpec{
				Replicas: int32(np.InitialNodeCount),
				Template: apiv1.NodeSpec{
					Versions: apiv1.NodeVersionInfo{
						Kubelet: np.Version,
					},
				},
			},
			Status: clusterv1alpha1.MachineDeploymentStatus{
				Replicas:      int32(np.InitialNodeCount),
				ReadyReplicas: readyReplicas,
			},
		},
		Cloud: &apiv2.ExternalClusterMachineDeploymentCloudSpec{
			GKE: &apiv2.GKEMachineDeploymentCloudSpec{},
		},
	}
	if np.Autoscaling != nil {
		md.Cloud.GKE.Autoscaling = &apiv2.GKENodePoolAutoscaling{
			Autoprovisioned: np.Autoscaling.Autoprovisioned,
			Enabled:         np.Autoscaling.Enabled,
			MaxNodeCount:    np.Autoscaling.MaxNodeCount,
			MinNodeCount:    np.Autoscaling.MinNodeCount,
		}
	}
	if np.Config != nil {
		md.Cloud.GKE.Config = &apiv2.GKENodeConfig{
			DiskSizeGb:    np.Config.DiskSizeGb,
			DiskType:      np.Config.DiskType,
			ImageType:     np.Config.ImageType,
			LocalSsdCount: np.Config.LocalSsdCount,
			MachineType:   np.Config.MachineType,
		}
	}
	if np.Management != nil {
		md.Cloud.GKE.Management = &apiv2.GKENodeManagement{
			AutoRepair:  np.Management.AutoRepair,
			AutoUpgrade: np.Management.AutoUpgrade,
		}
	}
	return md
}

func getGKENodePool(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gcp.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	return getGKEMachineDeployment(ctx, svc, project, cluster, nodeGroupName, clusterProvider)
}

func getGKENodes(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, nodePoolID string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterNode, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gcp.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.NodePools.Get(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodePoolID)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	var nodesV1 []apiv2.ExternalClusterNode

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[GKENodepoolNameLabel] == resp.Name {
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

func deleteGKENodePool(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, nodePoolID string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return err
	}
	svc, project, err := gcp.ConnectToContainerService(sa)
	if err != nil {
		return err
	}

	req := svc.Projects.Zones.Clusters.NodePools.Delete(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodePoolID)
	_, err = req.Context(ctx).Do()
	return err
}

func patchGKEMachineDeployment(ctx context.Context, old, new *apiv2.ExternalClusterMachineDeployment, cluster *kubermaticapiv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gcp.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	// The GKE can update Node Pool size or NodePoll object. Can't change both because first update blocks the second one.
	// It's required to update Node Poll size separately.

	// only when size was updates otherwise change NodePoll object with other parameters
	if old.Spec.Replicas != new.Spec.Replicas {
		sizeRequest := &container.SetNodePoolSizeRequest{
			NodeCount: int64(new.Spec.Replicas),
		}
		sizeReq := svc.Projects.Zones.Clusters.NodePools.SetSize(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, old.Name, sizeRequest)
		_, err = sizeReq.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		return new, nil
	}

	updateRequest := &container.UpdateNodePoolRequest{
		NodeVersion: new.Spec.Template.Versions.Kubelet,
	}
	updateReq := svc.Projects.Zones.Clusters.NodePools.Update(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, old.Name, updateRequest)
	_, err = updateReq.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return new, nil
}

func getGKEMachineDeployment(ctx context.Context, svc *container.Service, projectID string, cluster *kubermaticapiv1.ExternalCluster, nodeGroupName string, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	req := svc.Projects.Zones.Clusters.NodePools.Get(projectID, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodeGroupName)
	np, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var readyReplicas int32
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[GKENodepoolNameLabel] == np.Name {
				readyReplicas++
			}
		}
	}
	md := createMachineDeploymentFromGKENodePoll(np, readyReplicas)
	return &md, nil
}

func createGKENodePool(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, machineDeployment apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gcp.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	if machineDeployment.Cloud.GKE == nil {
		return nil, fmt.Errorf("GKE cloud spec cannot be empty")
	}

	gke := machineDeployment.Cloud.GKE

	nodePool := &container.NodePool{
		Config:            nil,
		InitialNodeCount:  int64(machineDeployment.Spec.Replicas),
		InstanceGroupUrls: nil,
		Locations:         nil,
		Management:        nil,
		MaxPodsConstraint: nil,
		Name:              machineDeployment.Name,
		Version:           machineDeployment.Spec.Template.Versions.Kubelet,
	}

	if gke.Config != nil {
		nodePool.Config = &container.NodeConfig{
			DiskSizeGb:    gke.Config.DiskSizeGb,
			DiskType:      gke.Config.DiskType,
			ImageType:     gke.Config.ImageType,
			Labels:        gke.Config.Labels,
			LocalSsdCount: gke.Config.LocalSsdCount,
			MachineType:   gke.Config.MachineType,
		}
	}
	if gke.Autoscaling != nil {
		nodePool.Autoscaling = &container.NodePoolAutoscaling{
			Autoprovisioned: gke.Autoscaling.Autoprovisioned,
			Enabled:         gke.Autoscaling.Enabled,
			MaxNodeCount:    gke.Autoscaling.MaxNodeCount,
			MinNodeCount:    gke.Autoscaling.MinNodeCount,
		}
	}
	if gke.Management != nil {
		nodePool.Management = &container.NodeManagement{
			AutoRepair:  gke.Management.AutoRepair,
			AutoUpgrade: gke.Management.AutoUpgrade,
		}
	}

	createRequest := &container.CreateNodePoolRequest{
		NodePool: nodePool,
	}
	req := svc.Projects.Zones.Clusters.NodePools.Create(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, createRequest)
	_, err = req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return &machineDeployment, nil
}
