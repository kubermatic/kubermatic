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
		Path("/dc/{dc}/cluster/{cluster}/upgrades").
		Handler(r.getPossibleClusterUpgradesV3())

	mux.Methods(http.MethodPut).
		Path("/dc/{dc}/cluster/{cluster}/upgrade").
		Handler(r.performClusterUpgradeV3())
}

// Creates a cluster
// swagger:route POST /api/v3/cluster cluster createCluster
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
		)(newClusterEndpoint(r.sshKeyProvider)),
		decodeNewClusterReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Get the cluster
// swagger:route GET /api/v3/cluster/{cluster} cluster getCluster
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
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// kubeconfigHandler returns the kubeconfig for the cluster.
// swagger:route GET /api/v3/cluster/{cluster}/kubeconfig cluster getClusterKubeconfig
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
// swagger:route GET /api/v3/cluster cluster listClusters
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
// swagger:route DELETE /api/v3/cluster/{cluster} cluster deleteCluster
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
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get nodes
// swagger:route GET /api/v3/cluster/{cluster}/node cluster getClusterNodes
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeListV1
func (r Routing) nodesHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(nodesEndpoint()),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Create nodes
// swagger:route POST /api/v3/cluster/{cluster}/node cluster createClusterNodes
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: empty
func (r Routing) createNodesHandlerV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(createNodesEndpoint(r.cloudProviders, r.sshKeyProvider, r.versions)),
		decodeCreateNodesReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Delete's the node
// swagger:route DELETE /api/v3/cluster/{cluster}/node/{node} cluster deleteClusterNode
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
		)(deleteNodeEndpoint()),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getPossibleClusterUpgradesV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(getClusterUpgrades(r.versions, r.updates)),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) performClusterUpgradeV3() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(performClusterUpgrade(r.versions, r.updates)),
		decodeUpgradeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
