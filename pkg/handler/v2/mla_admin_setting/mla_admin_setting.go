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

package mla_admin_setting

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

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
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.NewNotAuthorized()
		}
		privilegedMLAAdminSettingProvider := ctx.Value(middleware.PrivilegedMLAAdminSettingProviderContextKey).(provider.PrivilegedMLAAdminSettingProvider)
		mlaAdminSetting, err := privilegedMLAAdminSettingProvider.GetUnsecured(c)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIMLAAdminSetting(mlaAdminSetting), nil
	}
}

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReq)
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.NewNotAuthorized()
		}
		privilegedMLAAdminSettingProvider := ctx.Value(middleware.PrivilegedMLAAdminSettingProviderContextKey).(provider.PrivilegedMLAAdminSettingProvider)
		mlaAdminSetting, err := convertAPIToInternalMLAAdminSetting(c, &req.Body)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		resMLAAdminSetting, err := privilegedMLAAdminSettingProvider.CreateUnsecured(mlaAdminSetting)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIMLAAdminSetting(resMLAAdminSetting), nil
	}
}

func UpdateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateReq)
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.NewNotAuthorized()
		}
		privilegedMLAAdminSettingProvider := ctx.Value(middleware.PrivilegedMLAAdminSettingProviderContextKey).(provider.PrivilegedMLAAdminSettingProvider)
		currentMLAAdminSetting, err := privilegedMLAAdminSettingProvider.GetUnsecured(c)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		newMLAAdminSetting, err := convertAPIToInternalMLAAdminSetting(c, &req.Body)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		currentMLAAdminSetting.Spec = newMLAAdminSetting.Spec
		resMLAAdminSetting, err := privilegedMLAAdminSettingProvider.UpdateUnsecured(currentMLAAdminSetting)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIMLAAdminSetting(resMLAAdminSetting), nil
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
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.NewNotAuthorized()
		}
		privilegedMLAAdminSettingProvider := ctx.Value(middleware.PrivilegedMLAAdminSettingProviderContextKey).(provider.PrivilegedMLAAdminSettingProvider)
		if err = privilegedMLAAdminSettingProvider.DeleteUnsecured(c); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func convertAPIToInternalMLAAdminSetting(cluster *kubermaticv1.Cluster, mlaAdminSetting *apiv2.MLAAdminSetting) (*kubermaticv1.MLAAdminSetting, error) {
	internalMLAAdminSetting := &kubermaticv1.MLAAdminSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.DefaultMLAAdminSettingName,
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.MLAAdminSettingSpec{
			ClusterName:          cluster.Name,
			MonitoringRateLimits: mlaAdminSetting.MonitoringRateLimits,
			LoggingRateLimits:    mlaAdminSetting.LoggingRateLimits,
		},
	}
	return internalMLAAdminSetting, nil
}

func convertInternalToAPIMLAAdminSetting(mlaAdminSetting *kubermaticv1.MLAAdminSetting) *apiv2.MLAAdminSetting {
	return &apiv2.MLAAdminSetting{
		MonitoringRateLimits: mlaAdminSetting.Spec.MonitoringRateLimits,
		LoggingRateLimits:    mlaAdminSetting.Spec.LoggingRateLimits,
	}
}
