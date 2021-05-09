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

package alertmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"gopkg.in/yaml.v2"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getAlertmanagerReq)
		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		alertmanager, config, err := getAlertmanagerConfig(ctx, userInfoGetter, c, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIAlertmanager(alertmanager, config), nil
	}
}

func UpdateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateAlertmanagerReq)

		if err := req.validateUpdateAlertmanagerReq(); err != nil {
			return nil, utilerrors.NewBadRequest(fmt.Errorf("invalid alertmanager configuration: %w", err).Error())
		}

		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		alertmanager, configSecret := convertAPIToInternalAlertmanager(c, &req.Body)
		al, config, err := updateAlertmanagerConfig(ctx, userInfoGetter, req.ProjectID, alertmanager, configSecret)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIAlertmanager(al, config), nil
	}
}

func ResetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetAlertmanagerReq)

		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		if err := resetAlertmanagerConfig(ctx, userInfoGetter, req.ProjectID, c); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

// getAlertmanagerReq defines HTTP request for getting alertmanager
// swagger:parameters getAlertmanager
type getAlertmanagerReq struct {
	cluster.GetClusterReq
}

// updateAlertmanagerReq defines HTTP request for creating alertmanager
// swagger:parameters updateAlertmanager
type updateAlertmanagerReq struct {
	cluster.GetClusterReq
	// in: body
	// required: true
	Body apiv2.Alertmanager
}

func (req *updateAlertmanagerReq) validateUpdateAlertmanagerReq() error {
	bodyMap := map[string]interface{}{}
	if err := yaml.Unmarshal(req.Body.Spec.Config, &bodyMap); err != nil {
		return fmt.Errorf("can not unmarshal yaml configuration: %w", err)
	}
	return nil
}

// resetAlertmanagerReq defines HTTP request for deleting alertmanager
// swagger:parameters resetAlertmanager
type resetAlertmanagerReq struct {
	cluster.GetClusterReq
}

func DecodeGetAlertmanagerReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getAlertmanagerReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)
	return req, nil
}

func DecodeUpdateAlertmanagerReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateAlertmanagerReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func DecodeResetAlertmanagerReq(c context.Context, r *http.Request) (interface{}, error) {
	var req resetAlertmanagerReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)
	return req, nil
}

func convertAPIToInternalAlertmanager(cluster *kubermaticv1.Cluster, alertmanager *apiv2.Alertmanager) (*kubermaticv1.Alertmanager, *corev1.Secret) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.DefaultAlertmanagerConfigSecretName,
			Namespace: cluster.Status.NamespaceName,
		},
		Data: map[string][]byte{
			resources.AlertmanagerConfigSecretKey: alertmanager.Spec.Config,
		},
	}
	internalAlertmanager := &kubermaticv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.AlertmanagerName,
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.AlertmanagerSpec{
			ConfigSecret: corev1.LocalObjectReference{
				Name: secret.Name,
			},
		},
	}
	return internalAlertmanager, secret
}

func convertInternalToAPIAlertmanager(alertmanager *kubermaticv1.Alertmanager, configSecret *corev1.Secret) *apiv2.Alertmanager {
	return &apiv2.Alertmanager{
		Spec: apiv2.AlertmanagerSpec{
			Config: configSecret.Data[resources.AlertmanagerConfigSecretKey],
		},
	}
}

func getAlertmanagerConfig(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID string) (*kubermaticv1.Alertmanager, *corev1.Secret, error) {
	adminUserInfo, privilegedAlertmanagerProvider, err := getAdminUserInfoPrivilegedAlertmanagerProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedAlertmanagerProvider.GetUnsecured(cluster)
	}
	userInfo, alertmanagerProvider, err := getUserInfoAlertmanagerProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, nil, err
	}
	return alertmanagerProvider.Get(cluster, userInfo)
}

func updateAlertmanagerConfig(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string, alertmanager *kubermaticv1.Alertmanager, config *corev1.Secret) (*kubermaticv1.Alertmanager, *corev1.Secret, error) {
	adminUserInfo, privilegedAlertmanagerProvider, err := getAdminUserInfoPrivilegedAlertmanagerProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedAlertmanagerProvider.UpdateUnsecured(alertmanager, config)
	}
	userInfo, alertmanagerProvider, err := getUserInfoAlertmanagerProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, nil, err
	}
	return alertmanagerProvider.Update(alertmanager, config, userInfo)
}

func resetAlertmanagerConfig(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string, cluster *kubermaticv1.Cluster) error {
	adminUserInfo, privilegedAlertmanagerProvider, err := getAdminUserInfoPrivilegedAlertmanagerProvider(ctx, userInfoGetter)
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		return privilegedAlertmanagerProvider.ResetUnsecured(cluster)
	}
	userInfo, alertmanagerProvider, err := getUserInfoAlertmanagerProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return err
	}
	return alertmanagerProvider.Reset(cluster, userInfo)
}

func getAdminUserInfoPrivilegedAlertmanagerProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter) (*provider.UserInfo, provider.PrivilegedAlertmanagerProvider, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	if !userInfo.IsAdmin {
		return userInfo, nil, nil
	}
	privilegedAlertmanagerProvider := ctx.Value(middleware.PrivilegedAlertmanagerProviderContextKey).(provider.PrivilegedAlertmanagerProvider)
	return userInfo, privilegedAlertmanagerProvider, nil
}

func getUserInfoAlertmanagerProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string) (*provider.UserInfo, provider.AlertmanagerProvider, error) {

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}

	alertmanagerProvider := ctx.Value(middleware.AlertmanagerProviderContextKey).(provider.AlertmanagerProvider)
	return userInfo, alertmanagerProvider, nil
}
