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

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func BindUserToRoleEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(roleUserReq)

		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.BindUserToRoleEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID, req.Namespace)
	}
}

func UnbindUserFromRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(roleUserReq)

		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.UnbindUserFromRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID, req.Namespace)
	}
}

// Validate validates roleUserReq request
func (req roleUserReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if req.Body.UserEmail == "" && req.Body.Group == "" {
		return fmt.Errorf("either user email or group must be set")
	}
	return nil
}

// roleUserReq defines HTTP request for bindUserToRole endpoint
// swagger:parameters bindUserToRoleV2 unbindUserFromRoleBindingV2
type roleUserReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
	// in: body
	Body apiv1.RoleUser
}

// GetSeedCluster returns the SeedCluster object
func (req roleUserReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeRoleUserReq(c context.Context, r *http.Request) (interface{}, error) {
	var req roleUserReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = projectReq.(common.ProjectReq)
	req.ClusterID = clusterID

	roleID := mux.Vars(r)["role_id"]
	if roleID == "" {
		return "", fmt.Errorf("'role_id' parameter is required but was not provided")
	}
	req.RoleID = roleID
	namespace := mux.Vars(r)["namespace"]
	if namespace == "" {
		return "", fmt.Errorf("'namespace' parameter is required but was not provided")
	}
	req.Namespace = namespace

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func BindUserToClusterRoleEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterRoleUserReq)

		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.BindUserToClusterRoleEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID)
	}
}

func UnbindUserFromClusterRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterRoleUserReq)

		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.UnbindUserFromClusterRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID)
	}
}

// Validate validates clusterRoleUserReq request
func (req clusterRoleUserReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if req.Body.UserEmail == "" && req.Body.Group == "" {
		return fmt.Errorf("either user email or group must be set")
	}

	return nil
}

// clusterRoleUserReq defines HTTP request for bindUserToClusterRoleV2 endpoint
// swagger:parameters bindUserToClusterRoleV2 unbindUserFromClusterRoleBindingV2
type clusterRoleUserReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: body
	Body apiv1.ClusterRoleUser
}

// GetSeedCluster returns the SeedCluster object
func (req clusterRoleUserReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeClusterRoleUserReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clusterRoleUserReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = projectReq.(common.ProjectReq)
	req.ClusterID = clusterID

	roleID := mux.Vars(r)["role_id"]
	if roleID == "" {
		return "", fmt.Errorf("'role_id' parameter is required but was not provided")
	}
	req.RoleID = roleID

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}
