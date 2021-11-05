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

package rulegroupadmin

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ruleGroupNamespace = "mla"
)

func GetEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getReq)
		ruleGroup, err := getRuleGroup(ctx, userInfoGetter, req.RuleGroupID, ruleGroupNamespace)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroup(ruleGroup), nil
	}
}

func ListEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		options := &provider.RuleGroupListOptions{}
		if req.Type != "" {
			options.RuleGroupType = kubermaticv1.RuleGroupType(req.Type)
		}
		ruleGroups, err := listRuleGroups(ctx, userInfoGetter, ruleGroupNamespace, options)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroups(ruleGroups), nil
	}
}

func CreateEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReq)
		groupName, err := req.validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(fmt.Errorf("invalid rule group: %w", err).Error())
		}
		ruleGroup, err := convertAPIToInternalRuleGroup(&req.Body, groupName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		resRuleGroup, err := createRuleGroup(ctx, userInfoGetter, ruleGroup)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroup(resRuleGroup), nil
	}
}

func UpdateEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateReq)
		if err := req.validate(); err != nil {
			return nil, utilerrors.NewBadRequest(fmt.Errorf("invalid rule group: %w", err).Error())
		}
		currentRuleGroup, err := getRuleGroup(ctx, userInfoGetter, req.RuleGroupID, ruleGroupNamespace)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		ruleGroup, err := convertAPIToInternalRuleGroup(&req.Body, req.RuleGroupID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		currentRuleGroup.Spec = ruleGroup.Spec

		resRuleGroup, err := updateRuleGroup(ctx, userInfoGetter, currentRuleGroup)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIRuleGroup(resRuleGroup), nil
	}
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteReq)
		if err := deleteRuleGroup(ctx, userInfoGetter, req.RuleGroupID, ruleGroupNamespace); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func getRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, ruleGroupID, namespace string) (*kubermaticv1.RuleGroup, error) {
	privilegedRuleGroupProvider, err := getPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	return privilegedRuleGroupProvider.GetUnsecured(ruleGroupID, namespace)
}

func listRuleGroups(ctx context.Context, userInfoGetter provider.UserInfoGetter, namespace string, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	privilegedRuleGroupProvider, err := getPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	return privilegedRuleGroupProvider.ListUnsecured(namespace, options)
}

func createRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	privilegedRuleGroupProvider, err := getPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	return privilegedRuleGroupProvider.CreateUnsecured(ruleGroup)
}

func updateRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	privilegedRuleGroupProvider, err := getPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	return privilegedRuleGroupProvider.UpdateUnsecured(ruleGroup)
}

func deleteRuleGroup(ctx context.Context, userInfoGetter provider.UserInfoGetter, ruleGroupID, namespace string) error {
	privilegedRuleGroupProvider, err := getPrivilegedRuleGroupProvider(ctx, userInfoGetter)
	if err != nil {
		return err
	}
	return privilegedRuleGroupProvider.DeleteUnsecured(ruleGroupID, namespace)
}

func convertAPIToInternalRuleGroup(ruleGroup *apiv2.RuleGroup, ruleGroupID string) (*kubermaticv1.RuleGroup, error) {
	internalRuleGroup := &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleGroupID,
			Namespace: ruleGroupNamespace,
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: ruleGroup.Type,
			Cluster: corev1.ObjectReference{
				Kind: kubermaticv1.ClusterKindName,
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

func getPrivilegedRuleGroupProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter) (provider.PrivilegedRuleGroupProvider, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if !userInfo.IsAdmin {
		return nil, utilerrors.NewNotAuthorized()
	}
	privilegedRuleGroupProvider := ctx.Value(middleware.PrivilegedRuleGroupProviderContextKey).(provider.PrivilegedRuleGroupProvider)
	return privilegedRuleGroupProvider, nil
}
