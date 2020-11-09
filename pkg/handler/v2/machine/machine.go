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

package machine

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func CreateMachineDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createMachineDeploymentReq)
		return handlercommon.CreateMachineDeployment(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, sshKeyProvider, seedsGetter, req.Body, req.ProjectID, req.ClusterID)
	}
}

// createMachineDeploymentReq defines HTTP request for createMachineDeployment
// swagger:parameters createMachineDeployment
type createMachineDeploymentReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: body
	Body apiv1.NodeDeployment
}

func DecodeCreateMachineDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req createMachineDeploymentReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req createMachineDeploymentReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DeleteMachineDeploymentNode(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteMachineDeploymentNodeReq)
		return handlercommon.DeleteMachineNode(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.NodeID)
	}
}

// deleteMachineDeploymentNodeReq defines HTTP request for deleteMachineDeploymentNode
// swagger:parameters deleteMachineDeploymentNode
type deleteMachineDeploymentNodeReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	NodeID string `json:"node_id"`
}

func DecodeDeleteMachineDeploymentNode(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteMachineDeploymentNodeReq

	nodeID := mux.Vars(r)["node_id"]
	if nodeID == "" {
		return "", fmt.Errorf("'node_id' parameter is required but was not provided")
	}

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)
	req.NodeID = nodeID

	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req deleteMachineDeploymentNodeReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// listMachineDeploymentsReq defines HTTP request for listMachineDeployments
// swagger:parameters listMachineDeployments
type listMachineDeploymentsReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

func DecodeListMachineDeployments(c context.Context, r *http.Request) (interface{}, error) {
	var req listMachineDeploymentsReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req listMachineDeploymentsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func ListMachineDeployments(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMachineDeploymentsReq)
		return handlercommon.ListMachineDeployments(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

func GetMachineDeployment(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(machineDeploymentReq)
		return handlercommon.GetMachineDeployment(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.MachineDeploymentID)
	}
}

// GetSeedCluster returns the SeedCluster object
func (req machineDeploymentReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// machineDeploymentReq defines HTTP request for getMachineDeployment
// swagger:parameters getMachineDeployment
type machineDeploymentReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	MachineDeploymentID string `json:"machinedeployment_id"`
}

func decodeMachineDeploymentID(c context.Context, r *http.Request) (string, error) {
	machineDeploymentID := mux.Vars(r)["machinedeployment_id"]
	if machineDeploymentID == "" {
		return "", fmt.Errorf("'machinedeployment_id' parameter is required but was not provided")
	}

	return machineDeploymentID, nil
}

func DecodeGetMachineDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req machineDeploymentReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	machineDeploymentID, err := decodeMachineDeploymentID(c, r)
	if err != nil {
		return nil, err
	}
	req.MachineDeploymentID = machineDeploymentID

	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req machineDeploymentNodesReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// machineDeploymentNodesReq defines HTTP request for listMachineDeploymentNodes
// swagger:parameters listMachineDeploymentNodes
type machineDeploymentNodesReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	MachineDeploymentID string `json:"machinedeployment_id"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func DecodeListMachineDeploymentNodes(c context.Context, r *http.Request) (interface{}, error) {
	var req machineDeploymentNodesReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	machineDeploymentID, err := decodeMachineDeploymentID(c, r)
	if err != nil {
		return nil, err
	}
	req.MachineDeploymentID = machineDeploymentID

	hideInitialConditions := r.URL.Query().Get("hideInitialConditions")
	if strings.EqualFold(hideInitialConditions, "true") {
		req.HideInitialConditions = true
	}

	return req, nil
}

func ListMachineDeploymentNodes(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(machineDeploymentNodesReq)
		return handlercommon.ListMachineDeploymentNodes(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.MachineDeploymentID, req.HideInitialConditions)
	}
}

// listNodesForClusterReq defines HTTP request for listNodesForCluster
// swagger:parameters listNodesForCluster
type listNodesForClusterReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

// GetSeedCluster returns the SeedCluster object
func (req listNodesForClusterReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeListNodesForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req listNodesForClusterReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	req.HideInitialConditions, _ = strconv.ParseBool(r.URL.Query().Get("hideInitialConditions"))

	return req, nil
}

func ListNodesForCluster(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listNodesForClusterReq)
		return handlercommon.ListNodesForCluster(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.HideInitialConditions)
	}
}

// machineDeploymentMetricsReq defines HTTP request for listMachineDeploymentMetrics
// swagger:parameters listMachineDeploymentMetrics
type machineDeploymentMetricsReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: path
	MachineDeploymentID string `json:"machinedeployment_id"`
}

// GetSeedCluster returns the SeedCluster object
func (req machineDeploymentMetricsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeListMachineDeploymentMetrics(c context.Context, r *http.Request) (interface{}, error) {
	var req machineDeploymentMetricsReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	machineDeploymentID, err := decodeMachineDeploymentID(c, r)
	if err != nil {
		return nil, err
	}
	req.MachineDeploymentID = machineDeploymentID

	return req, nil
}

func ListMachineDeploymentMetrics(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(machineDeploymentMetricsReq)
		return handlercommon.ListMachineDeploymentMetrics(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.MachineDeploymentID)
	}
}

// patchMachineDeploymentReq defines HTTP request for patchMachineDeployment endpoint
// swagger:parameters patchMachineDeployment
type patchMachineDeploymentReq struct {
	machineDeploymentReq

	// in: body
	Patch json.RawMessage
}

func DecodePatchMachineDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req patchMachineDeploymentReq

	rawMachineDeployment, err := DecodeGetMachineDeployment(c, r)
	if err != nil {
		return nil, err
	}
	md := rawMachineDeployment.(machineDeploymentReq)
	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}
	req.MachineDeploymentID = md.MachineDeploymentID
	req.ClusterID = md.ClusterID
	req.ProjectID = md.ProjectID

	return req, nil
}

func PatchMachineDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchMachineDeploymentReq)
		return handlercommon.PatchMachineDeployment(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, sshKeyProvider, seedsGetter, req.ProjectID, req.ClusterID, req.MachineDeploymentID, req.Patch)
	}
}
