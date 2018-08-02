package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Masterminds/semver"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	apierrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

func newDeleteNodeForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewDeleteNodeForClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		project, err := projectProvider.Get(user, req.ProjectID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
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

		machine, node, err := tryToFindMachineAndNode(req.NodeName, machineClient, kubeClient)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, k8cerrors.NewNotFound("Node", req.NodeName)
		}

		if machine != nil {
			return nil, kubernetesErrorToHTTPError(machineClient.MachineV1alpha1().Machines().Delete(machine.Name, nil))
		} else if node != nil {
			return nil, kubernetesErrorToHTTPError(kubeClient.CoreV1().Nodes().Delete(node.Name, nil))
		}
		return nil, nil
	}
}

func newListNodesForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewListNodesForClusterReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		project, err := projectProvider.Get(user, req.ProjectID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// TODO:
		// normally we have project, user and sshkey providers
		// but here we decided to use machineClient and kubeClient directly to access the user cluster.
		//
		// how about moving machineClient and kubeClient to their own provider ?
		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		kubeClient, err := clusterProvider.GetKubernetesClientForCustomerCluster(cluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		machineList, err := machineClient.MachineV1alpha1().Machines().List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to load machines from cluster: %v", err)
		}

		nodeList, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		//The following is a bit tricky. We might have a node which is not created by a machine and vice versa...
		var nodesV2 []*apiv2.Node
		matchedMachineNodes := sets.NewString()

		//Go over all machines first
		for i := range machineList.Items {
			node := getNodeForMachine(&machineList.Items[i], nodeList.Items)
			if node != nil {
				matchedMachineNodes.Insert(string(node.UID))
			}
			outNode, err := outputMachine(&machineList.Items[i], node, req.HideInitialConditions)
			if err != nil {
				return nil, fmt.Errorf("failed to output machine %s: %v", machineList.Items[i].Name, err)
			}
			nodesV2 = append(nodesV2, outNode)
		}

		// Now all nodes, which do not belong to a machine - Relevant for BYO
		for i := range nodeList.Items {
			if !matchedMachineNodes.Has(string(nodeList.Items[i].UID)) {
				nodesV2 = append(nodesV2, outputNode(&nodeList.Items[i], req.HideInitialConditions))
			}
		}
		return convertNodesV2ToNodesV1(nodesV2), nil
	}
}

func newGetNodeForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewNodeReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		project, err := projectProvider.Get(user, req.ProjectID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// TODO:
		// normally we have project, user and sshkey providers
		// but here we decided to use machineClient and kubeClient directly to access the user cluster.
		//
		// how about moving machineClient and kubeClient to their own provider ?
		kubeClient, err := clusterProvider.GetKubernetesClientForCustomerCluster(cluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		machine, node, err := tryToFindMachineAndNode(req.NodeName, machineClient, kubeClient)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, apierrors.NewNotFound("Node", req.NodeName)
		}

		if machine == nil {
			return convertNodeV2ToNodeV1(outputNode(node, req.HideInitialConditions)), nil
		}

		nodeV2, err := outputMachine(machine, node, req.HideInitialConditions)
		if err != nil {
			return nil, err
		}
		return convertNodeV2ToNodeV1(nodeV2), nil
	}
}

func newCreateNodeForCluster(sshKeyProvider provider.NewSSHKeyProvider, projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewCreateNodeReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		project, err := projectProvider.Get(user, req.ProjectID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		keys, err := sshKeyProvider.List(user, project, &provider.SSHKeyListOptions{ClusterName: req.ClusterName})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// TODO:
		// normally we have project, user and sshkey providers
		// but here we decided to use machineClient and kubeClient directly to access the user cluster.
		//
		// how about moving machineClient and kubeClient to their own provider ?
		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		dc, found := dcs[cluster.Spec.Cloud.DatacenterName]
		if !found {
			return nil, fmt.Errorf("unknown cluster datacenter %s", cluster.Spec.Cloud.DatacenterName)
		}

		node := convertNodeV1ToNodeV2(&req.Body)
		if node.Spec.Cloud.Openstack == nil &&
			node.Spec.Cloud.Digitalocean == nil &&
			node.Spec.Cloud.AWS == nil &&
			node.Spec.Cloud.Hetzner == nil &&
			node.Spec.Cloud.VSphere == nil &&
			node.Spec.Cloud.Azure == nil {
			return nil, errors.NewBadRequest("cannot create node without cloud provider")
		}

		// Support matrix: Ubuntu (crio + docker), containerlinux (docker), centos (docker)
		usesDocker := node.Spec.Versions.ContainerRuntime.Name == string(containerruntime.Docker)
		if node.Spec.OperatingSystem.CentOS != nil && !usesDocker {
			return nil, fmt.Errorf("only docker is allowd when using centos")
		}
		if node.Spec.OperatingSystem.ContainerLinux != nil && !usesDocker {
			return nil, fmt.Errorf("only docker is allowd when using container linux")
		}
		if node.Spec.OperatingSystem.ContainerLinux == nil && node.Spec.OperatingSystem.Ubuntu == nil && node.Spec.OperatingSystem.CentOS == nil {
			return nil, fmt.Errorf("no operating system specified")
		}

		//TODO(mrIncompetent): We need to make the kubelet version configurable but restrict it to master version
		if node.Spec.Versions.Kubelet != "" {
			kversion, err := semver.NewVersion(node.Spec.Versions.Kubelet)
			if err != nil {
				return nil, fmt.Errorf("failed to parse kubelet version: %v", err)
			}
			c, err := semver.NewConstraint(kubeletVersionConstraint)
			if err != nil {
				return nil, fmt.Errorf("failed to parse kubelet constraint version: %v", err)
			}

			if !c.Check(kversion) {
				return nil, fmt.Errorf("kubelet version does not fit constraint. Allowed %s", kubeletVersionConstraint)
			}
			node.Spec.Versions.Kubelet = kversion.String()
		} else {
			//TODO(mrIncompetent): rework the versions
			node.Spec.Versions.Kubelet = cluster.Spec.Version
		}

		if node.Metadata.Name == "" {
			node.Metadata.Name = "kubermatic-" + cluster.Name + "-" + rand.String(5)
		}

		// Create machine resource
		machine, err := machineresource.Machine(cluster, node, dc, keys)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine from template: %v", err)
		}

		// Send machine resource to k8s
		machine, err = machineClient.MachineV1alpha1().Machines().Create(machine)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine: %v", err)
		}

		nodeV2, err := outputMachine(machine, nil, false)
		if err != nil {
			return nil, err
		}
		return convertNodeV2ToNodeV1(nodeV2), nil
	}
}

