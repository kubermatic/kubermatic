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
	"strings"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
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
		if roleUser.UserEmail != "" && strings.EqualFold(subject.Name, roleUser.UserEmail) {
			return nil, utilerrors.NewBadRequest("user %s already connected to role %s", roleUser.UserEmail, roleID)
		}
		if roleUser.Group != "" && subject.Name == roleUser.Group {
			return nil, utilerrors.NewBadRequest("group %s already connected to role %s", roleUser.Group, roleID)
		}
		if roleUser.ServiceAccount != "" && subject.Name == roleUser.ServiceAccount && subject.Namespace == roleUser.ServiceAccountNamespace {
			return nil, utilerrors.NewBadRequest("service account %s/%s already connected to the role %s", roleUser.ServiceAccountNamespace, roleUser.ServiceAccount, roleID)
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
	if roleUser.ServiceAccount != "" {
		existingRoleBinding.Subjects = append(existingRoleBinding.Subjects,
			rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				APIGroup:  "",
				Name:      roleUser.ServiceAccount,
				Namespace: roleUser.ServiceAccountNamespace,
			})
	}

	if err := client.Patch(ctx, existingRoleBinding, ctrlruntimeclient.MergeFrom(oldBinding)); err != nil {
		return nil, fmt.Errorf("failed to update role binding: %w", err)
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
		if clusterRoleUser.UserEmail != "" && strings.EqualFold(subject.Name, clusterRoleUser.UserEmail) {
			return nil, utilerrors.NewBadRequest("user %s already connected to the cluster role %s", clusterRoleUser.UserEmail, roleID)
		}
		if clusterRoleUser.Group != "" && subject.Name == clusterRoleUser.Group {
			return nil, utilerrors.NewBadRequest("group %s already connected to the cluster role %s", clusterRoleUser.Group, roleID)
		}
		if clusterRoleUser.ServiceAccount != "" && subject.Name == clusterRoleUser.ServiceAccount && subject.Namespace == clusterRoleUser.ServiceAccountNamespace {
			return nil, utilerrors.NewBadRequest("service account %s/%s already connected to the cluster role %s", clusterRoleUser.ServiceAccountNamespace, clusterRoleUser.ServiceAccount, roleID)
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

	if clusterRoleUser.ServiceAccount != "" {
		existingClusterRoleBinding.Subjects = append(existingClusterRoleBinding.Subjects,
			rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				APIGroup:  "",
				Name:      clusterRoleUser.ServiceAccount,
				Namespace: clusterRoleUser.ServiceAccountNamespace,
			})
	}

	if err := client.Patch(ctx, existingClusterRoleBinding, ctrlruntimeclient.MergeFrom(oldBinding)); err != nil {
		return nil, fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return convertInternalClusterRoleBindingToExternal(existingClusterRoleBinding), nil
}

func UnbindUserFromRoleBindingEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, roleUser apiv1.RoleUser, projectID, clusterID, roleID, namespace string) (interface{}, error) {
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
		return nil, utilerrors.NewBadRequest("the role binding not found in namespace %s", namespace)
	}

	binding := existingRoleBinding.DeepCopy()
	var newSubjects []rbacv1.Subject
	for _, subject := range binding.Subjects {
		if roleUser.UserEmail != "" && subject.Name == roleUser.UserEmail {
			continue
		}
		if roleUser.Group != "" && subject.Name == roleUser.Group {
			continue
		}
		if roleUser.ServiceAccount != "" && subject.Name == roleUser.ServiceAccount && subject.Namespace == roleUser.ServiceAccountNamespace {
			continue
		}
		newSubjects = append(newSubjects, subject)
	}
	binding.Subjects = newSubjects

	if err := client.Update(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to update role binding: %w", err)
	}

	return convertInternalRoleBindingToExternal(binding), nil
}

func UnbindUserFromClusterRoleBindingEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterRoleUser apiv1.ClusterRoleUser, projectID, clusterID, roleID string) (interface{}, error) {
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
		return nil, utilerrors.NewBadRequest("the cluster role binding not found")
	}

	binding := existingClusterRoleBinding.DeepCopy()
	var newSubjects []rbacv1.Subject
	for _, subject := range binding.Subjects {
		if clusterRoleUser.UserEmail != "" && subject.Name == clusterRoleUser.UserEmail {
			continue
		}
		if clusterRoleUser.Group != "" && subject.Name == clusterRoleUser.Group {
			continue
		}
		if clusterRoleUser.ServiceAccount != "" && subject.Name == clusterRoleUser.ServiceAccount && subject.Namespace == clusterRoleUser.ServiceAccountNamespace {
			continue
		}
		newSubjects = append(newSubjects, subject)
	}
	binding.Subjects = newSubjects

	if err := client.Update(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return convertInternalClusterRoleBindingToExternal(binding), nil
}

