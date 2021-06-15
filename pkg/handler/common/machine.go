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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	jsonpatch "github.com/evanphx/json-patch"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/label"
	machineconversions "k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	machineresource "k8c.io/kubermatic/v2/pkg/resources/machine"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errGlue = " & "

	initialConditionParsingDelay = 5

	MachineDeploymentEventWarningType = "warning"
	MachineDeploymentEventNormalType  = "normal"
)

func GenerateDeploymentKeySpec(userSSHKeys []*kubermaticv1.UserSSHKey, caPublicKey []*kubermaticv1.UserSSHKey) (*kubermaticv1.DeploymentSSHKeys, error) {
	keys := kubermaticv1.DeploymentSSHKeys{
		UserSSHKey: userSSHKeys,
	}

	if len(caPublicKey) > 1 {
		return nil, fmt.Errorf("multiple ca keys found")
	}

	if len(caPublicKey) == 1 {
		keys.CAPublicKey = caPublicKey[0]
	}

	return &keys, nil
}

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

	userSSHKeys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: clusterID})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	caPublicKey, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: cluster.Name, IsCAKey: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH keys: %v", err)
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

	keys, err := GenerateDeploymentKeySpec(userSSHKeys, caPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment ssh key spec: %v", err)
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

func GetMachineDeployment(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, machineDeploymentID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: machineDeploymentID}, machineDeployment); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return outputMachineDeployment(machineDeployment)
}

func ListMachineDeploymentNodes(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, machineDeploymentID string, hideInitialConditions bool) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	machines, err := getMachinesForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, projectID, machineDeploymentID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	nodeList, err := getNodeList(ctx, cluster, clusterProvider)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var nodesV1 []*apiv1.Node
	for i := range machines.Items {
		node := getNodeForMachine(&machines.Items[i], nodeList.Items)
		outNode, err := outputMachine(&machines.Items[i], node, hideInitialConditions)
		if err != nil {
			return nil, fmt.Errorf("failed to output machine %s: %v", machines.Items[i].Name, err)
		}

		nodesV1 = append(nodesV1, outNode)
	}

	return nodesV1, nil
}

func ListNodesForCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string, hideInitialConditions bool) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machineList := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, machineList, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		return nil, fmt.Errorf("failed to load machines from cluster: %v", err)
	}

	nodeList, err := getNodeList(ctx, cluster, clusterProvider)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	// The following is a bit tricky. We might have a node which is not created by a machine and vice versa...
	var nodesV1 []*apiv1.Node
	matchedMachineNodes := sets.NewString()

	// Go over all machines first
	for i := range machineList.Items {
		node := getNodeForMachine(&machineList.Items[i], nodeList.Items)
		if node != nil {
			matchedMachineNodes.Insert(string(node.UID))
		}

		// Do not list Machines that are controlled, i.e. by Machine Set.
		if len(machineList.Items[i].ObjectMeta.OwnerReferences) != 0 {
			continue
		}

		outNode, err := outputMachine(&machineList.Items[i], node, hideInitialConditions)
		if err != nil {
			return nil, fmt.Errorf("failed to output machine %s: %v", machineList.Items[i].Name, err)
		}

		nodesV1 = append(nodesV1, outNode)
	}

	// Now all nodes, which do not belong to a machine - Relevant for BYO
	for i := range nodeList.Items {
		if !matchedMachineNodes.Has(string(nodeList.Items[i].UID)) {
			nodesV1 = append(nodesV1, outputNode(&nodeList.Items[i], hideInitialConditions))
		}
	}
	return nodesV1, nil
}

func outputNode(node *corev1.Node, hideInitialNodeConditions bool) *apiv1.Node {
	nodeStatus := apiv1.NodeStatus{}
	nodeStatus = apiNodeStatus(nodeStatus, node, hideInitialNodeConditions)
	var deletionTimestamp *apiv1.Time
	if node.DeletionTimestamp != nil {
		t := apiv1.NewTime(node.DeletionTimestamp.Time)
		deletionTimestamp = &t
	}

	return &apiv1.Node{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                node.Name,
			Name:              node.Name,
			DeletionTimestamp: deletionTimestamp,
			CreationTimestamp: apiv1.NewTime(node.CreationTimestamp.Time),
		},
		Spec: apiv1.NodeSpec{
			Versions:        apiv1.NodeVersionInfo{},
			OperatingSystem: apiv1.OperatingSystemSpec{},
			Cloud:           apiv1.NodeCloudSpec{},
		},
		Status: nodeStatus,
	}
}

