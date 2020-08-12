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

package externalcluster

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/clientcmd"
)

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		config, err := base64.StdEncoding.DecodeString(req.Body.Kubeconfig)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		cfg, err := clientcmd.Load(config)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if _, err := clusterProvider.GenerateClient(cfg); err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("cannot connect to the kubernetes cluster: %v", err))
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		newCluster := genExternalCluster(req.Body.Name)

		kuberneteshelper.AddFinalizer(newCluster, apiv1.ExternalClusterKubeconfigCleanupFinalizer)

		if err := clusterProvider.CreateOrUpdateKubeconfigSecretForCluster(ctx, newCluster, cfg); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		createdCluster, err := createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertClusterToAPI(createdCluster), nil
	}
}

// createClusterReq defines HTTP request for createExternalCluster
// swagger:parameters createExternalCluster
type createClusterReq struct {
	common.ProjectReq
	// in: body
	Body struct {
		// Name is human readable name for the external cluster
		Name string `json:"name"`
		// Kubeconfig Base64 encoded kubeconfig
		Kubeconfig string `json:"kubeconfig"`
	}
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// Validate validates CreateEndpoint request
func (req createClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	return nil
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, deleteCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, cluster)
	}
}

// deleteClusterReq defines HTTP request for deleteExternalCluster
// swagger:parameters deleteExternalCluster
type deleteClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	return req, nil
}

// Validate validates DeleteEndpoint request
func (req deleteClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func genExternalCluster(name string) *kubermaticapiv1.ExternalCluster {
	return &kubermaticapiv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{Name: rand.String(10)},
		Spec: kubermaticapiv1.ExternalClusterSpec{
			HumanReadableName: name,
		},
	}
}

func createNewCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, cluster *kubermaticapiv1.ExternalCluster, project *kubermaticapiv1.Project) (*kubermaticapiv1.ExternalCluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.NewUnsecured(project, cluster)
	}
	userInfo, err := userInfoGetter(ctx, project.Name)
	if err != nil {
		return nil, err
	}
	return clusterProvider.New(userInfo, project, cluster)
}

func convertClusterToAPI(internalCluster *kubermaticapiv1.ExternalCluster) *apiv1.Cluster {
	cluster := &apiv1.Cluster{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalCluster.Name,
			Name:              internalCluster.Spec.HumanReadableName,
			CreationTimestamp: apiv1.NewTime(internalCluster.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if internalCluster.DeletionTimestamp != nil {
					deletionTimestamp := apiv1.NewTime(internalCluster.DeletionTimestamp.Time)
					return &deletionTimestamp
				}
				return nil
			}(),
		},
		Labels: internalCluster.Labels,
		Type:   apiv1.KubernetesClusterType,
	}

	return cluster
}

func getCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, projectID, clusterName string) (*kubermaticapiv1.ExternalCluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.GetUnsecured(clusterName)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return clusterProvider.Get(userInfo, clusterName)
}

func deleteCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, projectID string, cluster *kubermaticapiv1.ExternalCluster) error {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.DeleteUnsecured(cluster)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return err
	}
	return clusterProvider.Delete(userInfo, cluster)
}
