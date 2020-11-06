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

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Masterminds/semver"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// createNodeDeploymentReq defines HTTP request for createMachineDeployment
// swagger:parameters createNodeDeployment
type createNodeDeploymentReq struct {
	common.GetClusterReq
	// in: body
	Body apiv1.NodeDeployment
}

func DecodeCreateNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req createNodeDeploymentReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.DCReq = dcr.(common.DCReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func CreateNodeDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createNodeDeploymentReq)
		return handlercommon.CreateMachineDeployment(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, sshKeyProvider, seedsGetter, req.Body, req.ProjectID, req.ClusterID)
	}
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

// listNodeDeploymentsReq defines HTTP request for listNodeDeployments
// swagger:parameters listNodeDeployments
type listNodeDeploymentsReq struct {
	common.GetClusterReq
}

func DecodeListNodeDeployments(c context.Context, r *http.Request) (interface{}, error) {
	var req listNodeDeploymentsReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func ListNodeDeployments(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listNodeDeploymentsReq)
		return handlercommon.ListMachineDeployments(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

// nodeDeploymentReq defines HTTP request for getNodeDeployment
// swagger:parameters getNodeDeployment
type nodeDeploymentReq struct {
	common.GetClusterReq
	// in: path
	NodeDeploymentID string `json:"nodedeployment_id"`
}

func decodeNodeDeploymentID(c context.Context, r *http.Request) (string, error) {
	nodeDeploymentID := mux.Vars(r)["nodedeployment_id"]
	if nodeDeploymentID == "" {
		return "", fmt.Errorf("'nodedeployment_id' parameter is required but was not provided")
	}

	return nodeDeploymentID, nil
}

func DecodeGetNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req nodeDeploymentReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	nodeDeploymentID, err := decodeNodeDeploymentID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeDeploymentID = nodeDeploymentID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func GetNodeDeployment(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentReq)
		return handlercommon.GetMachineDeployment(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.NodeDeploymentID)
	}
}

// nodeDeploymentNodesReq defines HTTP request for listNodeDeploymentNodes
// swagger:parameters listNodeDeploymentNodes
type nodeDeploymentNodesReq struct {
	common.GetClusterReq
	// in: path
	NodeDeploymentID string `json:"nodedeployment_id"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func DecodeListNodeDeploymentNodes(c context.Context, r *http.Request) (interface{}, error) {
	var req nodeDeploymentNodesReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	nodeDeploymentID, err := decodeNodeDeploymentID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeDeploymentID = nodeDeploymentID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
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

func ListNodeDeploymentNodes(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentNodesReq)
		return handlercommon.ListMachineDeploymentNodes(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.NodeDeploymentID, req.HideInitialConditions)
	}
}

// nodeDeploymentMetricsReq defines HTTP request for listNodeDeploymentMetrics
// swagger:parameters listNodeDeploymentMetrics
type nodeDeploymentMetricsReq struct {
	common.GetClusterReq
	// in: path
	NodeDeploymentID string `json:"nodedeployment_id"`
}

func DecodeListNodeDeploymentMetrics(c context.Context, r *http.Request) (interface{}, error) {
	var req nodeDeploymentMetricsReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	nodeDeploymentID, err := decodeNodeDeploymentID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeDeploymentID = nodeDeploymentID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func ListNodeDeploymentMetrics(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentMetricsReq)
		return handlercommon.ListMachineDeploymentMetrics(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.NodeDeploymentID)
	}
}

// patchNodeDeploymentReq defines HTTP request for patchNodeDeployment endpoint
// swagger:parameters patchNodeDeployment
type patchNodeDeploymentReq struct {
	nodeDeploymentReq

	// in: body
	Patch json.RawMessage
}

func DecodePatchNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req patchNodeDeploymentReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	nodeDeploymentID, err := decodeNodeDeploymentID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeDeploymentID = nodeDeploymentID
	req.DCReq = dcr.(common.DCReq)

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func PatchNodeDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchNodeDeploymentReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// We cannot use machineClient.ClusterV1alpha1().MachineDeployments().Patch() method as we are not exposing
		// MachineDeployment type directly. API uses NodeDeployment type and we cannot ensure compatibility here.
		machineDeployment := &clusterv1alpha1.MachineDeployment{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: req.NodeDeploymentID}, machineDeployment); err != nil {
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

		patchedNodeDeploymentJSON, err := jsonpatch.MergePatch(nodeDeploymentJSON, req.Patch)
		if err != nil {
			return nil, fmt.Errorf("cannot patch node deployment: %v", err)
		}

		var patchedNodeDeployment *apiv1.NodeDeployment
		if err := json.Unmarshal(patchedNodeDeploymentJSON, &patchedNodeDeployment); err != nil {
			return nil, fmt.Errorf("cannot decode patched cluster: %v", err)
		}

		//TODO: We need to make the kubelet version configurable but restrict it to versions supported by the control plane
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

		keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
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
}

// deleteNodeDeploymentReq defines HTTP request for deleteNodeDeployment
// swagger:parameters deleteNodeDeployment
type deleteNodeDeploymentReq struct {
	common.GetClusterReq
	// in: path
	NodeDeploymentID string `json:"nodedeployment_id"`
}

func DecodeDeleteNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteNodeDeploymentReq

	nodeDeploymentID, err := decodeNodeDeploymentID(c, r)
	if err != nil {
		return nil, err
	}

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeDeploymentID = nodeDeploymentID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

func DeleteNodeDeployment(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteNodeDeploymentReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, common.KubernetesErrorToHTTPError(client.Delete(ctx, &clusterv1alpha1.MachineDeployment{ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceSystem, Name: req.NodeDeploymentID}}))
	}
}

const (
	warningType = "warning"
	normalType  = "normal"
)

// nodeDeploymentNodesEventsReq defines HTTP request for listNodeDeploymentNodesEvents endpoint
// swagger:parameters listNodeDeploymentNodesEvents
type nodeDeploymentNodesEventsReq struct {
	common.GetClusterReq
	// in: query
	Type string `json:"type,omitempty"`

	// in: path
	NodeDeploymentID string `json:"nodedeployment_id"`
}

func DecodeListNodeDeploymentNodesEvents(c context.Context, r *http.Request) (interface{}, error) {
	var req nodeDeploymentNodesEventsReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	nodeDeploymentID, err := decodeNodeDeploymentID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeDeploymentID = nodeDeploymentID
	req.DCReq = dcr.(common.DCReq)

	req.Type = r.URL.Query().Get("type")
	if len(req.Type) > 0 {
		if req.Type == warningType || req.Type == normalType {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query parameter, unsupported type: %s", req.Type)
	}

	return req, nil
}

func ListNodeDeploymentNodesEvents(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentNodesEventsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machines, err := getMachinesForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, req.ProjectID, req.NodeDeploymentID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineSets, err := getMachineSetsForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, req.ProjectID, req.NodeDeploymentID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployment, err := getMachineDeploymentForNodeDeployment(ctx, clusterProvider, userInfoGetter, cluster, req.ProjectID, req.NodeDeploymentID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		eventType := ""
		events := make([]apiv1.Event, 0)

		switch req.Type {
		case warningType:
			eventType = corev1.EventTypeWarning
		case normalType:
			eventType = corev1.EventTypeNormal
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

		if len(eventType) > 0 {
			events = common.FilterEventsByType(events, eventType)
		}

		return events, nil
	}
}
