package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterv1alpha1clientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

const (
	errGlue = " & "

	initialConditionParsingDelay = 5
)

func deleteNodeForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteNodeForClusterReq)
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

		// TODO:
		// normally we have project, user and sshkey providers
		// but here we decided to use machineClient and kubeClient directly to access the user cluster.
		//
		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to create a machine client: %v", err)
		}

		kubeClient, err := clusterProvider.GetKubernetesClientForCustomerCluster(cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to create a kubernetes client: %v", err)
		}

		machine, node, err := findMachineAndNode(req.NodeID, machineClient, kubeClient)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, k8cerrors.NewNotFound("Node", req.NodeID)
		}

		if machine != nil {
			return nil, common.KubernetesErrorToHTTPError(machineClient.ClusterV1alpha1().Machines(machine.Namespace).Delete(machine.Name, nil))
		} else if node != nil {
			return nil, common.KubernetesErrorToHTTPError(kubeClient.CoreV1().Nodes().Delete(node.Name, nil))
		}
		return nil, nil
	}
}

func listNodesForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListNodesForClusterReq)
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

		// TODO:
		// normally we have project, user and sshkey providers
		// but here we decided to use machineClient and kubeClient directly to access the user cluster.
		//
		// how about moving machineClient and kubeClient to their own provider ?
		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineList, err := machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).List(metav1.ListOptions{IncludeUninitialized: true})
		if err != nil {
			return nil, fmt.Errorf("failed to load machines from cluster: %v", err)
		}

		nodeList, err := getNodeList(cluster, clusterProvider)
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

			outNode, err := outputMachine(&machineList.Items[i], node, req.HideInitialConditions)
			if err != nil {
				return nil, fmt.Errorf("failed to output machine %s: %v", machineList.Items[i].Name, err)
			}

			nodesV1 = append(nodesV1, outNode)
		}

		// Now all nodes, which do not belong to a machine - Relevant for BYO
		for i := range nodeList.Items {
			if !matchedMachineNodes.Has(string(nodeList.Items[i].UID)) {
				nodesV1 = append(nodesV1, outputNode(&nodeList.Items[i], req.HideInitialConditions))
			}
		}
		return nodesV1, nil
	}
}

func getNodeList(cluster *v1.Cluster, clusterProvider provider.ClusterProvider) (*corev1.NodeList, error) {
	kubeClient, err := clusterProvider.GetKubernetesClientForCustomerCluster(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
}

func getNodeForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NodeReq)
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

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// TODO:
		// normally we have project, user and sshkey providers
		// but here we decided to use machineClient and kubeClient directly to access the user cluster.
		//
		// how about moving machineClient and kubeClient to their own provider ?
		kubeClient, err := clusterProvider.GetKubernetesClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machine, node, err := findMachineAndNode(req.NodeID, machineClient, kubeClient)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, k8cerrors.NewNotFound("Node", req.NodeID)
		}

		if machine == nil {
			return outputNode(node, req.HideInitialConditions), nil
		}

		return outputMachine(machine, node, req.HideInitialConditions)
	}
}

func createNodeForClusterLegacy(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, k8cerrors.NewWithDetails(http.StatusBadRequest, "Creating Nodes is deprecated. Please create a Node Deployment instead", "If you are calling this API endpoint directrly then use POST \"v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments\" instead")
	}
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

func findMachineAndNode(name string, machineClient clusterv1alpha1clientset.Interface, kubeClient kubernetes.Interface) (*clusterv1alpha1.Machine, *corev1.Node, error) {
	machineList, err := machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).List(metav1.ListOptions{IncludeUninitialized: true})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load machines from cluster: %v", err)
	}

	nodeList, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
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

// DeleteNodeForClusterReq defines HTTP request for deleteNodeForCluster
// swagger:parameters deleteNodeForCluster
type DeleteNodeForClusterReq struct {
	common.GetClusterReq
	// in: path
	NodeID string `json:"node_id"`
}

func decodeDeleteNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteNodeForClusterReq

	nodeID := mux.Vars(r)["node_id"]
	if nodeID == "" {
		return "", fmt.Errorf("'node_id' parameter is required but was not provided")
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
	req.NodeID = nodeID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

// ListNodesForClusterReq defines HTTP request for listNodesForCluster
// swagger:parameters listNodesForCluster
type ListNodesForClusterReq struct {
	common.GetClusterReq
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeListNodesForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req ListNodesForClusterReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.HideInitialConditions, _ = strconv.ParseBool(r.URL.Query().Get("hideInitialConditions"))
	req.ClusterID = clusterID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

// CreateNodeReqLegacy defines HTTP request for createNodeForClusterLegacy
// swagger:parameters createNodeForClusterLegacy
type CreateNodeReqLegacy struct {
	common.GetClusterReq
	// in: body
	Body apiv1.Node
}

func decodeCreateNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateNodeReqLegacy

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

// NodeReq defines HTTP request for getNodeForCluster
// swagger:parameters getNodeForCluster
type NodeReq struct {
	common.GetClusterReq
	// in: path
	NodeID string `json:"node_id"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeGetNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	nodeID := mux.Vars(r)["node_id"]
	if nodeID == "" {
		return nil, fmt.Errorf("'node_id' parameter is required but was not provided")
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeID = nodeID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

// CreateNodeDeploymentReq defines HTTP request for createMachineDeployment
// swagger:parameters createNodeDeployment
type CreateNodeDeploymentReq struct {
	common.GetClusterReq
	// in: body
	Body apiv1.NodeDeployment
}

func decodeCreateNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateNodeDeploymentReq

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

func createNodeDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateNodeDeploymentReq)
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

		// TODO:
		// normally we have project, user and sshkey providers
		// but here we decided to use machineClient and kubeClient directly to access the user cluster.
		//
		// how about moving machineClient and kubeClient to their own provider ?
		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		dc, found := dcs[cluster.Spec.Cloud.DatacenterName]
		if !found {
			return nil, fmt.Errorf("unknown cluster datacenter %s", cluster.Spec.Cloud.DatacenterName)
		}

		nd := &req.Body

		if nd.Spec.Template.Cloud.Openstack == nil &&
			nd.Spec.Template.Cloud.Digitalocean == nil &&
			nd.Spec.Template.Cloud.AWS == nil &&
			nd.Spec.Template.Cloud.Hetzner == nil &&
			nd.Spec.Template.Cloud.VSphere == nil &&
			nd.Spec.Template.Cloud.Azure == nil {
			return nil, k8cerrors.NewBadRequest("cannot create node deployment without cloud provider")
		}

		//TODO: We need to make the kubelet version configurable but restrict it to versions supported by the control plane
		if nd.Spec.Template.Versions.Kubelet != "" {
			kversion, err := semver.NewVersion(nd.Spec.Template.Versions.Kubelet)
			if err != nil {
				return nil, k8cerrors.NewBadRequest("failed to parse kubelet version: %v", err)
			}

			if err = ensureVersionCompatible(cluster.Spec.Version.Semver(), kversion); err != nil {
				return nil, k8cerrors.NewBadRequest(err.Error())
			}

			nd.Spec.Template.Versions.Kubelet = kversion.String()
		} else {
			//TODO: rework the versions
			nd.Spec.Template.Versions.Kubelet = cluster.Spec.Version.String()
		}

		// Create Machine Deployment resource.
		md, err := machineresource.Deployment(cluster, nd, dc, keys)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine deployment from template: %v", err)
		}

		md, err = machineClient.ClusterV1alpha1().MachineDeployments(md.Namespace).Create(md)
		if err != nil {
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

// ListNodeDeploymentsReq defines HTTP request for listNodeDeployments
// swagger:parameters listNodeDeployments
type ListNodeDeploymentsReq struct {
	common.GetClusterReq
}

func decodeListNodeDeployments(c context.Context, r *http.Request) (interface{}, error) {
	var req ListNodeDeploymentsReq

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

func listNodeDeployments(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListNodeDeploymentsReq)
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

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployments, err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).List(metav1.ListOptions{})
		if err != nil {
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

// NodeDeploymentReq defines HTTP request for getNodeDeployment
// swagger:parameters getNodeDeployment
type NodeDeploymentReq struct {
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

func decodeGetNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeDeploymentReq

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

func getNodeDeployment(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NodeDeploymentReq)
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

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployment, err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).Get(req.NodeDeploymentID, metav1.GetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return outputMachineDeployment(machineDeployment)
	}
}

// NodeDeploymentNodesReq defines HTTP request for listNodeDeploymentNodes
// swagger:parameters listNodeDeploymentNodes
type NodeDeploymentNodesReq struct {
	common.GetClusterReq
	// in: path
	NodeDeploymentID string `json:"nodedeployment_id"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeListNodeDeploymentNodes(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeDeploymentNodesReq

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

func listNodeDeploymentNodes(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NodeDeploymentNodesReq)
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

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machineDeployment, err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).Get(req.NodeDeploymentID, metav1.GetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		listOptions := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(machineDeployment.Spec.Selector.MatchLabels).String()}
		machines, err := machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).List(listOptions)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		nodeList, err := getNodeList(cluster, clusterProvider)
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

// PatchNodeDeploymentReq defines HTTP request for patchNodeDeployment endpoint
// swagger:parameters patchNodeDeployment
type PatchNodeDeploymentReq struct {
	NodeDeploymentReq

	// in: body
	Patch []byte
}

func decodePatchNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req PatchNodeDeploymentReq

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

func patchNodeDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchNodeDeploymentReq)
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

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// We cannot use machineClient.ClusterV1alpha1().MachineDeployments().Patch() method as we are not exposing
		// MachineDeployment type directly. API uses NodeDeployment type and we cannot ensure compatibility here.
		machineDeployment, err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).Get(req.NodeDeploymentID, metav1.GetOptions{})
		if err != nil {
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
		if err = ensureVersionCompatible(cluster.Spec.Version.Semver(), kversion); err != nil {
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

		machineDeployment, err = machineClient.ClusterV1alpha1().MachineDeployments(machineDeployment.Namespace).Update(machineDeployment)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine deployment: %v", err)
		}

		return outputMachineDeployment(machineDeployment)
	}
}

// DeleteNodeDeploymentReq defines HTTP request for deleteNodeDeployment
// swagger:parameters deleteNodeDeployment
type DeleteNodeDeploymentReq struct {
	common.GetClusterReq
	// in: path
	NodeDeploymentID string `json:"nodedeployment_id"`
}

func decodeDeleteNodeDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteNodeDeploymentReq

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

func deleteNodeDeployment(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteNodeDeploymentReq)
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

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to create a machine client: %v", err)
		}

		return nil, common.KubernetesErrorToHTTPError(machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).Delete(req.NodeDeploymentID, nil))
	}
}
