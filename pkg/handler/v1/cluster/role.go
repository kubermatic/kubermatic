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
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateClusterRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClusterRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		userClusterAPIRole := req.Body

		clusterRole, err := generateRBACClusterRole(userClusterAPIRole.Name, userClusterAPIRole.Rules)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster role: %v", err)
		}

		if err := client.Create(ctx, clusterRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalClusterRoleToExternal(clusterRole), nil
	}
}

func CreateRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		apiRole := req.Body

		role, err := generateRBACRole(apiRole.Name, apiRole.Namespace, apiRole.Rules)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster role: %v", err)
		}

		if err := client.Create(ctx, role); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalRoleToExternal(role), nil
	}
}

// createClusterRoleReq defines HTTP request for createClusterRole endpoint
// swagger:parameters createClusterRole
type createClusterRoleReq struct {
	common.DCReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: body
	Body apiv1.ClusterRole
}

// createRoleReq defines HTTP request for createRole endpoint
// swagger:parameters createRole
type createRoleReq struct {
	common.DCReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: body
	Body apiv1.Role
}

// Validate validates createRoleReq request
func (r createRoleReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the project ID and datacenter cannot be empty")
	}

	if r.Body.Namespace == "" || r.Body.Name == "" {
		return fmt.Errorf("the request Body name and namespace cannot be empty")
	}
	return nil
}

// Validate validates createRoleReq request
func (r createClusterRoleReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.DC) == 0 {
		return fmt.Errorf("the project ID and datacenter cannot be empty")
	}

	if r.Body.Name == "" {
		return fmt.Errorf("the request Body name cannot be empty")
	}
	return nil
}

func DecodeCreateClusterRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createClusterRoleReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DecodeCreateRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createRoleReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func ListClusterRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListClusterRoleEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func ListClusterRoleNamesEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListClusterRoleNamesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func ListRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListRoleEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func ListRoleNamesEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListRoleNamesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

// listReq defines HTTP request for listClusterRole and listRole endpoint
// swagger:parameters listClusterRole listRole listRoleNames listClusterRoleNames
type listReq struct {
	common.DCReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeListClusterRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}

// GetClusterRoleEndpoint gets ClusterRole with given name.
func GetClusterRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getClusterRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRole := &rbacv1.ClusterRole{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID}, clusterRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalClusterRoleToExternal(clusterRole), nil
	}
}

func GetRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		role := &rbacv1.Role{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID, Namespace: req.Namespace}, role); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalRoleToExternal(role), nil
	}
}

// DeleteClusterRoleEndpoint deletes ClusterRole with given name
func DeleteClusterRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getClusterRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRole := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.RoleID,
			},
		}
		if err := client.Delete(ctx, clusterRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

// DeleteRoleEndpoint deletes Role with given name
func DeleteRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.RoleID,
				Namespace: req.Namespace,
			},
		}
		if err := client.Delete(ctx, role); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

// getRoleReq defines HTTP request for getRole endpoint
// swagger:parameters getRole deleteRole
type getRoleReq struct {
	common.DCReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
}

func DecodeGetRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getRoleReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

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
	return req, nil
}

// getClusterRoleReq defines HTTP request for getClusterRole endpoint
// swagger:parameters getClusterRole deleteClusterRole
type getClusterRoleReq struct {
	common.DCReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: path
	// required: true
	RoleID string `json:"role_id"`
}

func DecodeGetClusterRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getClusterRoleReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	roleID := mux.Vars(r)["role_id"]
	if roleID == "" {
		return "", fmt.Errorf("'role_id' parameter is required but was not provided")
	}
	req.RoleID = roleID

	return req, nil
}

// PatchRoleEndpoint patches Role with given name
func PatchRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingRole := &rbacv1.Role{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID, Namespace: req.Namespace}, existingRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingRoleJSON, err := json.Marshal(existingRole)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing role: %v", err)
		}

		patchedRoleJSON, err := jsonpatch.MergePatch(existingRoleJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch role: %v", err)
		}

		var patchedRole *rbacv1.Role
		err = json.Unmarshal(patchedRoleJSON, &patchedRole)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched role: %v", err)
		}

		if err := client.Update(ctx, patchedRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalRoleToExternal(patchedRole), nil
	}
}

// patchRoleReq defines HTTP request for patchRole endpoint
// swagger:parameters patchRole
type patchRoleReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: path
	// required: true
	Namespace string `json:"namespace"`
	// in: body
	Patch json.RawMessage
}

func DecodePatchRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchRoleReq
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

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// PatchRoleEndpoint patches ClusterRole with given name
func PatchClusterRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchClusterRoleReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingClusterRole := &rbacv1.ClusterRole{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID}, existingClusterRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingClusterRoleJSON, err := json.Marshal(existingClusterRole)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing cluster role: %v", err)
		}

		patchedClusterRoleJSON, err := jsonpatch.MergePatch(existingClusterRoleJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch cluster role: %v", err)
		}

		var patchedClusterRole *rbacv1.ClusterRole
		err = json.Unmarshal(patchedClusterRoleJSON, &patchedClusterRole)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched role: %v", err)
		}

		if err := client.Update(ctx, patchedClusterRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterRoleToExternal(patchedClusterRole), nil
	}
}

// patchClusterRoleReq defines HTTP request for patchClusterRole endpoint
// swagger:parameters patchClusterRole
type patchClusterRoleReq struct {
	common.GetClusterReq
	// in: path
	// required: true
	RoleID string `json:"role_id"`
	// in: body
	Patch json.RawMessage
}

func DecodePatchClusterRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchClusterRoleReq
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

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// generateRBACClusterRole creates cluster role
func generateRBACClusterRole(name string, rules []rbacv1.PolicyRule) (*rbacv1.ClusterRole, error) {
	if rules == nil {
		return nil, fmt.Errorf("the policy rule can not be nil")
	}
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterRoleComponentValue},
		},
		Rules: rules,
	}
	return clusterRole, nil
}

// generateRBACRole creates role
func generateRBACRole(name, namespace string, rules []rbacv1.PolicyRule) (*rbacv1.Role, error) {
	if rules == nil {
		return nil, fmt.Errorf("the policy rule can not be nil")
	}
	clusterRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{handlercommon.UserClusterComponentKey: handlercommon.UserClusterRoleComponentValue},
			Namespace: namespace,
		},
		Rules: rules,
	}
	return clusterRole, nil
}

func convertInternalClusterRoleToExternal(clusterRole *rbacv1.ClusterRole) *apiv1.ClusterRole {
	return &apiv1.ClusterRole{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                clusterRole.Name,
			Name:              clusterRole.Name,
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(clusterRole.CreationTimestamp.Time),
		},
		Rules: clusterRole.Rules,
	}
}

func convertInternalRoleToExternal(clusterRole *rbacv1.Role) *apiv1.Role {
	return &apiv1.Role{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                clusterRole.Name,
			Name:              clusterRole.Name,
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(clusterRole.CreationTimestamp.Time),
		},
		Namespace: clusterRole.Namespace,
		Rules:     clusterRole.Rules,
	}
}
