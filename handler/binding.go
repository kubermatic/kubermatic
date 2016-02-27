package handler

import (
	"net/http"
	"os"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

// Binding represents an object which binds endpoints to http handlers.
type Binding struct {
	ctx                   context.Context
	authenticated         func(http.Handler) http.Handler
	getAuthenticated      func(http.Handler) http.Handler
	datacenterEndpoint    endpoint.Endpoint
	datacentersEndpoint   endpoint.Endpoint
	newClusterEndpoint    endpoint.Endpoint
	deleteClusterEndpoint endpoint.Endpoint
	clusterEndpoint       endpoint.Endpoint
	kubeconfigEndpoint    endpoint.Endpoint
	clustersEndpoint      endpoint.Endpoint
	nodesEndpoint         endpoint.Endpoint
}

// NewBinding creates a new Binding.
func NewBinding(
	ctx context.Context,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
	auth bool,
	jwtKey string,
) Binding {
	var authenticated = func(h http.Handler) http.Handler { return h }
	var getAuthenticated = func(h http.Handler) http.Handler { return h }
	if auth {
		authenticated = jwtMiddleware(jwtKey).Handler
		getAuthenticated = jwtGetMiddleware(jwtKey).Handler
	}

	return Binding{
		ctx:                   ctx,
		authenticated:         authenticated,
		getAuthenticated:      getAuthenticated,
		datacenterEndpoint:    datacenterEndpoint(kps, cps),
		datacentersEndpoint:   datacentersEndpoint(kps, cps),
		newClusterEndpoint:    newClusterEndpoint(kps, cps),
		deleteClusterEndpoint: deleteClusterEndpoint(kps, cps),
		clusterEndpoint:       clusterEndpoint(kps, cps),
		kubeconfigEndpoint:    kubeconfigEndpoint(kps, cps),
		clustersEndpoint:      clustersEndpoint(kps, cps),
		nodesEndpoint:         nodesEndpoint(kps, cps),
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
		Handler(b.authenticated(b.datacentersHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}").
		Handler(b.authenticated(b.datacenterHandler()))

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster").
		Handler(b.authenticated(b.newClusterHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster").
		Handler(b.authenticated(b.clustersHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(b.authenticated(b.clusterHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/kubeconfig").
		Handler(b.getAuthenticated(b.kubeconfigHandler()))

	mux.
		Methods("DELETE").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(b.authenticated(b.deleteClusterHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(b.authenticated(b.nodesHandler()))
}

func (b Binding) datacentersHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.datacentersEndpoint,
		decodeDatacentersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) datacenterHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.datacenterEndpoint,
		decodeDatacenterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) newClusterHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.newClusterEndpoint,
		decodeNewClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) clusterHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.clusterEndpoint,
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) kubeconfigHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.kubeconfigEndpoint,
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) clustersHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.clustersEndpoint,
		decodeClustersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) deleteClusterHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.deleteClusterEndpoint,
		decodeDeleteClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) nodesHandler() http.Handler {
	logger := log.NewLogfmtLogger(os.Stderr)

	return httptransport.NewServer(
		b.ctx,
		b.nodesEndpoint,
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(logger),
		defaultHTTPErrorEncoder(),
	)
}
