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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
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

func BindUserToRoleEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, roleUser apiv1.RoleUser, projectID, clusterID, roleID, namespace string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: roleID, Namespace: namespace}, &rbacv1.Role{}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	roleBindingList := &rbacv1.RoleBindingList{}
	if err := client.List(ctx, roleBindingList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterBindingComponentValue}, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var existingRoleBinding *rbacv1.RoleBinding
	for _, roleBinding := range roleBindingList.Items {
		if roleBinding.RoleRef.Name == roleID {
			existingRoleBinding = roleBinding.DeepCopy()
			break
		}
	}

	if existingRoleBinding == nil {
		existingRoleBinding, err = generateRBACRoleBinding(ctx, client, namespace, roleID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
	}

	oldBinding := existingRoleBinding.DeepCopy()
	for _, subject := range existingRoleBinding.Subjects {
		if roleUser.UserEmail != "" && subject.Name == roleUser.UserEmail {
			return nil, errors.NewBadRequest("user %s already connected to role %s", roleUser.UserEmail, roleID)
		}
		if roleUser.Group != "" && subject.Name == roleUser.Group {
			return nil, errors.NewBadRequest("group %s already connected to role %s", roleUser.Group, roleID)
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

func BindUserToClusterRoleEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterRoleUser apiv1.ClusterRoleUser, projectID, clusterID, roleID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: roleID}, &rbacv1.ClusterRole{}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	if err := client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterBindingComponentValue}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var existingClusterRoleBinding *rbacv1.ClusterRoleBinding
	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		if clusterRoleBinding.RoleRef.Name == roleID {
			existingClusterRoleBinding = clusterRoleBinding.DeepCopy()
			break
		}
	}

	if existingClusterRoleBinding == nil {
		return nil, fmt.Errorf("the cluster role binding not found")
	}

	oldBinding := existingClusterRoleBinding.DeepCopy()
	for _, subject := range existingClusterRoleBinding.Subjects {
		if clusterRoleUser.UserEmail != "" && subject.Name == clusterRoleUser.UserEmail {
			return nil, errors.NewBadRequest("user %s already connected to the cluster role %s", clusterRoleUser.UserEmail, roleID)
		}
		if clusterRoleUser.Group != "" && subject.Name == clusterRoleUser.Group {
			return nil, errors.NewBadRequest("group %s already connected to the cluster role %s", clusterRoleUser.Group, roleID)
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

func convertInternalRoleBindingToExternal(clusterRole *rbacv1.RoleBinding) *apiv1.RoleBinding {
	roleBinding := &apiv1.RoleBinding{
		Namespace:   clusterRole.Namespace,
		RoleRefName: clusterRole.RoleRef.Name,
		Subjects:    clusterRole.Subjects,
	}

	return roleBinding
}

// generateRBACRoleBinding creates role binding
func generateRBACRoleBinding(ctx context.Context, client ctrlruntimeclient.Client, namespace, roleName string) (*rbacv1.RoleBinding, error) {

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s:%s", rand.String(10), roleName),
			Labels:    map[string]string{UserClusterComponentKey: UserClusterBindingComponentValue},
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

func convertInternalClusterRoleBindingToExternal(clusterRoleBinding *rbacv1.ClusterRoleBinding) *apiv1.ClusterRoleBinding {
	binding := &apiv1.ClusterRoleBinding{
		RoleRefName: clusterRoleBinding.RoleRef.Name,
		Subjects:    clusterRoleBinding.Subjects,
	}

	return binding
}