func ListMachineDeploymentMetrics(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, machineDeploymentID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	// check if logged user has privileges to list node deployments. If yes then we can use privileged client to
	// get metrics
	machines, err := getMachinesForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, projectID, machineDeploymentID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	nodeList, err := getNodeList(ctx, cluster, clusterProvider)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	availableResources := make(map[string]corev1.ResourceList)
	for i := range machines.Items {
		n := getNodeForMachine(&machines.Items[i], nodeList.Items)
		if n != nil {
			availableResources[n.Name] = n.Status.Allocatable
		}
	}

	dynamicCLient, err := clusterProvider.GetAdminClientForCustomerCluster(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	nodeDeploymentNodesMetrics := make([]v1beta1.NodeMetrics, 0)
	allNodeMetricsList := &v1beta1.NodeMetricsList{}
	if err := dynamicCLient.List(ctx, allNodeMetricsList); err != nil {
		// Happens during cluster creation when the CRD is not setup yet
		if _, ok := err.(*meta.NoKindMatchError); !ok {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
	}

	for _, m := range allNodeMetricsList.Items {
		if _, ok := availableResources[m.Name]; ok {
			nodeDeploymentNodesMetrics = append(nodeDeploymentNodesMetrics, m)
		}
	}

	return ConvertNodeMetrics(nodeDeploymentNodesMetrics, availableResources)
}

func PatchMachineDeployment(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, sshKeyProvider provider.SSHKeyProvider, seedsGetter provider.SeedsGetter, projectID, clusterID, machineDeploymentID string, patch json.RawMessage) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	// We cannot use machineClient.ClusterV1alpha1().MachineDeployments().Patch() method as we are not exposing
	// MachineDeployment type directly. API uses NodeDeployment type and we cannot ensure compatibility here.
	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: machineDeploymentID}, machineDeployment); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	nodeDeployment, err := outputMachineDeployment(machineDeployment)
	if err != nil {
		return nil, fmt.Errorf("cannot output existing node deployment: %v", err)
	}

	nodeDeploymentJSON, err := json.Marshal(nodeDeployment)
	if err != nil {
		return nil, fmt.Errorf("cannot decode existing node deployment: %v", err)
	}

	patchedNodeDeploymentJSON, err := jsonpatch.MergePatch(nodeDeploymentJSON, patch)
	if err != nil {
		return nil, fmt.Errorf("cannot patch node deployment: %v", err)
	}

	var patchedNodeDeployment *apiv1.NodeDeployment
	if err := json.Unmarshal(patchedNodeDeploymentJSON, &patchedNodeDeployment); err != nil {
		return nil, fmt.Errorf("cannot decode patched cluster: %v", err)
	}

	kversion, err := semver.NewVersion(patchedNodeDeployment.Spec.Template.Versions.Kubelet)
	if err != nil {
		return nil, k8cerrors.NewBadRequest("failed to parse kubelet version: %v", err)
	}
	if err = nodeupdate.EnsureVersionCompatible(cluster.Spec.Version.Semver(), kversion); err != nil {
		return nil, k8cerrors.NewBadRequest(err.Error())
	}

	_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, fmt.Errorf("error getting dc: %v", err)
	}

	userSSHKeys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: clusterID})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	caPublicKey, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: cluster.Name, IsCAKey: true})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, k8cerrors.New(http.StatusInternalServerError, "clusterprovider is not a kubernetesprovider.Clusterprovider, can not create nodeDeployment")
	}
	data := common.CredentialsData{
		Ctx:               ctx,
		KubermaticCluster: cluster,
		Client:            assertedClusterProvider.GetSeedClusterAdminRuntimeClient(),
	}

	keys, err := GenerateDeploymentKeySpec(userSSHKeys, caPublicKey)
	if err != nil {
		return nil, err
	}

	patchedMachineDeployment, err := machineresource.Deployment(cluster, patchedNodeDeployment, dc, keys, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine deployment from template: %v", err)
	}

	// Only the fields from NodeDeploymentSpec will be updated by a patch.
	// It ensures that the name and resource version are set and the selector stays the same.
	machineDeployment.Spec.Template.Spec = patchedMachineDeployment.Spec.Template.Spec
	machineDeployment.Spec.Replicas = patchedMachineDeployment.Spec.Replicas
	machineDeployment.Spec.Paused = patchedMachineDeployment.Spec.Paused

	if err := client.Update(ctx, machineDeployment); err != nil {
		return nil, fmt.Errorf("failed to update machine deployment: %v", err)
	}

	return outputMachineDeployment(machineDeployment)
}

