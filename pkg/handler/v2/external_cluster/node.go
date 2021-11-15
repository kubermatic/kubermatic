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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
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
		var nodesV1 []*apiv2.ExternalClusterNode

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

func ListMachineDeploymentEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMachineDeploymentsReq)
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

		var machineDeployments []apiv2.ExternalClusterMachineDeployment
		machineDeployments = make([]apiv2.ExternalClusterMachineDeployment, 0)

		cloud := cluster.Spec.CloudSpec
		if cloud != nil {
			secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

			if cloud.GKE != nil {
				np, err := getGKENodePools(ctx, cluster, secretKeySelector, cloud.GKE.CredentialsReference, clusterProvider)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				machineDeployments = np
			}
			if cloud.EKS != nil {
				np, err := getEKSNodePools(cluster, secretKeySelector, cloud, clusterProvider)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				machineDeployments = np
			}
		}

		return machineDeployments, nil
	}
}

func ListMachineDeploymentNodesEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMachineDeploymentNodesReq)
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

		var nodes []apiv2.ExternalClusterNode
		nodes = make([]apiv2.ExternalClusterNode, 0)

		cloud := cluster.Spec.CloudSpec
		if cloud != nil {
			secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

			if cloud.GKE != nil {
				n, err := getGKENodes(ctx, cluster, req.MachineDeploymentID, secretKeySelector, cloud.GKE.CredentialsReference, clusterProvider)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				nodes = n
			}
			if cloud.EKS != nil {
				n, err := getEKSNodes(cluster, req.MachineDeploymentID, secretKeySelector, cloud.EKS.CredentialsReference, clusterProvider)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				nodes = n
			}
		}

		return nodes, nil
	}
}

func DeleteMachineDeploymentEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(machineDeploymentReq)
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

		cloud := cluster.Spec.CloudSpec
		if cloud != nil {
			secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

			if cloud.GKE != nil {
				err := deleteGKENodePool(ctx, cluster, req.MachineDeploymentID, secretKeySelector, cloud.GKE.CredentialsReference, clusterProvider)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
			}
			if cloud.EKS != nil {
				err := deleteEKSNodePool(cluster, req.MachineDeploymentID, secretKeySelector, cloud.EKS.CredentialsReference, clusterProvider)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
			}
		}

		return nil, nil
	}
}

// Validate validates getMachineDeploymentReq request
func (req machineDeploymentReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	if len(req.MachineDeploymentID) == 0 {
		return fmt.Errorf("the machine deployment ID cannot be empty")
	}
	return nil
}

func DecodeGetMachineDeploymentReq(c context.Context, r *http.Request) (interface{}, error) {
	var req machineDeploymentReq

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

	machineDeploymentID, err := decodeMachineDeploymentID(c, r)
	if err != nil {
		return nil, err
	}
	req.MachineDeploymentID = machineDeploymentID

	return req, nil
}

// listMachineDeploymentNodesReq defines HTTP request for listExternalClusterMachineDeploymentNodes
// swagger:parameters listExternalClusterMachineDeploymentNodes
type listMachineDeploymentNodesReq struct {
	machineDeploymentReq
}

func DecodeListMachineDeploymentNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listMachineDeploymentNodesReq

	rawReq, err := DecodeGetMachineDeploymentReq(c, r)
	if err != nil {
		return nil, err
	}
	getReq := rawReq.(machineDeploymentReq)
	req.machineDeploymentReq = getReq
	return req, nil
}

func decodeMachineDeploymentID(c context.Context, r *http.Request) (string, error) {
	machineDeploymentID := mux.Vars(r)["machinedeployment_id"]
	if machineDeploymentID == "" {
		return "", fmt.Errorf("'machinedeployment_id' parameter is required but was not provided")
	}

	return machineDeploymentID, nil
}

// listMachineDeploymentsReq defines HTTP request for listExternalClusterMachineDeployments
// swagger:parameters listExternalClusterMachineDeployments
type listMachineDeploymentsReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

