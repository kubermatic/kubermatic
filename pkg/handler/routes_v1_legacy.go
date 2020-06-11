package handler

import (
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/node"
)

// RegisterV1Legacy declares legacy HTTP paths that can be deleted in the future
// At the time of this writing, there is no clear deprecation policy
func (r Routing) RegisterV1Legacy(mux *mux.Router) {
	//
	// Defines a set of HTTP endpoints for nodes that belong to a cluster
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.getNodeForClusterLegacy())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes").
		Handler(r.createNodeForClusterLegacy())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes").
		Handler(r.listNodesForClusterLegacy())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.deleteNodeForClusterLegacy())
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id} project getNodeForClusterLegacy
//
//     Deprecated:
//     Gets a node that is assigned to the given cluster.
//
//     This endpoint is deprecated, please create a Node Deployment instead.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Node
//       401: empty
//       403: empty
func (r Routing) getNodeForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(node.GetNodeForClusterLegacyEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		node.DecodeGetNodeForClusterLegacy,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes project createNodeForClusterLegacy
//
//     Deprecated:
//     Creates a node that will belong to the given cluster
//
//     This endpoint is deprecated, please create a Node Deployment instead.
//     Use POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: Node
//       401: empty
//       403: empty
func (r Routing) createNodeForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(node.CreateNodeForClusterLegacyEndpoint()),
		node.DecodeCreateNodeForClusterLegacy,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes project listNodesForClusterLegacy
//
//     Deprecated:
//     Lists nodes that belong to the given cluster
//
//     This endpoint is deprecated, please create a Node Deployment instead.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Node
//       401: empty
//       403: empty
func (r Routing) listNodesForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(node.ListNodesForClusterLegacyEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		node.DecodeListNodesForClusterLegacy,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id} project deleteNodeForClusterLegacy
//
//    Deprecated:
//    Deletes the given node that belongs to the cluster.
//
//     This endpoint is deprecated, please create a Node Deployment instead.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteNodeForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(node.DeleteNodeForClusterLegacyEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		node.DecodeDeleteNodeForClusterLegacy,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
