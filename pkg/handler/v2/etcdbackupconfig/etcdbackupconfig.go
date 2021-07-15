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
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createEtcdBackupConfigReq)

		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		if err := req.validate(); err != nil {
			return nil, err
		}

		ebc, err := convertAPIToInternalEtcdBackupConfig(req.Body.Name, &req.Body.Spec, c)
		if err != nil {
			return nil, err
		}

		ebc, err = createEtcdBackupConfig(ctx, userInfoGetter, req.ProjectID, ebc)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToAPIEtcdBackupConfig(ebc), nil
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
	Spec apiv2.EtcdBackupConfigSpec `json:"spec"`
}

func (r *createEtcdBackupConfigReq) validate() error {
	// schedule and keep have to either be both set, or both absent
	if (r.Body.Spec.Keep == nil && r.Body.Spec.Schedule != "") || (r.Body.Spec.Keep != nil && r.Body.Spec.Schedule == "") {
		return errors.NewBadRequest("EtcdBackupConfig has to have both Schedule and Keep options set or both empty")
	}
	return nil
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

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEtcdBackupConfigReq)

		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		ebc, err := getEtcdBackupConfig(ctx, userInfoGetter, c, req.ProjectID, req.EtcdBackupConfigName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToAPIEtcdBackupConfig(ebc), nil
	}
}

// getEtcdBackupConfigReq represents a request for getting a cluster etcd backup configuration
// swagger:parameters getEtcdBackupConfig
type getEtcdBackupConfigReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	EtcdBackupConfigName string `json:"ebc_name"`
}

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listEtcdBackupConfigReq)

		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		ebcList, err := listEtcdBackupConfig(ctx, userInfoGetter, c, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var ebcAPIList []*apiv2.EtcdBackupConfig
		for _, ebc := range ebcList.Items {
			ebcAPIList = append(ebcAPIList, convertInternalToAPIEtcdBackupConfig(&ebc))
		}

		return ebcAPIList, nil
	}
}

// listEtcdBackupConfigReq represents a request for listing cluster etcd backup configurations
// swagger:parameters listEtcdBackupConfig
type listEtcdBackupConfigReq struct {
	cluster.GetClusterReq
}

func DecodeListEtcdBackupConfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listEtcdBackupConfigReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)
	return req, nil
}

func DecodeGetEtcdBackupConfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getEtcdBackupConfigReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)

	req.EtcdBackupConfigName = mux.Vars(r)["ebc_name"]
	if req.EtcdBackupConfigName == "" {
		return "", fmt.Errorf("'ebc_name' parameter is required but was not provided")
	}

	return req, nil
}

func convertInternalToAPIEtcdBackupConfig(ebc *kubermaticv1.EtcdBackupConfig) *apiv2.EtcdBackupConfig {
	return &apiv2.EtcdBackupConfig{
		Name: ebc.Name,
		Spec: apiv2.EtcdBackupConfigSpec{
			ClusterID: ebc.Spec.Cluster.Name,
			Schedule:  ebc.Spec.Schedule,
			Keep:      ebc.Spec.Keep,
		},
		Status: ebc.Status,
	}
}

func convertAPIToInternalEtcdBackupConfig(name string, ebcSpec *apiv2.EtcdBackupConfigSpec, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfig, error) {

	clusterObjectRef, err := reference.GetReference(scheme.Scheme, cluster)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("error getting cluster object reference: %v", err))
	}

	return &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, kubermaticv1.SchemeGroupVersion.WithKind("Cluster")),
			},
		},
		Spec: kubermaticv1.EtcdBackupConfigSpec{
			Name:     name,
			Cluster:  *clusterObjectRef,
			Schedule: ebcSpec.Schedule,
			Keep:     ebcSpec.Keep,
		},
	}, nil
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

func getEtcdBackupConfig(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID, etcdBackupConfigName string) (*kubermaticv1.EtcdBackupConfig, error) {
	adminUserInfo, privilegedEtcdBackupConfigProvider, err := getAdminUserInfoPrivilegedEtcdBackupConfigProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedEtcdBackupConfigProvider.GetUnsecured(cluster, etcdBackupConfigName)
	}
	userInfo, etcdBackupConfigProvider, err := getUserInfoEtcdBackupConfigProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return etcdBackupConfigProvider.Get(userInfo, cluster, etcdBackupConfigName)
}

func listEtcdBackupConfig(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticv1.Cluster, projectID string) (*kubermaticv1.EtcdBackupConfigList, error) {
	adminUserInfo, privilegedEtcdBackupConfigProvider, err := getAdminUserInfoPrivilegedEtcdBackupConfigProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedEtcdBackupConfigProvider.ListUnsecured(cluster)
	}
	userInfo, etcdBackupConfigProvider, err := getUserInfoEtcdBackupConfigProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return etcdBackupConfigProvider.List(userInfo, cluster)
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
