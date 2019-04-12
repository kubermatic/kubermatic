package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	machineconversions "github.com/kubermatic/kubermatic/api/pkg/machine"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errGlue = " & "

	initialConditionParsingDelay = 5
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

func CreateNodeDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createNodeDeploymentReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		dc, found := dcs[cluster.Spec.Cloud.DatacenterName]
		if !found {
			return nil, fmt.Errorf("unknown cluster datacenter %s", cluster.Spec.Cloud.DatacenterName)
		}

		nd, err := machineresource.Validate(&req.Body, cluster.Spec.Version.Semver())
		if err != nil {
			return nil, k8cerrors.NewBadRequest(fmt.Sprintf("node deployment validation failed: %s", err.Error()))
		}

		md, err := machineresource.Deployment(cluster, nd, dc, keys)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine deployment from template: %v", err)
		}

		if err := client.Create(ctx, md); err != nil {
			return nil, fmt.Errorf("failed to create machine deployment: %v", err)
		}

		return outputMachineDeployment(md)
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
				Labels: md.Spec.Template.Spec.Labels,
				Versions: apiv1.NodeVersionInfo{
					Kubelet: md.Spec.Template.Spec.Versions.Kubelet,
				},
				OperatingSystem: *operatingSystemSpec,
				Cloud:           *cloudSpec,
			},
			Paused: &md.Spec.Paused,
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

func ListNodeDeployments(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listNodeDeploymentsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
		if err := client.List(ctx, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}, machineDeployments); err != nil {
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

func GetNodeDeployment(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployment := &clusterv1alpha1.MachineDeployment{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: req.NodeDeploymentID}, machineDeployment); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return outputMachineDeployment(machineDeployment)
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

func getMachinesForNodeDeployment(ctx context.Context, clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo, cluster *v1.Cluster, nodeDeploymentID string) (*clusterv1alpha1.MachineList, error) {

	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return nil, err
	}

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: nodeDeploymentID}, machineDeployment); err != nil {
		return nil, err
	}

	machines := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem, LabelSelector: labels.SelectorFromSet(machineDeployment.Spec.Selector.MatchLabels)}, machines); err != nil {
		return nil, err
	}
	return machines, nil
}

func getMachineSetsForNodeDeployment(ctx context.Context, clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo, cluster *v1.Cluster, nodeDeploymentID string) (*clusterv1alpha1.MachineSetList, error) {
	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return nil, err
	}

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: nodeDeploymentID}, machineDeployment); err != nil {
		return nil, err
	}

	machineSets := &clusterv1alpha1.MachineSetList{}
	if err := client.List(ctx, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem, LabelSelector: labels.SelectorFromSet(machineDeployment.Spec.Selector.MatchLabels)}, machineSets); err != nil {
		return nil, err
	}
	return machineSets, nil
}

func getMachineDeploymentForNodeDeployment(ctx context.Context, clusterProvider provider.ClusterProvider, userInfo *provider.UserInfo, cluster *v1.Cluster, nodeDeploymentID string) (*clusterv1alpha1.MachineDeployment, error) {
	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return nil, err
	}

	machineDeployment := &clusterv1alpha1.MachineDeployment{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: nodeDeploymentID}, machineDeployment); err != nil {
		return nil, err
	}

	return machineDeployment, nil
}

func ListNodeDeploymentNodes(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentNodesReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machines, err := getMachinesForNodeDeployment(ctx, clusterProvider, userInfo, cluster, req.NodeDeploymentID)
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
			outNode, err := outputMachine(&machines.Items[i], node, req.HideInitialConditions)
			if err != nil {
				return nil, fmt.Errorf("failed to output machine %s: %v", machines.Items[i].Name, err)
			}

			nodesV1 = append(nodesV1, outNode)
		}

		return nodesV1, nil
	}
}

