/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ServiceAccountComponentKey is the label key that tag k8s service account created though UI.
	ServiceAccountComponentKey = "component"

	// ServiceAccountComponentValue is the value associated to ServiceAccountComponentKey label.
	ServiceAccountComponentValue = "clusterServiceAccount"
)

func ListClusterSAEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listClusterSAReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		serviceAccountList := &corev1.ServiceAccountList{}
		if err := client.List(ctx, serviceAccountList, ctrlruntimeclient.MatchingLabels{ServiceAccountComponentKey: ServiceAccountComponentValue}); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		saList := make([]*apiv2.ClusterServiceAccount, len(serviceAccountList.Items))
		for i := range serviceAccountList.Items {
			saList[i] = convertInternalClusterSAToExternal(&serviceAccountList.Items[i])
		}
		return saList, nil
	}
}

func CreateClusterSAEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(CreateClusterSAReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Body.Name,
				Namespace: req.Body.Namespace,
				Labels:    map[string]string{ServiceAccountComponentKey: ServiceAccountComponentValue},
			},
		}
		if err := client.Create(ctx, serviceAccount); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		secretTokenSa := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    req.Body.Namespace,
				GenerateName: fmt.Sprintf("sa-%s-", serviceAccount.Name),
				Annotations:  map[string]string{corev1.ServiceAccountNameKey: serviceAccount.Name},
			},
			Type: corev1.SecretTypeServiceAccountToken,
		}

		if err := client.Create(ctx, secretTokenSa); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalClusterSAToExternal(serviceAccount), nil
	}
}

func DeleteClusterSAKubeconigEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(ClusterSAReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		serviceAccount := &corev1.ServiceAccount{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.ServiceAccountID}, serviceAccount); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if serviceAccount.GetLabels()[ServiceAccountComponentKey] != ServiceAccountComponentValue {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("can not delete service account which is not labeled %s=%s", ServiceAccountComponentKey, ServiceAccountComponentValue))
		}

		if err := handlercommon.UnbindServiceAccountFromRoles(ctx, client, serviceAccount); err != nil {
			return nil, err
		}

		if err := handlercommon.UnbindServiceAccountFromClusterRoles(ctx, client, serviceAccount); err != nil {
			return nil, err
		}

		if err := client.Delete(ctx, serviceAccount); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func convertInternalClusterSAToExternal(sa *corev1.ServiceAccount) *apiv2.ClusterServiceAccount {
	return &apiv2.ClusterServiceAccount{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                sa.Name,
			Name:              sa.Name,
			CreationTimestamp: apiv1.NewTime(sa.CreationTimestamp.Time),
		},
		Namespace: sa.Namespace,
	}
}

// listClusterSAReq defines HTTP request for ListClusterSAEndpoint
// swagger:parameters listClusterServiceAccount
type listClusterSAReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func (req listClusterSAReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeListClusterServiceAccount(c context.Context, r *http.Request) (interface{}, error) {
	var req listClusterSAReq

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	return req, nil
}

// Validate validates listClusterSAReq request.
func (req listClusterSAReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	return nil
}

// CreateClusterSAReq defines HTTP request for CreateClusterSAEndpoint
// swagger:parameters createClusterServiceAccount
type CreateClusterSAReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`

	// in: body
	Body apiv2.ClusterServiceAccount
}

func (req CreateClusterSAReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeCreateClusterServiceAccount(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateClusterSAReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

// Validate validates CreateClusterSAReq request.
func (req CreateClusterSAReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}

	if req.Body.Name == "" {
		return fmt.Errorf("name must be set")
	}
	if req.Body.Namespace == "" {
		return fmt.Errorf("namespace must be set")
	}
	return nil
}

// ClusterSAReq defines HTTP request for GetClusterSAKubeconigEndpoint and DeleteClusterSAKubeconigEndpoint
// swagger:parameters getClusterServiceAccountKubeconfig deleteClusterServiceAccount
type ClusterSAReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`

	// in: path
	// required: true
	Namespace string `json:"namespace"`

	// in: path
	// required: true
	ServiceAccountID string `json:"service_account_id"`
}

func (req ClusterSAReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeClusterSAReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterSAReq

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	ns := mux.Vars(r)["namespace"]
	if ns == "" {
		return "", fmt.Errorf("'namespace' parameter is required but was not provided")
	}
	req.Namespace = ns

	saID := mux.Vars(r)["service_account_id"]
	if saID == "" {
		return "", fmt.Errorf("'service_account_id' parameter is required but was not provided")
	}
	req.ServiceAccountID = saID

	return req, nil
}

// Validate validates ClusterSAReq request.
func (req ClusterSAReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	return nil
}
