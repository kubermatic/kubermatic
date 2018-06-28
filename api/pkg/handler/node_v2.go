package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	machineconversions "github.com/kubermatic/kubermatic/api/pkg/machine"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
)

const (
	kubeletVersionConstraint = ">= 1.8"
	errGlue                  = " & "

	initialConditionParsingDelay = 5
)

// CreateNodeReqV2 represent a request for specific data to create a node
// swagger:parameters createClusterNodeV2 createClusterNodeV3
type CreateNodeReqV2 struct {
	GetClusterReq
	// in: body
	Body CreateNodeReqBodyV2
}

// CreateNodeReqBodyV2 represents the request body of a create nodes request
type CreateNodeReqBodyV2 struct {
	apiv2.Node
}

// NodeReqV2 represent a request for node specific data
// swagger:parameters getClusterNodeV2 deleteClusterNodeV2 deleteClusterNodeV3 getClusterNodeV3
type NodeReqV2 struct {
	GetClusterReq
	// in: path
	NodeName string `json:"node"`
}

func decodeCreateNodeReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateNodeReqV2

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(GetClusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func outputNode(node *corev1.Node, hideInitialNodeConditions bool) *apiv2.Node {
	nodeStatus := apiv2.NodeStatus{}
	nodeStatus = apiNodeStatus(nodeStatus, node, hideInitialNodeConditions)
	var deletionTimestamp *time.Time
	if node.DeletionTimestamp != nil {
		deletionTimestamp = &node.DeletionTimestamp.Time
	}

	return &apiv2.Node{
		Metadata: apiv2.ObjectMeta{
			Name:              node.Name,
			DisplayName:       node.Name,
			Labels:            node.Labels,
			Annotations:       node.Annotations,
			DeletionTimestamp: deletionTimestamp,
			CreationTimestamp: node.CreationTimestamp.Time,
		},
		Spec: apiv2.NodeSpec{
			Versions:        apiv2.NodeVersionInfo{},
			OperatingSystem: apiv2.OperatingSystemSpec{},
			Cloud:           apiv2.NodeCloudSpec{},
		},
		Status: nodeStatus,
	}
}

func apiNodeStatus(status apiv2.NodeStatus, inputNode *corev1.Node, hideInitialNodeConditions bool) apiv2.NodeStatus {
	for _, address := range inputNode.Status.Addresses {
		status.Addresses = append(status.Addresses, apiv2.NodeAddress{
			Type:    string(address.Type),
			Address: string(address.Address),
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

func outputMachine(machine *v1alpha1.Machine, node *corev1.Node, hideInitialNodeConditions bool) (*apiv2.Node, error) {
	displayName := machine.Spec.Name
	labels := map[string]string{}
	annotations := map[string]string{}
	nodeStatus := apiv2.NodeStatus{}
	nodeStatus.MachineName = machine.Name
	var deletionTimestamp *time.Time
	if machine.DeletionTimestamp != nil {
		deletionTimestamp = &machine.DeletionTimestamp.Time
	}

	if machine.Status.ErrorReason != nil {
		nodeStatus.ErrorReason += string(*machine.Status.ErrorReason) + errGlue
		nodeStatus.ErrorMessage += string(*machine.Status.ErrorMessage) + errGlue
	}

	operatingSystemSpec, err := machineconversions.GetAPIV2OperatingSystemSpec(machine)
	if err != nil {
		return nil, fmt.Errorf("failed to get operating system spec from machine: %v", err)
	}

	cloudSpec, err := machineconversions.GetAPIV2NodeCloudSpec(machine)
	if err != nil {
		return nil, fmt.Errorf("failed to get node cloud spec from machine: %v", err)
	}

	if machine.Status.Versions != nil {
		nodeStatus.NodeInfo.ContainerRuntime = machine.Status.Versions.ContainerRuntime.Name
		nodeStatus.NodeInfo.ContainerRuntimeVersion = machine.Status.Versions.ContainerRuntime.Version
	}

	if node != nil {
		if node.Name != machine.Spec.Name {
			displayName = node.Name
		}

		labels = node.Labels
		annotations = node.Annotations
		nodeStatus = apiNodeStatus(nodeStatus, node, hideInitialNodeConditions)
	}

	nodeStatus.ErrorReason = strings.TrimSuffix(nodeStatus.ErrorReason, errGlue)
	nodeStatus.ErrorMessage = strings.TrimSuffix(nodeStatus.ErrorMessage, errGlue)

	return &apiv2.Node{
		Metadata: apiv2.ObjectMeta{
			Name:              machine.Name,
			DisplayName:       displayName,
			Labels:            labels,
			Annotations:       annotations,
			DeletionTimestamp: deletionTimestamp,
			CreationTimestamp: machine.CreationTimestamp.Time,
		},
		Spec: apiv2.NodeSpec{
			Versions: apiv2.NodeVersionInfo{
				Kubelet: machine.Spec.Versions.Kubelet,
				ContainerRuntime: apiv2.NodeContainerRuntimeInfo{
					Name:    machine.Spec.Versions.ContainerRuntime.Name,
					Version: machine.Spec.Versions.ContainerRuntime.Version,
				},
			},
			OperatingSystem: *operatingSystemSpec,
			Cloud:           *cloudSpec,
		},
		Status: nodeStatus,
	}, nil
}

func parseNodeConditions(node *corev1.Node) (reason string, message string) {
	for _, condition := range node.Status.Conditions {
		goodConditionType := condition.Type == corev1.NodeReady || condition.Type == corev1.NodeKubeletConfigOk
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

func createNodeEndpointV2(dcs map[string]provider.DatacenterMeta, dp provider.SSHKeyProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		req := request.(CreateNodeReqV2)
		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			return nil, err
		}

		keys, err := dp.ClusterSSHKeys(user, c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve ssh keys: %v", err)
		}

		dc, found := dcs[c.Spec.Cloud.DatacenterName]
		if !found {
			return nil, fmt.Errorf("unknown cluster datacenter %s", c.Spec.Cloud.DatacenterName)
		}

		client, err := clusterProvider.GetMachineClient(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create a machine client: %v", err)
		}

		node := req.Body
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
			node.Spec.Versions.Kubelet = c.Spec.Version
		}

		if node.Metadata.Name == "" {
			node.Metadata.Name = "kubermatic-" + c.Name + "-" + rand.String(5)
		}

		// Create machine resource
		machine, err := machineresource.Machine(c, &node.Node, dc, keys)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine from template: %v", err)
		}

		// Send machine resource to k8s
		machine, err = client.MachineV1alpha1().Machines().Create(machine)
		if err != nil {
			return nil, fmt.Errorf("failed to create machine: %v", err)
		}

		return outputMachine(machine, nil, false)
	}
}

func getNodeForMachine(machine *v1alpha1.Machine, nodes []corev1.Node) *corev1.Node {
	for _, node := range nodes {
		if (machine.Status.NodeRef != nil && node.UID == machine.Status.NodeRef.UID) || node.Name == machine.Name {
			return &node
		}
	}
	return nil
}

func getMachineForNode(node *corev1.Node, machines []v1alpha1.Machine) *v1alpha1.Machine {
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

func getNodesEndpointV2() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		req := request.(NodesV2Req)
		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			return nil, err
		}

		machineClient, err := clusterProvider.GetMachineClient(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create a machine client: %v", err)
		}

		kubeClient, err := clusterProvider.GetClient(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create a kubernetes client: %v", err)
		}

		machineList, err := machineClient.MachineV1alpha1().Machines().List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to load machines from cluster: %v", err)
		}

		nodeList, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to load nodes from cluster: %v", err)
		}

		//The following is a bit tricky. We might have a node which is not created by a machine and vice versa...
		var apiNodes []*apiv2.Node
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
			apiNodes = append(apiNodes, outNode)
		}

		// Now all nodes, which do not belong to a machine - Relevant for BYO
		for i := range nodeList.Items {
			if !matchedMachineNodes.Has(string(nodeList.Items[i].UID)) {
				apiNodes = append(apiNodes, outputNode(&nodeList.Items[i], req.HideInitialConditions))
			}
		}
		return apiNodes, nil
	}
}

