package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// RegisterV3 declares all router paths for v3
func (r Routing) RegisterV3(mux *mux.Router) {
	mux.Methods(http.MethodPost).
		Path("/dc/{dc}/cluster").
		Handler(r.newClusterHandlerV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster").
		Handler(r.clustersHandlerV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster/{cluster}").
		Handler(r.clusterHandlerV3())

	mux.Methods(http.MethodPut).
		Path("/dc/{dc}/cluster/{cluster}").
		Handler(r.updateClusterHandlerV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster/{cluster}/kubeconfig").
		Handler(r.kubeconfigHandlerV3())

	mux.Methods(http.MethodDelete).
		Path("/dc/{dc}/cluster/{cluster}").
		Handler(r.deleteClusterHandlerV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster/{cluster}/node").
		Handler(r.nodesHandlerV3())

	mux.Methods(http.MethodPost).
		Path("/dc/{dc}/cluster/{cluster}/node").
		Handler(r.createNodesHandlerV3())

	mux.Methods(http.MethodDelete).
		Path("/dc/{dc}/cluster/{cluster}/node/{node}").
		Handler(r.deleteNodeHandlerV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster/{cluster}/node/{node}").
		Handler(r.getNodeHandlerV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster/{cluster}/upgrades").
		Handler(r.legacyGetPossibleClusterUpgradesV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster/{cluster}/metrics").
		Handler(r.legacyClusterMetricsHandlerV3())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}/cluster/{cluster}/prometheus/{query_path}").
		Handler(r.prometheusProxyHandlerV3())
}

// Creates a cluster
// swagger:route POST /api/v3/dc/{dc}/cluster cluster createClusterV3
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: ClusterV1
func (r Routing) newClusterHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(newClusterEndpoint(r.sshKeyProvider, r.cloudProviders, r.updateManager)),
		decodeNewClusterReq,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Get the cluster
// swagger:route GET /api/v3/dc/{dc}/cluster/{cluster} cluster getClusterV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterV1
func (r Routing) clusterHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(clusterEndpoint()),
		decodeLegacyClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Update the cluster
// swagger:route PUT /api/v3/dc/{dc}/cluster/{cluster} cluster updateClusterV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterV1
func (r Routing) updateClusterHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(updateClusterEndpoint(r.cloudProviders)),
		decodeUpdateClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// kubeconfigHandler returns the kubeconfig for the cluster.
// swagger:route GET /api/v3/dc/{dc}/cluster/{cluster}/kubeconfig cluster getClusterKubeconfigV3
//
//     Produces:
//     - application/yaml
//
//     Responses:
//       default: errorResponse
//       200: Kubeconfig
func (r Routing) kubeconfigHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(kubeconfigEndpoint()),
		decodeKubeconfigReq,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// List clusters
// swagger:route GET /api/v3/dc/{dc}/cluster cluster listClustersV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterListV1
func (r Routing) clustersHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(clustersEndpoint()),
		decodeClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v3/dc/{dc}/cluster/{cluster} cluster deleteClusterV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteClusterHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(deleteClusterEndpoint()),
		decodeLegacyClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get nodes
// swagger:route GET /api/v3/dc/{dc}/cluster/{cluster}/node cluster nodesHandlerV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeListV2
func (r Routing) nodesHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(getNodesEndpointV3()),
		decodeNodesV3Req,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Create nodes
// swagger:route POST /api/v3/dc/{dc}/cluster/{cluster}/node cluster createNodesHandlerV3
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: NodeV2
func (r Routing) createNodesHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(createNodeEndpointV3(r.datacenters, r.sshKeyProvider)),
		decodeCreateNodeReqV3,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Delete's the node
// swagger:route DELETE /api/v3/dc/{dc}/cluster/{cluster}/node/{node} cluster deleteNodeHandlerV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteNodeHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(deleteNodeEndpointV3()),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get node
// swagger:route GET /api/v3/dc/{dc}/cluster/{cluster}/node/{node} cluster getNodeHandlerV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeV2
func (r Routing) getNodeHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(getNodeEndpointV3()),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get possible cluster upgrades
// swagger:route GET /api/v3/dc/{dc}/cluster/{cluster}/upgrades cluster legacyGetPossibleClusterUpgradesV3
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: MasterVersion
func (r Routing) legacyGetPossibleClusterUpgradesV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(legacyGetClusterUpgrades(r.updateManager)),
		decodeLegacyClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) legacyClusterMetricsHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(legacyGetClusterMetricsEndpoint(r.prometheusClient)),
		decodeLegacyClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) prometheusProxyHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(getPrometheusProxyEndpoint()),
		decodeLegacyPrometheusProxyReq,
		encodeRawResponse,
		r.defaultServerOptions()...,
	)
}
