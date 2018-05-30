package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// RegisterV2 declares all router paths for v2
func (r Routing) RegisterV2(mux *mux.Router) {
	mux.Methods(http.MethodPost).
		Path("/cluster/{cluster}/nodes").
		Handler(r.createNodeHandlerV2())

	mux.Methods(http.MethodGet).
		Path("/cluster/{cluster}/nodes").
		Handler(r.getNodesHandlerV2())

	mux.Methods(http.MethodGet).
		Path("/cluster/{cluster}/nodes/{node}").
		Handler(r.getNodeHandlerV2())

	mux.Methods(http.MethodDelete).
		Path("/cluster/{cluster}/nodes/{node}").
		Handler(r.deleteNodeHandlerV2())
}

// Create node
// swagger:route POST /api/v2/cluster/{cluster}/nodes cluster createClusterNodeV2
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
func (r Routing) createNodeHandlerV2() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(createNodeEndpointV2(r.datacenters, r.sshKeyProvider)),
		decodeCreateNodeReqV2,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Get nodes
// swagger:route GET /api/v2/cluster/{cluster}/nodes cluster getClusterNodesV2
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeListV2
func (r Routing) getNodesHandlerV2() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(getNodesEndpointV2()),
		decodeNodesV2Req,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get node
// swagger:route GET /api/v2/cluster/{cluster}/nodes/{node} cluster getClusterNodeV2
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeV2
func (r Routing) getNodeHandlerV2() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(getNodeEndpointV2()),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Delete node
// swagger:route DELETE /api/v2/cluster/{cluster}/nodes/{node} cluster deleteClusterNodeV2
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteNodeHandlerV2() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.datacenterMiddleware(),
		)(deleteNodeEndpointV2()),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