// Validate validates ListMachineDeploymentEndpoint request
func (req listMachineDeploymentsReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func DecodeListMachineDeploymentReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listMachineDeploymentsReq

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

func outputNode(node corev1.Node) (*apiv2.ExternalClusterNode, error) {
	displayName := node.Name
	nodeStatus := apiv1.NodeStatus{}
	nodeStatus = apiNodeStatus(nodeStatus, node)

	nodeStatus.ErrorReason = strings.TrimSuffix(nodeStatus.ErrorReason, errGlue)
	nodeStatus.ErrorMessage = strings.TrimSuffix(nodeStatus.ErrorMessage, errGlue)

	return &apiv2.ExternalClusterNode{
		Node: apiv1.Node{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                node.Name,
				Name:              displayName,
				CreationTimestamp: apiv1.NewTime(node.CreationTimestamp.Time),
			},
			Spec: apiv1.NodeSpec{
				Versions: apiv1.NodeVersionInfo{
					Kubelet: node.Status.NodeInfo.KubeletVersion,
				},
				Labels: node.Labels,
			},
			Status: nodeStatus,
		},
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

func GetMachineDeploymentEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(machineDeploymentReq)
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

		var machineDeployment apiv2.ExternalClusterMachineDeployment

		cloud := cluster.Spec.CloudSpec
		if cloud != nil {
			secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

			if cloud.EKS != nil {
				np, err := getEKSNodePool(cluster, req.MachineDeploymentID, secretKeySelector, cloud, clusterProvider)
				if err != nil {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				machineDeployment = *np
			}
		}

		return machineDeployment, nil
	}
}

func PatchMachineDeploymentEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(patchMachineDeploymentReq)

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

		cloud := cluster.Spec.CloudSpec
		if cloud != nil {

			secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

			if cloud.EKS != nil {
				mdToPatch := apiv2.ExternalClusterMachineDeployment{}
				patchedMD := apiv2.ExternalClusterMachineDeployment{}

				md, err := getEKSNodePool(cluster, req.MachineDeploymentID, secretKeySelector, cloud, clusterProvider)
				if err != nil {
					return nil, err
				}
				mdToPatch.NodeDeployment = md.NodeDeployment
				if err := patchMD(&mdToPatch, &patchedMD, req.Patch); err != nil {
					return nil, err
				}
				return patchEKSMD(&mdToPatch, &patchedMD, secretKeySelector, cloud)
			}
		}
		return nil, fmt.Errorf("unsupported or missing cloud provider fields")
	}
}

func patchMD(mdToPatch, patchedMD *apiv2.ExternalClusterMachineDeployment, patchJson json.RawMessage) error {

	existingMDJSON, err := json.Marshal(mdToPatch)
	if err != nil {
		return errors.NewBadRequest("cannot decode existing md: %v", err)
	}
	patchedMDJSON, err := jsonpatch.MergePatch(existingMDJSON, patchJson)
	if err != nil {
		return errors.NewBadRequest("cannot patch md: %v, %v", err, patchJson)
	}
	err = json.Unmarshal(patchedMDJSON, &patchedMD)
	if err != nil {
		return errors.NewBadRequest("cannot decode patched md: %v", err)
	}
	return nil
}

// machineDeploymentReq defines HTTP request for getExternalClusterMachineDeployment deleteExternalClusterMachineDeployment
// swagger:parameters getExternalClusterMachineDeployment deleteExternalClusterMachineDeployment
type machineDeploymentReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	MachineDeploymentID string `json:"machinedeployment_id"`
}

// patchMachineDeploymentReq defines HTTP request for patchExternalClusterMachineDeployments endpoint
// swagger:parameters patchExternalClusterMachineDeployments
type patchMachineDeploymentReq struct {
	machineDeploymentReq
	// in: body
	Patch json.RawMessage
}

func DecodePatchMachineDeploymentReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchMachineDeploymentReq

	rawMachineDeployment, err := DecodeGetMachineDeploymentReq(c, r)
	if err != nil {
		return nil, err
	}
	md := rawMachineDeployment.(machineDeploymentReq)
	req.ClusterID = md.ClusterID
	req.ProjectID = md.ProjectID
	req.MachineDeploymentID = md.MachineDeploymentID

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}
