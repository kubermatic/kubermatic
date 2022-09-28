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
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func BindUserToRoleEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RoleUserReq)

		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.BindUserToRoleEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID, req.Namespace)
	}
}

func UnbindUserFromRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RoleUserReq)

		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.UnbindUserFromRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID, req.Namespace)
	}
}

func ListRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listBindingReq)
		return handlercommon.ListRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

func BindUserToClusterRoleEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterRoleUserReq)

		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.BindUserToClusterRoleEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID)
	}
}

func UnbindUserFromClusterRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterRoleUserReq)

		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest("invalid request: %v", err)
		}

		return handlercommon.UnbindUserFromClusterRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.RoleID)
	}
}

func ListClusterRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listBindingReq)
		return handlercommon.ListClusterRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

// Validate validates roleUserReq request.
func (req RoleUserReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if req.Body.UserEmail == "" && req.Body.Group == "" && req.Body.ServiceAccount == "" && req.Body.ServiceAccountNamespace == "" {
		return fmt.Errorf("either user email, group or service account must be set")
	}

	if req.Body.UserEmail != "" && (req.Body.Group != "" || req.Body.ServiceAccount != "" || req.Body.ServiceAccountNamespace != "") {
		return fmt.Errorf("user email can not be used in conjunction with group or service account")
	}

	if req.Body.Group != "" && (req.Body.ServiceAccount != "" || req.Body.ServiceAccountNamespace != "") {
		return fmt.Errorf("group can not be used in conjunction with email or service account")
	}

	if (req.Body.ServiceAccount == "" && req.Body.ServiceAccountNamespace != "") || (req.Body.ServiceAccount != "" && req.Body.ServiceAccountNamespace == "") {
		return fmt.Errorf("both service account and service account namespace must be defined")
	}
	return nil
}

// RoleUserReq defines HTTP request for bindUserToRole endpoint
// swagger:parameters bindUserToRoleV2 unbindUserFromRoleBindingV2
type RoleUserReq struct {
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

// GetSeedCluster returns the SeedCluster object.
func (req RoleUserReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeRoleUserReq(c context.Context, r *http.Request) (interface{}, error) {
	var req RoleUserReq
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

// listBindingReq defines HTTP request for listClusterRoleBinding endpoint
// swagger:parameters listClusterRoleBindingV2 listRoleBindingV2
type listBindingReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

// GetSeedCluster returns the SeedCluster object.
func (req listBindingReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeListBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listBindingReq
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

	return req, nil
}

// Validate validates clusterRoleUserReq request.
func (req ClusterRoleUserReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if req.Body.UserEmail == "" && req.Body.Group == "" && req.Body.ServiceAccount == "" && req.Body.ServiceAccountNamespace == "" {
		return fmt.Errorf("either user email or group or service account must be set")
	}

	if req.Body.UserEmail != "" && (req.Body.Group != "" || req.Body.ServiceAccount != "" || req.Body.ServiceAccountNamespace != "") {
		return fmt.Errorf("user email can not be used in conjunction with group or service account")
	}

	if req.Body.Group != "" && (req.Body.ServiceAccount != "" || req.Body.ServiceAccountNamespace != "") {
		return fmt.Errorf("group can not be used in conjunction with email or service account")
	}

	if (req.Body.ServiceAccount == "" && req.Body.ServiceAccountNamespace != "") || (req.Body.ServiceAccount != "" && req.Body.ServiceAccountNamespace == "") {
		return fmt.Errorf("both service account and service account namespace must be defined")
	}
	return nil
}

// ClusterRoleUserReq defines HTTP request for bindUserToClusterRoleV2 endpoint
// swagger:parameters bindUserToClusterRoleV2 unbindUserFromClusterRoleBindingV2
type ClusterRoleUserReq struct {
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

// GetSeedCluster returns the SeedCluster object.
func (req ClusterRoleUserReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeClusterRoleUserReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterRoleUserReq
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
