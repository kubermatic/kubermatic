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

		c, userInfo, alertmanagerProvider, err := getClusterUserInfoAlertmanagerProvider(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		alertmanager, config, err := alertmanagerProvider.Get(c, userInfo)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIAlertmanager(alertmanager, config), nil
	}
}

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createAlertmanagerReq)

		if err := req.validateCreateAlertmanagerReq(); err != nil {
			return nil, utilerrors.NewBadRequest(fmt.Errorf("invalid alertmanager configuration: %w", err).Error())
		}

		c, userInfo, alertmanagerProvider, err := getClusterUserInfoAlertmanagerProvider(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		alertmanager, configSecret := convertAPIToInternalAlertmanager(c, &req.Body)
		al, config, err := alertmanagerProvider.Create(alertmanager, configSecret, userInfo)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalToAPIAlertmanager(al, config), nil
	}
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteAlertmanagerReq)

		c, userInfo, alertmanagerProvider, err := getClusterUserInfoAlertmanagerProvider(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		if err := alertmanagerProvider.Delete(c, userInfo); err != nil {
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

// createAlertmanagerReq defines HTTP request for creating alertmanager
// swagger:parameters createAlertmanager
type createAlertmanagerReq struct {
	cluster.GetClusterReq
	// in: body
	// required: true
	Body apiv2.Alertmanager
}

func (req *createAlertmanagerReq) validateCreateAlertmanagerReq() error {
	bodyMap := map[string]interface{}{}
	if err := yaml.Unmarshal(req.Body.Spec.Config, &bodyMap); err != nil {
		return fmt.Errorf("can not unmarshal yaml configuration: %w", err)
	}
	return nil
}

// deleteAlertmanagerReq defines HTTP request for deleting alertmanager
// swagger:parameters deleteAlertmanager
type deleteAlertmanagerReq struct {
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

func DecodeCreateAlertmanagerReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createAlertmanagerReq

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

func DecodeDeleteAlertmanagerReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteAlertmanagerReq

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

func getClusterUserInfoAlertmanagerProvider(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (*kubermaticv1.Cluster, *provider.UserInfo, provider.AlertmanagerProvider, error) {
	c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, nil, nil, err
	}

	alertmanagerProvider := ctx.Value(middleware.AlertmanagerProviderContextKey).(provider.AlertmanagerProvider)
	return c, userInfo, alertmanagerProvider, nil
}