// patchNodeDeploymentReq defines HTTP request for patchNodeDeployment endpoint
// swagger:parameters patchNodeDeployment
type patchNodeDeploymentReq struct {
	nodeDeploymentReq

	// in: body
	Patch []byte
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

func PatchNodeDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchNodeDeploymentReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
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
		err = json.Unmarshal(patchedNodeDeploymentJSON, &patchedNodeDeployment)
		if err != nil {
			return nil, fmt.Errorf("cannot decode patched cluster: %v", err)
		}

		//TODO: We need to make the kubelet version configurable but restrict it to versions supported by the control plane
		kversion, err := semver.NewVersion(patchedNodeDeployment.Spec.Template.Versions.Kubelet)
		if err != nil {
			return nil, k8cerrors.NewBadRequest("failed to parse kubelet version: %v", err)
		}
		if err = common.EnsureVersionCompatible(cluster.Spec.Version.Semver(), kversion); err != nil {
			return nil, k8cerrors.NewBadRequest(err.Error())
		}

		dc, found := dcs[cluster.Spec.Cloud.DatacenterName]
		if !found {
			return nil, fmt.Errorf("unknown cluster datacenter %s", cluster.Spec.Cloud.DatacenterName)
		}

		keys, err := sshKeyProvider.List(project, &provider.SSHKeyListOptions{ClusterName: req.ClusterID})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		patchedMachineDeployment, err := machineresource.Deployment(cluster, patchedNodeDeployment, dc, keys)
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

func DeleteNodeDeployment(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteNodeDeploymentReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to create a machine client: %v", err)
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
	Type string

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
		return nil, fmt.Errorf("wrong query paramater, unsupported type: %s", req.Type)
	}

	return req, nil
}

func ListNodeDeploymentNodesEvents() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentNodesEventsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machines, err := getMachinesForNodeDeployment(ctx, clusterProvider, userInfo, cluster, req.NodeDeploymentID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineSets, err := getMachineSetsForNodeDeployment(ctx, clusterProvider, userInfo, cluster, req.NodeDeploymentID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployment, err := getMachineDeploymentForNodeDeployment(ctx, clusterProvider, userInfo, cluster, req.NodeDeploymentID)
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
			kubermaticEvents, err := getEvents(ctx, client, &machine)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			events = append(events, kubermaticEvents...)
		}

		for _, machineSet := range machineSets.Items {
			kubermaticEvents, err := getEvents(ctx, client, &machineSet)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			events = append(events, kubermaticEvents...)
		}

		kubermaticEvents, err := getEvents(ctx, client, machineDeployment)
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

func getEvents(ctx context.Context, client ctrlruntimeclient.Client, obj metav1.Object) ([]apiv1.Event, error) {
	events := &corev1.EventList{}
	listOpts := &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem, FieldSelector: fields.OneTermEqualSelector("involvedObject.uid", string(obj.GetUID()))}
	if err := client.List(ctx, listOpts, events); err != nil {
		return nil, err
	}

	kubermaticEvents := make([]apiv1.Event, 0)
	for _, event := range events.Items {
		kubermaticEvent := common.ConvertInternalEventToExternal(event)
		kubermaticEvents = append(kubermaticEvents, kubermaticEvent)
	}

	return kubermaticEvents, nil
}

func getNodeList(ctx context.Context, cluster *v1.Cluster, clusterProvider provider.ClusterProvider) (*corev1.NodeList, error) {
	client, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
	if err != nil {
		return nil, err
	}

	nodeList := &corev1.NodeList{}
	if err := client.List(ctx, &ctrlruntimeclient.ListOptions{}, nodeList); err != nil {
		return nil, err
	}
	return nodeList, nil
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
	return status
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

func getNodeForMachine(machine *clusterv1alpha1.Machine, nodes []corev1.Node) *corev1.Node {
	for _, node := range nodes {
		if (machine.Status.NodeRef != nil && node.UID == machine.Status.NodeRef.UID) || node.Name == machine.Name {
			return &node
		}
	}
	return nil
}
