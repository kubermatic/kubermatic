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

func DecodeRoleUserReq(c context.Context, r *http.Request) (interface{}, error) {
	var req roleUserReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)
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

func ListRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listBindingReq)
		return handlercommon.ListRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
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

func ListClusterRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listBindingReq)
		return handlercommon.ListClusterRoleBindingEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

// Validate validates roleUserReq request
func (r roleUserReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the project ID and datacenter cannot be empty")
	}
	if r.Body.UserEmail == "" && r.Body.Group == "" {
		return fmt.Errorf("either user email or group must be set")
	}
	return nil
}

// roleUserReq defines HTTP request for bindUserToRole endpoint
// swagger:parameters bindUserToRole unbindUserFromRoleBinding
type roleUserReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
	// in: body
	Body apiv1.RoleUser
}

// Validate validates clusterRoleUserReq request
func (r clusterRoleUserReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the project ID and datacenter cannot be empty")
	}
	if r.Body.UserEmail == "" && r.Body.Group == "" {
		return fmt.Errorf("either user email or group must be set")
	}

	return nil
}

// clusterRoleUserReq defines HTTP request for bindUserToClusterRole endpoint
// swagger:parameters bindUserToClusterRole unbindUserFromClusterRoleBinding
type clusterRoleUserReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: body
	Body apiv1.ClusterRoleUser
}

func DecodeClusterRoleUserReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clusterRoleUserReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)
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

// listBindingReq defines HTTP request for listClusterRoleBinding endpoint
// swagger:parameters listClusterRoleBinding listRoleBinding
type listBindingReq struct {
	common.GetClusterReq
}

func DecodeListBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listBindingReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)
	req.ClusterID = clusterID

	return req, nil
}
