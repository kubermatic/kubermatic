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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"

	rbacv1 "k8s.io/api/rbac/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	UserClusterComponentKey       = "component"
	UserClusterRoleComponentValue = "userClusterRole"
	UserClusterRoleLabelSelector  = "component=userClusterRole"
)

func ListClusterRoleEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := clusterProvider.Get(userInfo, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterRoleList := &rbacv1.ClusterRoleList{}
	if err := client.List(ctx, clusterRoleList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterRoleComponentValue}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertInternalClusterRolesToExternal(clusterRoleList), nil
}

func ListClusterRoleNamesEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := clusterProvider.Get(userInfo, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterRoleList := &rbacv1.ClusterRoleList{}
	if err := client.List(ctx, clusterRoleList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterRoleComponentValue}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var clusterRoleNames []apiv1.ClusterRoleName
	for _, clusterRole := range clusterRoleList.Items {
		clusterRoleNames = append(clusterRoleNames, apiv1.ClusterRoleName{
			Name: clusterRole.Name,
		})
	}
	return clusterRoleNames, nil
}

func ListRoleEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := clusterProvider.Get(userInfo, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	roleList := &rbacv1.RoleList{}
	if err := client.List(ctx, roleList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterRoleComponentValue}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertInternalRolesToExternal(roleList), nil
}

func ListRoleNamesEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	cluster, err := clusterProvider.Get(userInfo, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	roleList := &rbacv1.RoleList{}
	if err := client.List(ctx, roleList, ctrlruntimeclient.MatchingLabels{UserClusterComponentKey: UserClusterRoleComponentValue}); err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertRoleNames(roleList), nil
}

func convertRoleNames(internalRoles *rbacv1.RoleList) []*apiv1.RoleName {
	var apiRoleName []*apiv1.RoleName
	roleMap := map[string][]string{}
	if internalRoles != nil {
		for _, role := range internalRoles.Items {
			if roleMap[role.Name] == nil {
				roleMap[role.Name] = []string{}
			}
			roleMap[role.Name] = append(roleMap[role.Name], role.Namespace)
		}
	}
	for name, namespaces := range roleMap {
		apiRoleName = append(apiRoleName, &apiv1.RoleName{
			Name:      name,
			Namespace: namespaces,
		})
	}
	return apiRoleName
}

func convertInternalClusterRolesToExternal(internalClusterRoles *rbacv1.ClusterRoleList) []*apiv1.ClusterRole {
	var apiClusterRole []*apiv1.ClusterRole
	if internalClusterRoles != nil {
		for _, clusterRole := range internalClusterRoles.Items {
			apiClusterRole = append(apiClusterRole, convertInternalClusterRoleToExternal(clusterRole.DeepCopy()))
		}
	}
	return apiClusterRole
}

func convertInternalClusterRoleToExternal(clusterRole *rbacv1.ClusterRole) *apiv1.ClusterRole {
	return &apiv1.ClusterRole{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                clusterRole.Name,
			Name:              clusterRole.Name,
			CreationTimestamp: apiv1.NewTime(clusterRole.CreationTimestamp.Time),
		},
		Rules: clusterRole.Rules,
	}
}

func convertInternalRolesToExternal(internalRoles *rbacv1.RoleList) []*apiv1.Role {
	var apiClusterRole []*apiv1.Role
	if internalRoles != nil {
		for _, clusterRole := range internalRoles.Items {
			apiClusterRole = append(apiClusterRole, convertInternalRoleToExternal(clusterRole.DeepCopy()))
		}
	}
	return apiClusterRole
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