func ListMachineDeploymentNodesEvents(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, machineDeploymentID, eventType string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := clusterProvider.GetAdminClientForCustomerCluster(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machines, err := getMachinesForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, projectID, machineDeploymentID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machineSets, err := getMachineSetsForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, projectID, machineDeploymentID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	machineDeployment, err := getMachineDeploymentForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, projectID, machineDeploymentID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	apiEventType := ""
	events := make([]apiv1.Event, 0)

	switch eventType {
	case MachineDeploymentEventWarningType:
		apiEventType = corev1.EventTypeWarning
	case MachineDeploymentEventNormalType:
		apiEventType = corev1.EventTypeNormal
	}

	for _, machine := range machines.Items {
		kubermaticEvents, err := common.GetEvents(ctx, client, &machine, metav1.NamespaceSystem)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		events = append(events, kubermaticEvents...)
	}

	for _, machineSet := range machineSets.Items {
		kubermaticEvents, err := common.GetEvents(ctx, client, &machineSet, metav1.NamespaceSystem)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		events = append(events, kubermaticEvents...)
	}

	kubermaticEvents, err := common.GetEvents(ctx, client, machineDeployment, metav1.NamespaceSystem)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	events = append(events, kubermaticEvents...)

	if len(apiEventType) > 0 {
		events = common.FilterEventsByType(events, apiEventType)
	}

	return events, nil
}

func DeleteMachineDeployment(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, machineDeploymentID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return nil, common.KubernetesErrorToHTTPError(client.Delete(ctx, &clusterv1alpha1.MachineDeployment{ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceSystem, Name: machineDeploymentID}}))
}

func getMachineSetsForNodeDeployment(ctx context.Context, clusterProvider provider.ClusterProvider, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID, nodeDeploymentID string) (*clusterv1alpha1.MachineSetList, error) {
	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, err
	}

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: nodeDeploymentID}, machineDeployment); err != nil {
		return nil, err
	}

	machineSets := &clusterv1alpha1.MachineSetList{}
	listOpts := &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem, LabelSelector: labels.SelectorFromSet(machineDeployment.Spec.Selector.MatchLabels)}
	if err := client.List(ctx, machineSets, listOpts); err != nil {
		return nil, err
	}
	return machineSets, nil
}

func getMachineDeploymentForNodeDeployment(ctx context.Context, clusterProvider provider.ClusterProvider, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID, nodeDeploymentID string) (*clusterv1alpha1.MachineDeployment, error) {
	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, err
	}

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: nodeDeploymentID}, machineDeployment); err != nil {
		return nil, err
	}

	return machineDeployment, nil
}

