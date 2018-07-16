package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	machineresource "github.com/kubermatic/kubermatic/api/pkg/resources/machine"
	apierrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
)

func getNodeForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewNodeReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		project, err := projectProvider.Get(user, req.ProjectName)
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

		project, err := projectProvider.Get(user, req.ProjectName)
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

// NewCreateNodeReq defines HTTP request for newCreateNodeForCluster
// swagger:parameters newCreateNodeForCluster
type NewCreateNodeReq struct {
	NewGetClusterReq
	// in: body
	Body apiv1.Node
}

func decodeCreateNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req NewCreateNodeReq

	clusterName, projectName, err := decodeClusterNameAndProject(c, r)
	if err != nil {
		return nil, err
	}
	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectName = projectName
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

	clusterName, projectName, err := decodeClusterNameAndProject(c, r)
	if err != nil {
		return nil, err
	}
	nodeName, ok := mux.Vars(r)["node_name"]
	if !ok {
		return nil, fmt.Errorf("'node_name' parameter is required but was not provided")
	}

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectName = projectName
	req.ClusterName = clusterName
	req.NodeName = nodeName
	req.DCReq = dcr.(DCReq)

	return req, nil
}
