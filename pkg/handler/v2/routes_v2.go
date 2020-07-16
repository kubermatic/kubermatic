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

package v2

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/pkg/handler"
	"github.com/kubermatic/kubermatic/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/pkg/handler/v2/cluster"
)

// RegisterV2 declares all router paths for v2
func (r Routing) RegisterV2(mux *mux.Router, metrics common.ServerMetrics) {

	// Defines a set of HTTP endpoints for cluster that belong to a project.
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters").
		Handler(r.createCluster(metrics.InitNodeDeploymentFailures))

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters").
		Handler(r.listClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}").
		Handler(r.getCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}").
		Handler(r.deleteCluster())
}

// swagger:route POST /api/v2/projects/{project_id}/clusters project createClusterV2
//
//     Creates a cluster for the given project.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: Cluster
//       401: empty
//       403: empty
func (r Routing) createCluster(initNodeDeploymentFailures *prometheus.CounterVec) http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.CreateEndpoint(r.sshKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, initNodeDeploymentFailures, r.eventRecorderProvider, r.presetsProvider, r.exposeStrategy, r.userInfoGetter, r.settingsProvider, r.updateManager)),
		cluster.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters project listClustersV2
//
//     Lists clusters for the specified project.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterList
//       401: empty
//       403: empty
func (r Routing) listClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id} project getClusterV2
//
//     Gets the cluster with the given name
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Cluster
//       401: empty
//       403: empty
func (r Routing) getCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id} project deleteClusterV2
//
//     Deletes the specified cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.DeleteEndpoint(r.sshKeyProvider, r.privilegedSSHKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeDeleteReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}
