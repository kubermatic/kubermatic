package handler

import (
	"net/http"
	"os"

	"github.com/docker/docker/daemon/logger"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

// Binding represents an object which binds endpoints to http handlers.
type Binding struct {
	ctx              context.Context
	authenticated    func(http.Handler) http.Handler
	getAuthenticated func(http.Handler) http.Handler
	kps              map[string]provider.KubernetesProvider
	cps              map[string]provider.CloudProvider
	logger           logger.Logger
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
		ctx:              ctx,
		authenticated:    authenticated,
		getAuthenticated: getAuthenticated,
		kps:              kps,
		cps:              cps,
		logger:           log.NewLogfmtLogger(os.Stderr),
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
	return httptransport.NewServer(
		b.ctx,
		datacentersEndpoint(b.kps, b.cps),
		decodeDatacentersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		datacenterEndpoint(b.kps, b.cps),
		decodeDatacenterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) newClusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		newClusterEndpoint(b.kps, b.cps),
		decodeNewClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) clusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		clusterEndpoint(b.kps, b.cps),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		kubeconfigEndpoint(b.kps, b.cps),
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) clustersHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		clustersEndpoint(b.kps, b.cps),
		decodeClustersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		deleteClusterEndpoint(b.kps, b.cps),
		decodeDeleteClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Binding) nodesHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		nodesEndpoint(b.kps, b.cps),
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}
