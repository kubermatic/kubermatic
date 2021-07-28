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

package etcdrestore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"k8s.io/client-go/kubernetes/scheme"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
)

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createEtcdRestoreReq)

		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		if err := req.validate(); err != nil {
			return nil, err
		}

		er, err := convertAPIToInternalEtcdRestore(req.Body.Name, &req.Body.Spec, c)
		if err != nil {
			return nil, err
		}

		er, err = createEtcdRestore(ctx, userInfoGetter, req.ProjectID, er)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToAPIEtcdRestore(er), nil
	}
}

// createEtcdRestoreReq represents a request for creating a cluster etcd backup restore
// swagger:parameters createEtcdRestore
type createEtcdRestoreReq struct {
	cluster.GetClusterReq
	// in: body
	Body erBody
}

type erBody struct {
	// Name of the etcd backup restore
	Name string `json:"name"`
	// EtcdRestoreSpec Spec of the etcd backup restore
	Spec apiv2.EtcdRestoreSpec `json:"spec"`
}

func (r *createEtcdRestoreReq) validate() error {
	if r.Body.Spec.BackupName == "" {
		return errors.NewBadRequest("backup name cannot be empty")
	}
	// NOTE we can check if the backup really exists on S3 or if the backup secret exists (if set), but the restore status will give this info as well
	return nil
}

func DecodeCreateEtcdRestoreReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createEtcdRestoreReq
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

func convertInternalToAPIEtcdRestore(er *kubermaticv1.EtcdRestore) *apiv2.EtcdRestore {
	return &apiv2.EtcdRestore{
		Name: er.Name,
		Spec: apiv2.EtcdRestoreSpec{
			ClusterID:                       er.Spec.Cluster.Name,
			BackupName:                      er.Spec.BackupName,
			BackupDownloadCredentialsSecret: er.Spec.BackupDownloadCredentialsSecret,
		},
		Status: er.Status,
	}
}

func convertAPIToInternalEtcdRestore(name string, erSpec *apiv2.EtcdRestoreSpec, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdRestore, error) {

	clusterObjectRef, err := reference.GetReference(scheme.Scheme, cluster)
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("error getting cluster object reference: %v", err))
	}

	return &kubermaticv1.EtcdRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, kubermaticv1.SchemeGroupVersion.WithKind("Cluster")),
			},
		},
		Spec: kubermaticv1.EtcdRestoreSpec{
			Name:                            name,
			Cluster:                         *clusterObjectRef,
			BackupName:                      erSpec.BackupName,
			BackupDownloadCredentialsSecret: erSpec.BackupDownloadCredentialsSecret,
		},
	}, nil
}

func createEtcdRestore(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string, etcdRestore *kubermaticv1.EtcdRestore) (*kubermaticv1.EtcdRestore, error) {
	adminUserInfo, privilegedEtcdRestoreProvider, err := getAdminUserInfoPrivilegedEtcdRestoreProvider(ctx, userInfoGetter)
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedEtcdRestoreProvider.CreateUnsecured(etcdRestore)
	}
	userInfo, etcdRestoreProvider, err := getUserInfoEtcdRestoreProvider(ctx, userInfoGetter, projectID)
	if err != nil {
		return nil, err
	}
	return etcdRestoreProvider.Create(userInfo, etcdRestore)
}

func getAdminUserInfoPrivilegedEtcdRestoreProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter) (*provider.UserInfo, provider.PrivilegedEtcdRestoreProvider, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	if !userInfo.IsAdmin {
		return userInfo, nil, nil
	}
	privilegedEtcdRestoreProvider := ctx.Value(middleware.PrivilegedEtcdRestoreProviderContextKey).(provider.PrivilegedEtcdRestoreProvider)
	return userInfo, privilegedEtcdRestoreProvider, nil
}

func getUserInfoEtcdRestoreProvider(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string) (*provider.UserInfo, provider.EtcdRestoreProvider, error) {

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}

	etcdRestoreProvider := ctx.Value(middleware.EtcdRestoreProviderContextKey).(provider.EtcdRestoreProvider)
	return userInfo, etcdRestoreProvider, nil
}
