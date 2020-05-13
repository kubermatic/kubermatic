package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteNodeForClusterLegacyEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteNodeForClusterLegacyReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machine, node, err := findMachineAndNode(ctx, req.NodeID, client)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, k8cerrors.NewNotFound("Node", req.NodeID)
		}

		if machine != nil {
			return nil, common.KubernetesErrorToHTTPError(client.Delete(ctx, machine))
		} else if node != nil {
			return nil, common.KubernetesErrorToHTTPError(client.Delete(ctx, node))
		}
		return nil, nil
	}
}

func ListNodesForClusterLegacyEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listNodesForClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
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

func GetNodeForClusterLegacyEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getNodeLegacyReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetAdminClientForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		machine, node, err := findMachineAndNode(ctx, req.NodeID, client)
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

func CreateNodeForClusterLegacyEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, k8cerrors.NewWithDetails(http.StatusBadRequest, "Creating Nodes is deprecated. Please create a Node Deployment instead", []string{"If you are calling this API endpoint directly then use POST \"v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments\" instead"})
	}
}

// deleteNodeForClusterLegacyReq defines HTTP request for deleteNodeForClusterLegacy
// swagger:parameters deleteNodeForClusterLegacy
type deleteNodeForClusterLegacyReq struct {
	common.GetClusterReq
	// in: path
	NodeID string `json:"node_id"`
}

func DecodeDeleteNodeForClusterLegacy(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteNodeForClusterLegacyReq

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

// listNodesForClusterReq defines HTTP request for listNodesForClusterLegacy
// swagger:parameters listNodesForClusterLegacy
type listNodesForClusterReq struct {
	common.GetClusterReq
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func DecodeListNodesForClusterLegacy(c context.Context, r *http.Request) (interface{}, error) {
	var req listNodesForClusterReq

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

// createNodeReqLegacyReq defines HTTP request for createNodeForClusterLegacy
// swagger:parameters createNodeForClusterLegacy
type createNodeReqLegacyReq struct {
	common.GetClusterReq
	// in: body
	Body apiv1.Node
}

func DecodeCreateNodeForClusterLegacy(c context.Context, r *http.Request) (interface{}, error) {
	var req createNodeReqLegacyReq

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

// getNodeLegacyReq defines HTTP request for getNodeForClusterLegacy
// swagger:parameters getNodeForClusterLegacy
type getNodeLegacyReq struct {
	common.GetClusterReq
	// in: path
	NodeID string `json:"node_id"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func DecodeGetNodeForClusterLegacy(c context.Context, r *http.Request) (interface{}, error) {
	var req getNodeLegacyReq

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