// UnbindServiceAccountFromRoles unbinds the service account from all rolebindings labelled UserClusterComponentKey = UserClusterBindingComponentValue.
func UnbindServiceAccountFromRoles(ctx context.Context, client ctrlruntimeclient.Client, serviceAccount *corev1.ServiceAccount) error {
	roleBindingList := &rbacv1.RoleBindingList{}
	if err := client.List(ctx, roleBindingList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterBindingComponentValue}); err != nil {
		return fmt.Errorf("failed to list rolebinding: %w", err)
	}

	for _, roleBinding := range roleBindingList.Items {
		shouldUpate := false
		var newSubjects []rbacv1.Subject
		for _, subject := range roleBinding.Subjects {
			if subject.Name == serviceAccount.Name && subject.Namespace == serviceAccount.Namespace {
				shouldUpate = true
				continue
			}
			newSubjects = append(newSubjects, subject)
		}
		if shouldUpate {
			binding := roleBinding.DeepCopy()
			binding.Subjects = newSubjects
			if err := client.Update(ctx, binding); err != nil {
				return fmt.Errorf("failed to unbind service account '%s/%s' from role binding '%s/%s': %w", serviceAccount.Namespace, serviceAccount.Name, roleBinding.Namespace, roleBinding.Name, err)
			}
		}
	}
	return nil
}

// UnbindServiceAccountFromClusterRoles unbinds the service account from all clusterRolebindings labelled UserClusterComponentKey = UserClusterBindingComponentValue.
func UnbindServiceAccountFromClusterRoles(ctx context.Context, client ctrlruntimeclient.Client, serviceAccount *corev1.ServiceAccount) error {
	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	if err := client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterBindingComponentValue}); err != nil {
		return fmt.Errorf("failed to list clusterRoleBinding: %w", err)
	}

	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		shouldUpate := false
		var newSubjects []rbacv1.Subject
		for _, subject := range clusterRoleBinding.Subjects {
			if subject.Name == serviceAccount.Name && subject.Namespace == serviceAccount.Namespace {
				shouldUpate = true
				continue
			}
			newSubjects = append(newSubjects, subject)
		}
		if shouldUpate {
			binding := clusterRoleBinding.DeepCopy()
			binding.Subjects = newSubjects
			if err := client.Update(ctx, binding); err != nil {
				return fmt.Errorf("failed to unbind service account '%s/%s' from cluster role binding '%s': %w", serviceAccount.Namespace, serviceAccount.Name, clusterRoleBinding.Name, err)
			}
		}
	}
	return nil
}
func ListRoleBindingEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	roleBindingList := &rbacv1.RoleBindingList{}
	if err := client.List(
		ctx,
		roleBindingList,
		ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterBindingComponentValue},
	); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertInternalRoleBindingsToExternal(roleBindingList.Items), nil
}

func ListClusterRoleBindingEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	if err := client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterBindingComponentValue}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertInternalClusterRoleBindingsToExternal(clusterRoleBindingList.Items), nil
}

func convertInternalClusterRoleBindingsToExternal(clusterRoleBindings []rbacv1.ClusterRoleBinding) []*apiv1.ClusterRoleBinding {
	var apiClusterRoleBinding []*apiv1.ClusterRoleBinding
	for _, binding := range clusterRoleBindings {
		apiClusterRoleBinding = append(apiClusterRoleBinding, convertInternalClusterRoleBindingToExternal(binding.DeepCopy()))
	}

	return apiClusterRoleBinding
}

func convertInternalRoleBindingsToExternal(roleBindings []rbacv1.RoleBinding) []*apiv1.RoleBinding {
	var apiRoleBinding []*apiv1.RoleBinding
	for _, binding := range roleBindings {
		apiRoleBinding = append(apiRoleBinding, convertInternalRoleBindingToExternal(binding.DeepCopy()))
	}

	return apiRoleBinding
}

func convertInternalRoleBindingToExternal(clusterRole *rbacv1.RoleBinding) *apiv1.RoleBinding {
	roleBinding := &apiv1.RoleBinding{
		Namespace:   clusterRole.Namespace,
		RoleRefName: clusterRole.RoleRef.Name,
		Subjects:    clusterRole.Subjects,
	}

	return roleBinding
}

// generateRBACRoleBinding creates role binding.
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
