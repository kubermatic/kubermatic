package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"
)

// Binding represents an object which binds endpoints to http handlers.
type Binding struct {
	ctx                 context.Context
	datacenterEndpoint  endpoint.Endpoint
	datacentersEndpoint endpoint.Endpoint
	newClusterEndpoint  endpoint.Endpoint
	clusterEndpoint     endpoint.Endpoint
	clustersEndpoint    endpoint.Endpoint
	nodesEndpoint       endpoint.Endpoint
}

// NewBinding creates a new Binding.
func NewBinding(
	ctx context.Context,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) Binding {
	return Binding{
		ctx:                 ctx,
		datacenterEndpoint:  datacenterEndpoint(kps, cps),
		datacentersEndpoint: datacentersEndpoint(kps, cps),
		newClusterEndpoint:  newClusterEndpoint(kps, cps),
		clusterEndpoint:     clusterEndpoint(kps, cps),
		clustersEndpoint:    clustersEndpoint(kps, cps),
		nodesEndpoint:       nodesEndpoint(kps, cps),
	}
}

// Register registers all known endpoints in the given router.
func (b Binding) Register(mux *mux.Router) {
	mux.
		Methods("GET").
		Path("/").
		HandlerFunc(StatusOK)

	mux.
		Methods("GET").
		Path("/api/v1/dc").
		Handler(b.datacentersHandler())

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}").
		Handler(b.datacenterHandler())

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(b.newClusterHandler())

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster").
		Handler(b.clustersHandler())

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(b.clusterHandler())

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(b.nodesHandler())
}

func (b Binding) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		b.datacentersEndpoint,
		decodeDatacentersReq,
		encodeJSON,
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		b.datacenterEndpoint,
		decodeDatacenterReq,
		encodeJSON,
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) newClusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		b.newClusterEndpoint,
		decodeNewClusterReq,
		encodeJSON,
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) clusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		b.clusterEndpoint,
		decodeClusterReq,
		encodeJSON,
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) clustersHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		b.clustersEndpoint,
		decodeClustersReq,
		encodeJSON,
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) nodesHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		b.nodesEndpoint,
		decodeNodesReq,
		encodeJSON,
		defaultHTTPErrorEncoder(),
	)
}