func outputMachine(machine *clusterv1alpha1.Machine, node *corev1.Node, hideInitialNodeConditions bool) (*apiv1.Node, error) {
	displayName := machine.Spec.Name
	nodeStatus := apiv1.NodeStatus{}
	nodeStatus.MachineName = machine.Name
	var deletionTimestamp *apiv1.Time
	if machine.DeletionTimestamp != nil {
		dt := apiv1.NewTime(machine.DeletionTimestamp.Time)
		deletionTimestamp = &dt
	}

	if machine.Status.ErrorReason != nil {
		nodeStatus.ErrorReason += string(*machine.Status.ErrorReason) + errGlue
		nodeStatus.ErrorMessage += *machine.Status.ErrorMessage + errGlue
	}

	operatingSystemSpec, err := machineconversions.GetAPIV1OperatingSystemSpec(machine.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to get operating system spec from machine: %v", err)
	}

	cloudSpec, err := machineconversions.GetAPIV2NodeCloudSpec(machine.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to get node cloud spec from machine: %v", err)
	}

	if node != nil {
		if node.Name != machine.Spec.Name {
			displayName = node.Name
		}
		nodeStatus = apiNodeStatus(nodeStatus, node, hideInitialNodeConditions)
	}

	nodeStatus.ErrorReason = strings.TrimSuffix(nodeStatus.ErrorReason, errGlue)
	nodeStatus.ErrorMessage = strings.TrimSuffix(nodeStatus.ErrorMessage, errGlue)

	sshUserName, err := machineconversions.GetSSHUserName(operatingSystemSpec, cloudSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to get ssh login name: %v", err)
	}

	return &apiv1.Node{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                machine.Name,
			Name:              displayName,
			DeletionTimestamp: deletionTimestamp,
			CreationTimestamp: apiv1.NewTime(machine.CreationTimestamp.Time),
		},
		Spec: apiv1.NodeSpec{
			Versions: apiv1.NodeVersionInfo{
				Kubelet: machine.Spec.Versions.Kubelet,
			},
			OperatingSystem: *operatingSystemSpec,
			Cloud:           *cloudSpec,
			SSHUserName:     sshUserName,
		},
		Status: nodeStatus,
	}, nil
}

func parseNodeConditions(node *corev1.Node) (reason string, message string) {
	for _, condition := range node.Status.Conditions {
		goodConditionType := condition.Type == corev1.NodeReady
		if goodConditionType && condition.Status != corev1.ConditionTrue {
			reason += condition.Reason + errGlue
			message += condition.Message + errGlue
		} else if !goodConditionType && condition.Status == corev1.ConditionTrue {
			reason += condition.Reason + errGlue
			message += condition.Message + errGlue
		}
	}
	return reason, message
}

func apiNodeStatus(status apiv1.NodeStatus, inputNode *corev1.Node, hideInitialNodeConditions bool) apiv1.NodeStatus {
	for _, address := range inputNode.Status.Addresses {
		status.Addresses = append(status.Addresses, apiv1.NodeAddress{
			Type:    string(address.Type),
			Address: address.Address,
		})
	}

	if !hideInitialNodeConditions || time.Since(inputNode.CreationTimestamp.Time).Minutes() > initialConditionParsingDelay {
		reason, message := parseNodeConditions(inputNode)
		status.ErrorReason += reason
		status.ErrorMessage += message
	}

	status.Allocatable.Memory = inputNode.Status.Allocatable.Memory().String()
	status.Allocatable.CPU = inputNode.Status.Allocatable.Cpu().String()

	status.Capacity.Memory = inputNode.Status.Capacity.Memory().String()
	status.Capacity.CPU = inputNode.Status.Capacity.Cpu().String()

	status.NodeInfo.OperatingSystem = inputNode.Status.NodeInfo.OperatingSystem
	status.NodeInfo.KubeletVersion = inputNode.Status.NodeInfo.KubeletVersion
	status.NodeInfo.Architecture = inputNode.Status.NodeInfo.Architecture
	status.NodeInfo.ContainerRuntimeVersion = inputNode.Status.NodeInfo.ContainerRuntimeVersion
	status.NodeInfo.KernelVersion = inputNode.Status.NodeInfo.KernelVersion
	return status
}

func getNodeList(ctx context.Context, cluster *kubermaticv1.Cluster, clusterProvider provider.ClusterProvider) (*corev1.NodeList, error) {
	client, err := clusterProvider.GetAdminClientForCustomerCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, nodeList); err != nil {
		return nil, err
	}
	return nodeList, nil
}

func getMachinesForNodeDeployment(ctx context.Context, clusterProvider provider.ClusterProvider, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID, nodeDeploymentID string) (*clusterv1alpha1.MachineList, error) {

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, err
	}

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: nodeDeploymentID}, machineDeployment); err != nil {
		return nil, err
	}

	machines := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, machines, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem, LabelSelector: labels.SelectorFromSet(machineDeployment.Spec.Selector.MatchLabels)}); err != nil {
		return nil, err
	}
	return machines, nil
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
