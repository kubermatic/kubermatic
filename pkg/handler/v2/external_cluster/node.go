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
	"net/http"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

const errGlue = " & "

func ListNodesEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listNodesReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		var nodesV1 []*apiv1.Node

		nodes, err := clusterProvider.ListNodes(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, n := range nodes.Items {
			outNode, err := outputNode(n)
			if err != nil {
				return nil, fmt.Errorf("failed to output node %s: %v", n.Name, err)
			}
			nodesV1 = append(nodesV1, outNode)
		}
		return nodesV1, nil
	}
}

func ListNodesMetricsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(listNodesReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		nodeMetrics := make([]apiv1.NodeMetric, 0)

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		isMetricServer, err := clusterProvider.IsMetricServerAvailable(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if isMetricServer {
			client, err := clusterProvider.GetClient(cluster)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			nodes := &corev1.NodeList{}
			if err := client.List(ctx, nodes); err != nil {
				return nil, err
			}
			availableResources := make(map[string]corev1.ResourceList)
			for _, n := range nodes.Items {
				availableResources[n.Name] = n.Status.Allocatable
			}

			nodeDeploymentNodesMetrics := make([]v1beta1.NodeMetrics, 0)
			allNodeMetricsList := &v1beta1.NodeMetricsList{}
			if err := client.List(ctx, allNodeMetricsList); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			for _, m := range allNodeMetricsList.Items {
				if _, ok := availableResources[m.Name]; ok {
					nodeDeploymentNodesMetrics = append(nodeDeploymentNodesMetrics, m)
				}
			}
			return handlercommon.ConvertNodeMetrics(nodeDeploymentNodesMetrics, availableResources)
		}

		return nodeMetrics, nil
	}
}

// listNodesReq defines HTTP request for listExternalClusterNodes
// swagger:parameters listExternalClusterNodes listExternalClusterNodesMetrics
type listNodesReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeListNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listNodesReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	return req, nil
}

// Validate validates CreateEndpoint request
func (req listNodesReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func GetNodeEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getNodeReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		node, err := clusterProvider.GetNode(cluster, req.NodeID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return outputNode(*node)
	}
}

// getNodeReq defines HTTP request for getExternalClusterNode
// swagger:parameters getExternalClusterNode
type getNodeReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: path
	// required: true
	NodeID string `json:"node_id"`
}

func DecodeGetNodeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getNodeReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	req.NodeID = mux.Vars(r)["node_id"]

	return req, nil
}

// Validate validates CreateEndpoint request
func (req getNodeReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	if len(req.NodeID) == 0 {
		return fmt.Errorf("the node ID cannot be empty")
	}
	return nil
}

func outputNode(node corev1.Node) (*apiv1.Node, error) {
	displayName := node.Name
	nodeStatus := apiv1.NodeStatus{}
	nodeStatus = apiNodeStatus(nodeStatus, node)

	nodeStatus.ErrorReason = strings.TrimSuffix(nodeStatus.ErrorReason, errGlue)
	nodeStatus.ErrorMessage = strings.TrimSuffix(nodeStatus.ErrorMessage, errGlue)

	return &apiv1.Node{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                node.Name,
			Name:              displayName,
			CreationTimestamp: apiv1.NewTime(node.CreationTimestamp.Time),
		},
		Spec: apiv1.MachineSpec{
			Versions: apiv1.MachineVersionInfo{
				Kubelet: node.Status.NodeInfo.KubeletVersion,
			},
			Labels: node.Labels,
		},
		Status: nodeStatus,
	}, nil
}

func apiNodeStatus(status apiv1.NodeStatus, inputNode corev1.Node) apiv1.NodeStatus {
	for _, address := range inputNode.Status.Addresses {
		status.Addresses = append(status.Addresses, apiv1.NodeAddress{
			Type:    string(address.Type),
			Address: address.Address,
		})
	}

	reason, message := parseNodeConditions(inputNode)
	status.ErrorReason += reason
	status.ErrorMessage += message

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

func parseNodeConditions(node corev1.Node) (reason string, message string) {
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
