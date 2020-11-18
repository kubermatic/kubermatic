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
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	UserClusterBindingComponentValue = "userClusterBinding"
)

func BindUserToRoleEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(roleUserReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID, Namespace: req.Namespace}, &rbacv1.Role{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		roleBindingList := &rbacv1.RoleBindingList{}
		if err := client.List(ctx, roleBindingList, ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: UserClusterBindingComponentValue}, ctrlruntimeclient.InNamespace(req.Namespace)); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var existingRoleBinding *rbacv1.RoleBinding
		for _, roleBinding := range roleBindingList.Items {
			if roleBinding.RoleRef.Name == req.RoleID {
				existingRoleBinding = roleBinding.DeepCopy()
				break
			}
		}

		if existingRoleBinding == nil {
			existingRoleBinding, err = generateRBACRoleBinding(ctx, client, req.Namespace, req.RoleID)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}

		roleUser := req.Body

		oldBinding := existingRoleBinding.DeepCopy()
		for _, subject := range existingRoleBinding.Subjects {
			if roleUser.UserEmail != "" && subject.Name == roleUser.UserEmail {
				return nil, errors.NewBadRequest("user %s already connected to role %s", roleUser.UserEmail, req.RoleID)
			}
			if roleUser.Group != "" && subject.Name == roleUser.Group {
				return nil, errors.NewBadRequest("group %s already connected to role %s", roleUser.Group, req.RoleID)
			}
		}

		if roleUser.UserEmail != "" {
			existingRoleBinding.Subjects = append(existingRoleBinding.Subjects,
				rbacv1.Subject{
					Kind:     rbacv1.UserKind,
					APIGroup: rbacv1.GroupName,
					Name:     roleUser.UserEmail,
				})
		}
		if roleUser.Group != "" {
			existingRoleBinding.Subjects = append(existingRoleBinding.Subjects,
				rbacv1.Subject{
					Kind:     rbacv1.GroupKind,
					APIGroup: rbacv1.GroupName,
					Name:     roleUser.Group,
				})
		}

		if err := client.Patch(ctx, existingRoleBinding, ctrlruntimeclient.MergeFrom(oldBinding)); err != nil {
			return nil, fmt.Errorf("failed to update role binding: %v", err)
		}

		return convertInternalRoleBindingToExternal(existingRoleBinding), nil
	}
}

func UnbindUserFromRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(roleUserReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID, Namespace: req.Namespace}, &rbacv1.Role{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		roleBindingList := &rbacv1.RoleBindingList{}
		if err := client.List(ctx, roleBindingList, ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: UserClusterBindingComponentValue}, ctrlruntimeclient.InNamespace(req.Namespace)); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var existingRoleBinding *rbacv1.RoleBinding
		for _, roleBinding := range roleBindingList.Items {
			if roleBinding.RoleRef.Name == req.RoleID {
				existingRoleBinding = roleBinding.DeepCopy()
				break
			}
		}

		if existingRoleBinding == nil {
			return nil, errors.NewBadRequest("the role binding not found in namespace %s", req.Namespace)
		}

		roleUser := req.Body
		binding := existingRoleBinding.DeepCopy()
		var newSubjects []rbacv1.Subject
		for _, subject := range binding.Subjects {
			if roleUser.UserEmail != "" && subject.Name == roleUser.UserEmail {
				continue
			}
			if roleUser.Group != "" && subject.Name == roleUser.Group {
				continue
			}
			newSubjects = append(newSubjects, subject)
		}
		binding.Subjects = newSubjects

		if err := client.Update(ctx, binding); err != nil {
			return nil, fmt.Errorf("failed to update role binding: %v", err)
		}

		return convertInternalRoleBindingToExternal(binding), nil
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
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		roleBindingList := &rbacv1.RoleBindingList{}
		if err := client.List(
			ctx,
			roleBindingList,
			ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: UserClusterBindingComponentValue},
		); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalRoleBindingsToExternal(roleBindingList.Items), nil
	}
}

func BindUserToClusterRoleEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterRoleUserReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID}, &rbacv1.ClusterRole{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
		if err := client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: UserClusterBindingComponentValue}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var existingClusterRoleBinding *rbacv1.ClusterRoleBinding
		for _, clusterRoleBinding := range clusterRoleBindingList.Items {
			if clusterRoleBinding.RoleRef.Name == req.RoleID {
				existingClusterRoleBinding = clusterRoleBinding.DeepCopy()
				break
			}
		}

		if existingClusterRoleBinding == nil {
			return nil, fmt.Errorf("the cluster role binding not found")
		}

		clusterRoleUser := req.Body

		oldBinding := existingClusterRoleBinding.DeepCopy()
		for _, subject := range existingClusterRoleBinding.Subjects {
			if clusterRoleUser.UserEmail != "" && subject.Name == clusterRoleUser.UserEmail {
				return nil, errors.NewBadRequest("user %s already connected to role %s", clusterRoleUser.UserEmail, req.RoleID)
			}
			if clusterRoleUser.Group != "" && subject.Name == clusterRoleUser.Group {
				return nil, errors.NewBadRequest("group %s already connected to role %s", clusterRoleUser.Group, req.RoleID)
			}
		}

		if clusterRoleUser.UserEmail != "" {
			existingClusterRoleBinding.Subjects = append(existingClusterRoleBinding.Subjects,
				rbacv1.Subject{
					Kind:     rbacv1.UserKind,
					APIGroup: rbacv1.GroupName,
					Name:     clusterRoleUser.UserEmail,
				})
		}
		if clusterRoleUser.Group != "" {
			existingClusterRoleBinding.Subjects = append(existingClusterRoleBinding.Subjects,
				rbacv1.Subject{
					Kind:     rbacv1.GroupKind,
					APIGroup: rbacv1.GroupName,
					Name:     clusterRoleUser.Group,
				})
		}

		if err := client.Patch(ctx, existingClusterRoleBinding, ctrlruntimeclient.MergeFrom(oldBinding)); err != nil {
			return nil, fmt.Errorf("failed to update cluster role binding: %v", err)
		}

		return convertInternalClusterRoleBindingToExternal(existingClusterRoleBinding), nil
	}
}

func UnbindUserFromClusterRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterRoleUserReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest("invalid request: %v", err)
		}

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: req.RoleID}, &rbacv1.ClusterRole{}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
		if err := client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: UserClusterBindingComponentValue}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var existingClusterRoleBinding *rbacv1.ClusterRoleBinding
		for _, clusterRoleBinding := range clusterRoleBindingList.Items {
			if clusterRoleBinding.RoleRef.Name == req.RoleID {
				existingClusterRoleBinding = clusterRoleBinding.DeepCopy()
				break
			}
		}

		if existingClusterRoleBinding == nil {
			return nil, errors.NewBadRequest("the cluster role binding not found")
		}

		clusterRoleUser := req.Body
		binding := existingClusterRoleBinding.DeepCopy()
		var newSubjects []rbacv1.Subject
		for _, subject := range binding.Subjects {
			if clusterRoleUser.UserEmail != "" && subject.Name == clusterRoleUser.UserEmail {
				continue
			}
			if clusterRoleUser.Group != "" && subject.Name == clusterRoleUser.Group {
				continue
			}
			newSubjects = append(newSubjects, subject)
		}
		binding.Subjects = newSubjects

		if err := client.Update(ctx, binding); err != nil {
			return nil, fmt.Errorf("failed to update cluster role binding: %v", err)
		}

		return convertInternalClusterRoleBindingToExternal(binding), nil
	}
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

func ListClusterRoleBindingEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listBindingReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
		if err := client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: UserClusterBindingComponentValue}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterRoleBindingsToExternal(clusterRoleBindingList.Items), nil
	}
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

// generateRBACRoleBinding creates role binding
func generateRBACRoleBinding(ctx context.Context, client ctrlruntimeclient.Client, namespace, roleName string) (*rbacv1.RoleBinding, error) {

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s:%s", rand.String(10), roleName),
			Labels:    map[string]string{handlercommon.UserClusterComponentKey: UserClusterBindingComponentValue},
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{},
	}

	if err := client.Create(ctx, roleBinding); err != nil {
		return nil, err
	}

	return roleBinding, nil
}

func convertInternalRoleBindingToExternal(clusterRole *rbacv1.RoleBinding) *apiv1.RoleBinding {
	roleBinding := &apiv1.RoleBinding{
		Namespace:   clusterRole.Namespace,
		RoleRefName: clusterRole.RoleRef.Name,
		Subjects:    clusterRole.Subjects,
	}

	return roleBinding
}

func convertInternalRoleBindingsToExternal(roleBindings []rbacv1.RoleBinding) []*apiv1.RoleBinding {
	var apiRoleBinding []*apiv1.RoleBinding
	for _, binding := range roleBindings {
		apiRoleBinding = append(apiRoleBinding, convertInternalRoleBindingToExternal(binding.DeepCopy()))
	}

	return apiRoleBinding
}

func convertInternalClusterRoleBindingToExternal(clusterRoleBinding *rbacv1.ClusterRoleBinding) *apiv1.ClusterRoleBinding {
	binding := &apiv1.ClusterRoleBinding{
		RoleRefName: clusterRoleBinding.RoleRef.Name,
		Subjects:    clusterRoleBinding.Subjects,
	}

	return binding
}

func convertInternalClusterRoleBindingsToExternal(clusterRoleBindings []rbacv1.ClusterRoleBinding) []*apiv1.ClusterRoleBinding {
	var apiClusterRoleBinding []*apiv1.ClusterRoleBinding
	for _, binding := range clusterRoleBindings {
		apiClusterRoleBinding = append(apiClusterRoleBinding, convertInternalClusterRoleBindingToExternal(binding.DeepCopy()))
	}

	return apiClusterRoleBinding
}