func convertNodeV1ToNodeV2(nodeV1 *apiv1.Node) *apiv2.Node {
	return &apiv2.Node{
		Metadata: apiv2.ObjectMeta{
			Name:              nodeV1.ID,
			DisplayName:       nodeV1.Name,
			CreationTimestamp: nodeV1.CreationTimestamp,
			DeletionTimestamp: nodeV1.DeletionTimestamp,
		},
		Spec:   nodeV1.Spec,
		Status: nodeV1.Status,
	}
}

func convertNodeV2ToNodeV1(nodeV2 *apiv2.Node) *apiv1.Node {
	return &apiv1.Node{
		NewObjectMeta: apiv1.NewObjectMeta{
			ID:                nodeV2.Metadata.Name,
			Name:              nodeV2.Metadata.DisplayName,
			CreationTimestamp: nodeV2.Metadata.CreationTimestamp,
			DeletionTimestamp: nodeV2.Metadata.DeletionTimestamp,
		},
		Spec:   nodeV2.Spec,
		Status: nodeV2.Status,
	}
}

func convertNodesV2ToNodesV1(nodesV2 []*apiv2.Node) []*apiv1.Node {
	nodesV1 := make([]*apiv1.Node, len(nodesV2))
	for index, nodeV2 := range nodesV2 {

		nodesV1[index] = convertNodeV2ToNodeV1(nodeV2)
	}
	return nodesV1
}

// NewDeleteNodeForClusterReq defines HTTP request for newDeleteNodeForCluster
// swagger:parameters newDeleteNodeForCluster
type NewDeleteNodeForClusterReq struct {
	NewGetClusterReq
	// in: path
	NodeName string `json:"node_name"`
}

func decodeDeleteNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req NewDeleteNodeForClusterReq

	nodeName := mux.Vars(r)["node_name"]
	if nodeName == "" {
		return "", fmt.Errorf("'node_name' parameter is required but was not provided")
	}

	clusterName, err := decodeClusterName(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterName = clusterName
	req.NodeName = nodeName
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// NewListNodesForClusterReq defines HTTP request for newListNodesForCluster
// swagger:parameters newListNodesForCluster
type NewListNodesForClusterReq struct {
	NewGetClusterReq
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeListNodesForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req NewListNodesForClusterReq

	clusterName, err := decodeClusterName(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.HideInitialConditions, _ = strconv.ParseBool(r.URL.Query().Get("hideInitialConditions"))
	req.ClusterName = clusterName
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// NewCreateNodeReq defines HTTP request for newCreateNodeForCluster
// swagger:parameters newCreateNodeForCluster
type NewCreateNodeReq struct {
	NewGetClusterReq
	// in: body
	Body apiv1.Node
}

func decodeCreateNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req NewCreateNodeReq

	clusterName, err := decodeClusterName(c, r)
	if err != nil {
		return nil, err
	}
	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterName = clusterName
	req.DCReq = dcr.(DCReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// NewNodeReq defines HTTP request for newGetNodeForCluster
// swagger:parameters newGetNodeForCluster
type NewNodeReq struct {
	NewGetClusterReq
	// in: path
	NodeName string `json:"node_name"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeGetNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req NewNodeReq

	clusterName, err := decodeClusterName(c, r)
	if err != nil {
		return nil, err
	}
	nodeName := mux.Vars(r)["node_name"]
	if nodeName == "" {
		return nil, fmt.Errorf("'node_name' parameter is required but was not provided")
	}

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterName = clusterName
	req.NodeName = nodeName
	req.DCReq = dcr.(DCReq)

	return req, nil
}
