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

package etcdbackupconfig

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createEtcdBackupConfigReq)

		ebc := &kubermaticv1.EtcdBackupConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Body.Name,
			},
			Spec: req.Body.Spec,
		}

		ebc, err := createEtcdBackupConfig(ctx, userInfoGetter, req.ProjectID, ebc)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToAPIETCDBackupConfig(ebc), nil
	}
}

// createEtcdBackupConfigReq represents a request for creating a cluster etcd backup configuration
// swagger:parameters createEtcdBackupConfig
type createEtcdBackupConfigReq struct {
	cluster.GetClusterReq
	// in: body
	Body ebcBody
}

type ebcBody struct {
	// Name of the etcd backup config
	Name string `json:"name"`
	// EtcdBackupConfigSpec Spec of the etcd backup config
	Spec kubermaticv1.EtcdBackupConfigSpec `json:"spec"`
}

func DecodeCreateEtcdBackupConfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createEtcdBackupConfigReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func convertInternalToAPIETCDBackupConfig(ebc *kubermaticv1.EtcdBackupConfig) *apiv2.EtcdBackupConfig {
	return &apiv2.EtcdBackupConfig{
		Name:   ebc.Name,
		Spec:   ebc.Spec,
		Status: ebc.Status,
	}
}

func createEtcdBackupConfig(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string, etcdBackupConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	adminUserInfo, privilegedEtcdBackupConfigProvider, err := getAdminUserInfoPrivilegedEtcdBackupConfigProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedEtcdBackupConfigProvider.CreateUnsecured(etcdBackupConfig)
	}
	userInfo, etcdBackupConfigProvider, err := getUserInfoEtcdBackupConfigProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return etcdBackupConfigProvider.Create(userInfo, etcdBackupConfig)
}

func getAdminUserInfoPrivilegedEtcdBackupConfigProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter) (*provider.UserInfo, provider.PrivilegedEtcdBackupConfigProvider, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	if !userInfo.IsAdmin {
		return userInfo, nil, nil
	}
	privilegedEtcdBackupConfigProvider := ctx.Value(middleware.PrivilegedEtcdBackupConfigProviderContextKey).(provider.PrivilegedEtcdBackupConfigProvider)
	return userInfo, privilegedEtcdBackupConfigProvider, nil
}

func getUserInfoEtcdBackupConfigProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string) (*provider.UserInfo, provider.EtcdBackupConfigProvider, error) {

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}

	etcdBackupConfigProvider := ctx.Value(middleware.EtcdBackupConfigProviderContextKey).(provider.EtcdBackupConfigProvider)
	return userInfo, etcdBackupConfigProvider, nil
}
