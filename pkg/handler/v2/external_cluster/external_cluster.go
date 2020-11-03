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
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	warningType = "warning"
	normalType  = "normal"
)

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

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

		if err := clusterProvider.CreateOrUpdateKubeconfigSecretForCluster(ctx, newCluster, req.Body.Kubeconfig); err != nil {
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
	Body body
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

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

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

func ListEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(listClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		clusterList, err := clusterProvider.List(project)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		apiClusters := make([]*apiv1.Cluster, 0)

		for _, cluster := range clusterList.Items {
			apiClusters = append(apiClusters, convertClusterToAPI(&cluster))
		}

		return apiClusters, nil
	}
}

// listClusterReq defines HTTP request for listExternalClusters
// swagger:parameters listExternalClusters
type listClusterReq struct {
	common.ProjectReq
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// Validate validates ListEndpoint request
func (req listClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	return nil
}

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(getClusterReq)
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

		version, err := clusterProvider.GetVersion(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiCluster := convertClusterToAPI(cluster)
		apiCluster.Spec = apiv1.ClusterSpec{
			Version: *version,
		}

		return apiCluster, nil
	}
}

// getClusterReq defines HTTP request for getExternalCluster
// swagger:parameters getExternalCluster getExternalClusterMetrics
type getClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getClusterReq

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
func (req getClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func UpdateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(updateClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if req.Body.Kubeconfig != "" {
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
			if err := clusterProvider.CreateOrUpdateKubeconfigSecretForCluster(ctx, cluster, req.Body.Kubeconfig); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}

		if req.Body.Name != "" && req.Body.Name != cluster.Spec.HumanReadableName {
			cluster.Spec.HumanReadableName = req.Body.Name
			cluster, err = updateCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, cluster)
			if err != nil {
				return nil, errors.NewBadRequest(err.Error())
			}
		}

		return convertClusterToAPI(cluster), nil
	}
}

// updateClusterReq defines HTTP request for updateExternalCluster
// swagger:parameters updateExternalCluster
type updateClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: body
	Body body
}

func DecodeUpdateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateClusterReq

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

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// Validate validates CreateEndpoint request
func (req updateClusterReq) Validate() error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	if len(req.ClusterID) == 0 {
		return fmt.Errorf("the cluster ID cannot be empty")
	}
	return nil
}

func GetMetricsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(getClusterReq)
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

		isMetricServer, err := clusterProvider.IsMetricServerAvailable(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if isMetricServer {
			client, err := clusterProvider.GetClient(cluster)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			nodeList := &corev1.NodeList{}
			if err := client.List(ctx, nodeList); err != nil {
				return nil, err
			}
			availableResources := make(map[string]corev1.ResourceList)
			for _, n := range nodeList.Items {
				availableResources[n.Name] = n.Status.Allocatable
			}
			allNodeMetricsList := &v1beta1.NodeMetricsList{}
			if err := client.List(ctx, allNodeMetricsList); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			podMetricsList := &v1beta1.PodMetricsList{}
			if err := client.List(ctx, podMetricsList, &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			return handlercommon.ConvertClusterMetrics(podMetricsList, allNodeMetricsList.Items, availableResources, cluster.Name)
		}

		return &apiv1.ClusterMetrics{
			Name:                cluster.Name,
			ControlPlaneMetrics: apiv1.ControlPlaneMetrics{},
			NodesMetrics:        apiv1.NodesMetric{},
		}, nil
	}
}

func ListEventsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(listEventsReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		client, err := clusterProvider.GetClient(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		eventType := ""
		events := make([]apiv1.Event, 0)

		switch req.Type {
		case warningType:
			eventType = corev1.EventTypeWarning
		case normalType:
			eventType = corev1.EventTypeNormal
		}

		// get nodes events
		nodes := &corev1.NodeList{}
		if err := client.List(ctx, nodes); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		for _, node := range nodes.Items {
			nodeEvents, err := common.GetEvents(ctx, client, &node, "")
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			events = append(events, nodeEvents...)
		}

		// get pods events from kube-system namespace
		pods := &corev1.PodList{}
		if err := client.List(ctx, pods, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		for _, pod := range pods.Items {
			nodeEvents, err := common.GetEvents(ctx, client, &pod, metav1.NamespaceSystem)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			events = append(events, nodeEvents...)
		}

		if len(eventType) > 0 {
			events = common.FilterEventsByType(events, eventType)
		}

		return events, nil
	}
}

// listEventsReq defines HTTP request for listExternalClusterEvents
// swagger:parameters listExternalClusterEvents
type listEventsReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`

	// in: query
	Type string `json:"type,omitempty"`
}

func DecodeListEventsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listEventsReq

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

	req.Type = r.URL.Query().Get("type")
	if len(req.Type) > 0 {
		if req.Type == warningType || req.Type == normalType {
			return req, nil
		}
		return nil, fmt.Errorf("wrong query parameter, unsupported type: %s", req.Type)
	}

	return req, nil
}

// Validate validates ListNodesEventsEndpoint request
func (req listEventsReq) Validate() error {
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

func updateCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, projectID string, cluster *kubermaticapiv1.ExternalCluster) (*kubermaticapiv1.ExternalCluster, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		return privilegedClusterProvider.UpdateUnsecured(cluster)
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return clusterProvider.Update(userInfo, cluster)
}

func AreExternalClustersEnabled(provider provider.SettingsProvider) bool {
	settings, err := provider.GetGlobalSettings()
	if err != nil {
		return false
	}

	return settings.Spec.EnableExternalClusterImport
}

type body struct {
	// Name is human readable name for the external cluster
	Name string `json:"name"`
	// Kubeconfig Base64 encoded kubeconfig
	Kubeconfig string `json:"kubeconfig"`
}
