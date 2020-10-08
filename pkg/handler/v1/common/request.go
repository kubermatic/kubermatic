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

package common

import (
	"context"
	"fmt"
	"net/http"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"

	"github.com/gorilla/mux"
)

func DecodeEmptyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req struct{}
	return req, nil
}

// ProjectReq represents a request for project-specific data
type ProjectReq struct {
	// in: path
	// required: true
	ProjectID string `json:"project_id"`
}

// GetProjectID returns the ID of a requested project
func (pr ProjectReq) GetProjectID() string {
	return pr.ProjectID
}

func DecodeProjectRequest(c context.Context, r *http.Request) (interface{}, error) {
	return ProjectReq{
		ProjectID: mux.Vars(r)["project_id"],
	}, nil
}

// ProjectIDGetter knows how to get project ID from the request
type ProjectIDGetter interface {
	GetProjectID() string
}

// GetProjectRq defines HTTP request for getProject endpoint
// swagger:parameters getProject getUsersForProject listClustersForProject listServiceAccounts listClustersV2
type GetProjectRq struct {
	ProjectReq
}

func DecodeGetProject(c context.Context, r *http.Request) (interface{}, error) {
	projectReq, err := DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	return GetProjectRq{projectReq.(ProjectReq)}, nil
}

// DCReq represent a request for datacenter specific data in a given project
type DCReq struct {
	ProjectReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// GetSeedCluster returns the SeedCluster object
func (req DCReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.DC,
	}
}

func DecodeDcReq(c context.Context, r *http.Request) (interface{}, error) {
	projectReq, err := DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	return DCReq{
		DC:         mux.Vars(r)["dc"],
		ProjectReq: projectReq.(ProjectReq),
	}, nil
}

// GetClusterReq defines HTTP request for deleteCluster and getClusterKubeconfig endpoints
// swagger:parameters getCluster getClusterKubeconfig getOidcClusterKubeconfig listAWSSizesNoCredentials getClusterHealth getClusterUpgrades getClusterMetrics getClusterNodeUpgrades listGCPZonesNoCredentials listGCPNetworksNoCredentials listAWSZonesNoCredentials listAWSSubnetsNoCredentials listAlibabaInstanceTypesNoCredentials listNamespace
type GetClusterReq struct {
	DCReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeGetClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GetClusterReq
	clusterID, err := DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

func DecodeClusterID(c context.Context, r *http.Request) (string, error) {
	clusterID := mux.Vars(r)["cluster_id"]
	if clusterID == "" {
		return "", fmt.Errorf("'cluster_id' parameter is required but was not provided")
	}

	return clusterID, nil
}

// UserIDGetter knows how to get user ID from the request
type UserIDGetter interface {
	GetUserID() string
}

func DecodeSSHKeyID(c context.Context, r *http.Request) (string, error) {
	keyID := mux.Vars(r)["key_id"]
	if keyID == "" {
		return "", fmt.Errorf("'key_id' parameter is required but was not provided")
	}

	return keyID, nil
}