func tryToFindMachineAndNode(name string, machineClient machineclientset.Interface, kubeClient kubernetes.Interface) (*v1alpha1.Machine, *corev1.Node, error) {
	machineList, err := machineClient.MachineV1alpha1().Machines().List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load machines from cluster: %v", err)
	}

	nodeList, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load nodes from cluster: %v", err)
	}

	var node *corev1.Node
	var machine *v1alpha1.Machine

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

	if node == nil && machine == nil {
		return nil, nil, nil
	}

	if node != nil && machine != nil {
		return machine, node, nil
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

func getNodeEndpointV2() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NodeReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			return nil, err
		}

		machineClient, err := clusterProvider.GetMachineClient(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create a machine client: %v", err)
		}

		kubeClient, err := clusterProvider.GetClient(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create a kubernetes client: %v", err)
		}

		machine, node, err := tryToFindMachineAndNode(req.NodeName, machineClient, kubeClient)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, errors.NewNotFound("Node", req.NodeName)
		}

		if machine == nil {
			return outputNode(node, req.HideInitialConditions), nil
		}
		return outputMachine(machine, node, req.HideInitialConditions)
	}
}

func deleteNodeEndpointV2() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NodeReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			return nil, err
		}

		machineClient, err := clusterProvider.GetMachineClient(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create a machine client: %v", err)
		}

		kubeClient, err := clusterProvider.GetClient(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create a kubernetes client: %v", err)
		}

		machine, node, err := tryToFindMachineAndNode(req.NodeName, machineClient, kubeClient)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, errors.NewNotFound("Node", req.NodeName)
		}

		if machine != nil {
			if err := machineClient.MachineV1alpha1().Machines().Delete(machine.Name, nil); err != nil {
			}
		} else if node != nil {
			return nil, kubeClient.CoreV1().Nodes().Delete(node.Name, nil)
		}
		return nil, nil
	}
}
