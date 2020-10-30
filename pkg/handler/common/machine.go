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

package common

import (
	"context"
	"fmt"
	"net/http"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/label"
	machineconversions "k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	machineresource "k8c.io/kubermatic/v2/pkg/resources/machine"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateMachineDeployment(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, sshKeyProvider provider.SSHKeyProvider, seedsGetter provider.SeedsGetter, machineDeployment apiv1.NodeDeployment, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	isBYO, err := common.IsBringYourOwnProvider(cluster.Spec.Cloud)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if isBYO {
		return nil, k8cerrors.NewBadRequest("You cannot create a node deployment for KubeAdm provider")
	}

	keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: clusterID})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, project.Name)
	if err != nil {
		return nil, err
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, fmt.Errorf("error getting dc: %v", err)
	}

	nd, err := machineresource.Validate(&machineDeployment, cluster.Spec.Version.Semver())
	if err != nil {
		return nil, k8cerrors.NewBadRequest(fmt.Sprintf("node deployment validation failed: %s", err.Error()))
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, k8cerrors.New(http.StatusInternalServerError, "clusterprovider is not a kubernetesprovider.Clusterprovider, can not create secret")
	}

	data := common.CredentialsData{
		Ctx:               ctx,
		KubermaticCluster: cluster,
		Client:            assertedClusterProvider.GetSeedClusterAdminRuntimeClient(),
	}

	md, err := machineresource.Deployment(cluster, nd, dc, keys, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine deployment from template: %v", err)
	}

	if err := client.Create(ctx, md); err != nil {
		return nil, fmt.Errorf("failed to create machine deployment: %v", err)
	}

	return outputMachineDeployment(md)
}

func outputMachineDeployment(md *clusterv1alpha1.MachineDeployment) (*apiv1.NodeDeployment, error) {
	nodeStatus := apiv1.NodeStatus{}
	nodeStatus.MachineName = md.Name

	var deletionTimestamp *apiv1.Time
	if md.DeletionTimestamp != nil {
		dt := apiv1.NewTime(md.DeletionTimestamp.Time)
		deletionTimestamp = &dt
	}

	operatingSystemSpec, err := machineconversions.GetAPIV1OperatingSystemSpec(md.Spec.Template.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to get operating system spec from machine deployment: %v", err)
	}

	cloudSpec, err := machineconversions.GetAPIV2NodeCloudSpec(md.Spec.Template.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to get node cloud spec from machine deployment: %v", err)
	}

	taints := make([]apiv1.TaintSpec, len(md.Spec.Template.Spec.Taints))
	for i, taint := range md.Spec.Template.Spec.Taints {
		taints[i] = apiv1.TaintSpec{
			Effect: string(taint.Effect),
			Key:    taint.Key,
			Value:  taint.Value,
		}
	}

	hasDynamicConfig := md.Spec.Template.Spec.ConfigSource != nil

	return &apiv1.NodeDeployment{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                md.Name,
			Name:              md.Name,
			DeletionTimestamp: deletionTimestamp,
			CreationTimestamp: apiv1.NewTime(md.CreationTimestamp.Time),
		},
		Spec: apiv1.NodeDeploymentSpec{
			Replicas: *md.Spec.Replicas,
			Template: apiv1.NodeSpec{
				Labels: label.FilterLabels(label.NodeDeploymentResourceType, md.Spec.Template.Spec.Labels),
				Taints: taints,
				Versions: apiv1.NodeVersionInfo{
					Kubelet: md.Spec.Template.Spec.Versions.Kubelet,
				},
				OperatingSystem: *operatingSystemSpec,
				Cloud:           *cloudSpec,
			},
			Paused:        &md.Spec.Paused,
			DynamicConfig: &hasDynamicConfig,
		},
		Status: md.Status,
	}, nil
}

func DeleteMachineNode(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, machineID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machine, node, err := findMachineAndNode(ctx, machineID, client)
	if err != nil {
		return nil, err
	}
	if machine == nil && node == nil {
		return nil, k8cerrors.NewNotFound("Node", machineID)
	}

	if machine != nil {
		return nil, common.KubernetesErrorToHTTPError(client.Delete(ctx, machine))
	} else if node != nil {
		return nil, common.KubernetesErrorToHTTPError(client.Delete(ctx, node))
	}
	return nil, nil

}

func ListMachineDeployments(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	if err := client.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	nodeDeployments := make([]*apiv1.NodeDeployment, 0, len(machineDeployments.Items))
	for i := range machineDeployments.Items {
		nd, err := outputMachineDeployment(&machineDeployments.Items[i])
		if err != nil {
			return nil, fmt.Errorf("failed to output machine deployment %s: %v", machineDeployments.Items[i].Name, err)
		}

		nodeDeployments = append(nodeDeployments, nd)
	}

	return nodeDeployments, nil
}

func findMachineAndNode(ctx context.Context, name string, client ctrlruntimeclient.Client) (*clusterv1alpha1.Machine, *corev1.Node, error) {
	machineList := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, machineList, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		return nil, nil, fmt.Errorf("failed to load machines from cluster: %v", err)
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, nodeList); err != nil {
		return nil, nil, fmt.Errorf("failed to load nodes from cluster: %v", err)
	}

	var node *corev1.Node
	var machine *clusterv1alpha1.Machine

	for i, n := range nodeList.Items {
		if n.Name == name {
			node = &nodeList.Items[i]
			break
		}
	}

	for i, m := range machineList.Items {
		if m.Name == name {
			machine = &machineList.Items[i]
			break
		}
	}

	//Check if we can get a owner ref from a machine
	if node != nil && machine == nil {
		machine = getMachineForNode(node, machineList.Items)
	}

	if machine != nil && node == nil {
		node = getNodeForMachine(machine, nodeList.Items)
	}

	return machine, node, nil
}

func getMachineForNode(node *corev1.Node, machines []clusterv1alpha1.Machine) *clusterv1alpha1.Machine {
	ref := metav1.GetControllerOf(node)
	if ref == nil {
		return nil
	}
	for _, machine := range machines {
		if ref.UID == machine.UID {
			return &machine
		}
	}
	return nil
}

func getNodeForMachine(machine *clusterv1alpha1.Machine, nodes []corev1.Node) *corev1.Node {
	for _, node := range nodes {
		if (machine.Status.NodeRef != nil && node.UID == machine.Status.NodeRef.UID) || node.Name == machine.Name {
			return &node
		}
	}
	return nil
}
