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

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/endpoint"
	"github.com/prometheus/client_golang/prometheus"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	handlercommon "github.com/kubermatic/kubermatic/pkg/handler/common"
	"github.com/kubermatic/kubermatic/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func CreateEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	initNodeDeploymentFailures *prometheus.CounterVec, eventRecorderProvider provider.EventRecorderProvider, credentialManager provider.PresetProvider,
	exposeStrategy corev1.ServiceType, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider, updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateClusterReq)
		globalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate(globalSettings.Spec.ClusterTypeOptions, updateManager)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		return handlercommon.CreateEndpoint(ctx, req.ProjectID, req.Body, sshKeyProvider, projectProvider, privilegedProjectProvider, seedsGetter, initNodeDeploymentFailures, eventRecorderProvider, credentialManager, exposeStrategy, userInfoGetter)

	}
}

// ListEndpoint list clusters for the given project
func ListEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, clusterProviderGetter provider.ClusterProviderGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetProjectRq)
		allClusters := make([]*apiv1.Cluster, 0)

		seeds, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for _, seed := range seeds {
			// if a Seed is bad, do not forward that error to the user, but only log
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				klog.Errorf("failed to create cluster provider for seed %s: %v", seed.Name, err)
				continue
			}
			apiClusters, err := handlercommon.GetExternalClusters(ctx, userInfoGetter, clusterProvider, projectProvider, privilegedProjectProvider, req.ProjectID)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			allClusters = append(allClusters, apiClusters...)
		}

		return allClusters, nil
	}
}

func GetEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetClusterReq)
		return handlercommon.GetEndpoint(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func DeleteEndpoint(sshKeyProvider provider.SSHKeyProvider, privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteReq)
		return handlercommon.DeleteEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.DeleteVolumes, req.DeleteLoadBalancers, sshKeyProvider, privilegedSSHKeyProvider, projectProvider, privilegedProjectProvider)
	}
}

func PatchEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchReq)
		return handlercommon.PatchEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.Patch, seedsGetter, projectProvider, privilegedProjectProvider)
	}
}

// PatchReq defines HTTP request for patchCluster endpoint
// swagger:parameters patchClusterV2
type PatchReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`

	// in: body
	Patch json.RawMessage
}

func DecodePatchReq(c context.Context, r *http.Request) (interface{}, error) {
	var req PatchReq

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

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req PatchReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// DeleteReq defines HTTP request for deleteCluster endpoint
// swagger:parameters deleteClusterV2
type DeleteReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: header
	// DeleteVolumes if true all cluster PV's and PVC's will be deleted from cluster
	DeleteVolumes bool
	// in: header
	// DeleteLoadBalancers if true all load balancers will be deleted from cluster
	DeleteLoadBalancers bool
}

// GetSeedCluster returns the SeedCluster object
func (req DeleteReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteReq

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

	headerValue := r.Header.Get("DeleteVolumes")
	if len(headerValue) > 0 {
		deleteVolumes, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteVolumes = deleteVolumes
	}

	headerValue = r.Header.Get("DeleteLoadBalancers")
	if len(headerValue) > 0 {
		deleteLB, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteLoadBalancers = deleteLB
	}

	return req, nil
}

// GetClusterReq defines HTTP request for getCluster endpoint.
// swagger:parameters getClusterV2
type GetClusterReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeGetClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GetClusterReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req GetClusterReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// CreateClusterReq defines HTTP request for createCluster
// swagger:parameters createClusterV2
type CreateClusterReq struct {
	common.ProjectReq
	// in: body
	Body apiv1.CreateClusterSpec

	// private field for the seed name. Needed for the cluster provider.
	seedName string
}

// GetSeedCluster returns the SeedCluster object
func (req CreateClusterReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.seedName,
	}
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	if len(req.Body.Cluster.Type) == 0 {
		req.Body.Cluster.Type = apiv1.KubernetesClusterType
	}

	seedName, err := findSeedNameForDatacenter(c, req.Body.Cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, err
	}
	req.seedName = seedName
	return req, nil
}

// Validate validates CreateEndpoint request
func (req CreateClusterReq) Validate(clusterType kubermaticv1.ClusterType, updateManager common.UpdateManager) error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the project ID cannot be empty")
	}
	return handlercommon.ValidateClusterSpec(clusterType, updateManager, req.Body)
}

func findSeedNameForDatacenter(ctx context.Context, datacenter string) (string, error) {
	seedsGetter, ok := ctx.Value(middleware.SeedsGetterContextKey).(provider.SeedsGetter)
	if !ok {
		return "", fmt.Errorf("seeds getter is not set")
	}
	seeds, err := seedsGetter()
	if err != nil {
		return "", fmt.Errorf("failed to list seeds: %v", err)
	}
	for name, seed := range seeds {
		if _, ok := seed.Spec.Datacenters[datacenter]; ok {
			return name, nil
		}
	}
	return "", fmt.Errorf("can not find seed for datacenter %s", datacenter)
}
