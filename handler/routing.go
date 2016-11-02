package handler

import (
	"net/http"
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	ctx                 context.Context
	authenticated       func(http.Handler) http.Handler
	getAuthenticated    func(http.Handler) http.Handler
	datacenters         map[string]provider.DatacenterMeta
	kubernetesProviders map[string]provider.KubernetesProvider
	cloudProviders      map[string]provider.CloudProvider
	logger              log.Logger
}

// NewRouting creates a new Routing.
func NewRouting(
	ctx context.Context,
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
	auth bool,
	jwtKey string,
) Routing {
	var authenticated = func(h http.Handler) http.Handler { return h }
	var getAuthenticated = func(h http.Handler) http.Handler { return h }
	if auth {
		authenticated = jwtMiddleware(jwtKey).Handler
		getAuthenticated = jwtGetMiddleware(jwtKey).Handler
	}

	return Routing{
		ctx:                 ctx,
		authenticated:       authenticated,
		getAuthenticated:    getAuthenticated,
		datacenters:         dcs,
		kubernetesProviders: kps,
		cloudProviders:      cps,
		logger:              log.NewLogfmtLogger(os.Stderr),
	}
}

// Register registers all known endpoints in the given router.
func (b Routing) Register(mux *mux.Router) {
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
		Methods("PUT").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/cloud").
		Handler(b.authenticated(b.setCloudHandler()))

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

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(b.authenticated(b.createNodesHandler()))

	mux.
		Methods("DELETE").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node/{node}").
		Handler(b.authenticated(b.deleteNodeHandler()))
}

// datacentersHandler handles
func (b Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		datacentersEndpoint(b.datacenters, b.kubernetesProviders, b.cloudProviders),
		decodeDatacentersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// datacenterHandler handles
func (b Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		datacenterEndpoint(b.datacenters, b.kubernetesProviders, b.cloudProviders),
		decodeDcReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// newClusterHandler
func (b Routing) newClusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		newClusterEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeNewClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// clusterHandler
func (b Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		clusterEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// setCloudHandler
func (b Routing) setCloudHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		setCloudEndpoint(b.datacenters, b.kubernetesProviders, b.cloudProviders),
		decodeSetCloudReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// kubeconfigHandler
func (b Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		kubeconfigEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// clustersHandler lists all clusters from a user.
func (b Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		clustersEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeClustersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// deleteClusterHandler deletes a cluster.
func (b Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		deleteClusterEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeDeleteClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// nodesHandler returns all nodes from a user.
func (b Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		nodesEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// createNodesHandler let's you create nodes.
func (b Routing) createNodesHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		createNodesEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeCreateNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

// deleteNodeHandler let's you delete nodes.
func (b Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		deleteNodeEndpoint(b.kubernetesProviders, b.cloudProviders),
		decodeNodeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}
