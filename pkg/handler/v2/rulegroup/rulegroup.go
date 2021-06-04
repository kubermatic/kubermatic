/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package rulegroup

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getReq)
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		ruleGroup, err := getRuleGroup(ctx, userInfoGetter, c, req.ProjectID, req.RuleGroupID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroup(ruleGroup), nil
	}
}

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		options := &provider.RuleGroupListOptions{}
		if req.Type != "" {
			options.RuleGroupType = kubermaticv1.RuleGroupType(req.Type)
		}
		ruleGroups, err := listRuleGroups(ctx, userInfoGetter, c, req.ProjectID, options)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroups(ruleGroups), nil
	}
}

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReq)
		groupName, err := req.validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(fmt.Errorf("invalid rule group: %w", err).Error())
		}
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		ruleGroup, err := convertAPIToInternalRuleGroup(c, &req.Body, groupName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		resRuleGroup, err := createRuleGroup(ctx, userInfoGetter, req.ProjectID, ruleGroup)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroup(resRuleGroup), nil
	}
}

func UpdateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateReq)
		if err := req.validate(); err != nil {
			return nil, utilerrors.NewBadRequest(fmt.Errorf("invalid rule group: %w", err).Error())
		}
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		currentRuleGroup, err := getRuleGroup(ctx, userInfoGetter, c, req.ProjectID, req.RuleGroupID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		ruleGroup, err := convertAPIToInternalRuleGroup(c, &req.Body, req.RuleGroupID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		currentRuleGroup.Spec = ruleGroup.Spec

		resRuleGroup, err := updateRuleGroup(ctx, userInfoGetter, req.ProjectID, currentRuleGroup)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroup(resRuleGroup), nil
	}
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteReq)
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		if err = deleteRuleGroup(ctx, userInfoGetter, c, req.ProjectID, req.RuleGroupID); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func getRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID, ruleGroupID string) (*kubermaticv1.RuleGroup, error) {
	adminUserInfo, privilegedRuleGroupProvider, err := getAdminUserInfoPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedRuleGroupProvider.GetUnsecured(cluster, ruleGroupID)
	}
	userInfo, alertmanagerProvider, err := getUserInfoRuleGroupProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return alertmanagerProvider.Get(userInfo, cluster, ruleGroupID)
}

func listRuleGroups(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID string, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	adminUserInfo, privilegedRuleGroupProvider, err := getAdminUserInfoPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedRuleGroupProvider.ListUnsecured(cluster, options)
	}
	userInfo, alertmanagerProvider, err := getUserInfoRuleGroupProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return alertmanagerProvider.List(userInfo, cluster, options)
}

func createRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	adminUserInfo, privilegedRuleGroupProvider, err := getAdminUserInfoPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedRuleGroupProvider.CreateUnsecured(ruleGroup)
	}
	userInfo, alertmanagerProvider, err := getUserInfoRuleGroupProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return alertmanagerProvider.Create(userInfo, ruleGroup)
}

func updateRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	adminUserInfo, privilegedRuleGroupProvider, err := getAdminUserInfoPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedRuleGroupProvider.UpdateUnsecured(ruleGroup)
	}
	userInfo, alertmanagerProvider, err := getUserInfoRuleGroupProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return alertmanagerProvider.Update(userInfo, ruleGroup)
}

func deleteRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID, ruleGroupID string) error {
	adminUserInfo, privilegedRuleGroupProvider, err := getAdminUserInfoPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		return privilegedRuleGroupProvider.DeleteUnsecured(cluster, ruleGroupID)
	}
	userInfo, alertmanagerProvider, err := getUserInfoRuleGroupProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return err
	}
	return alertmanagerProvider.Delete(userInfo, cluster, ruleGroupID)
}

func convertAPIToInternalRuleGroup(cluster *kubermaticv1.Cluster, ruleGroup *apiv2.RuleGroup, ruleGroupID string) (*kubermaticv1.RuleGroup, error) {
	internalRuleGroup := &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleGroupID,
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: ruleGroup.Type,
			Cluster: corev1.ObjectReference{
				Kind:            kubermaticv1.ClusterKindName,
				Namespace:       "",
				Name:            cluster.Name,
				UID:             cluster.UID,
				APIVersion:      cluster.APIVersion,
				ResourceVersion: cluster.ResourceVersion,
			},
			Data: ruleGroup.Data,
		},
	}
	return internalRuleGroup, nil
}

func convertInternalToAPIRuleGroups(ruleGroups []*kubermaticv1.RuleGroup) []*apiv2.RuleGroup {
	var apiRuleGroups []*apiv2.RuleGroup
	for _, ruleGroup := range ruleGroups {
		apiRuleGroups = append(apiRuleGroups, convertInternalToAPIRuleGroup(ruleGroup))
	}
	return apiRuleGroups
}

func convertInternalToAPIRuleGroup(ruleGroup *kubermaticv1.RuleGroup) *apiv2.RuleGroup {
	return &apiv2.RuleGroup{
		Data: ruleGroup.Spec.Data,
		Type: ruleGroup.Spec.RuleGroupType,
	}
}

func getAdminUserInfoPrivilegedRuleGroupProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter) (*provider.UserInfo, provider.PrivilegedRuleGroupProvider, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	if !userInfo.IsAdmin {
		return userInfo, nil, nil
	}
	privilegedRuleGroupProvider := ctx.Value(middleware.PrivilegedRuleGroupProviderContextKey).(provider.PrivilegedRuleGroupProvider)
	return userInfo, privilegedRuleGroupProvider, nil
}

func getUserInfoRuleGroupProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string) (*provider.UserInfo, provider.RuleGroupProvider, error) {

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}

	ruleGroupProvider := ctx.Value(middleware.RuleGroupProviderContextKey).(provider.RuleGroupProvider)
	return userInfo, ruleGroupProvider, nil
}
