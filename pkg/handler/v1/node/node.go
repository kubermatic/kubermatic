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

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return handlercommon.PatchMachineDeployment(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, sshKeyProvider, seedsGetter, req.ProjectID, req.ClusterID, req.NodeDeploymentID, req.Patch)
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
		if req.Type == handlercommon.MachineDeploymentEventWarningType || req.Type == handlercommon.MachineDeploymentEventNormalType {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query parameter, unsupported type: %s", req.Type)
	}

	return req, nil
}

func ListNodeDeploymentNodesEvents(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeDeploymentNodesEventsReq)
		return handlercommon.ListMachineDeploymentNodesEvents(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.NodeDeploymentID, req.Type)
	}
}
