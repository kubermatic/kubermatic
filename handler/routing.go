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
	ctx              context.Context
	authenticated    func(http.Handler) http.Handler
	getAuthenticated func(http.Handler) http.Handler
	dcs              map[string]provider.DatacenterMeta
	kps              map[string]provider.KubernetesProvider
	cps              map[string]provider.CloudProvider
	logger           log.Logger
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
		ctx:              ctx,
		authenticated:    authenticated,
		getAuthenticated: getAuthenticated,
		dcs:              dcs,
		kps:              kps,
		cps:              cps,
		logger:           log.NewLogfmtLogger(os.Stderr),
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

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/addon").
		Handler(b.authenticated(b.createAddonHandler()))

}

func (b Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		datacentersEndpoint(b.dcs, b.kps, b.cps),
		decodeDatacentersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		datacenterEndpoint(b.dcs, b.kps, b.cps),
		decodeDcReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) newClusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		newClusterEndpoint(b.kps, b.cps),
		decodeNewClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		clusterEndpoint(b.kps, b.cps),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) setCloudHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		setCloudEndpoint(b.dcs, b.kps, b.cps),
		decodeSetCloudReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		kubeconfigEndpoint(b.kps, b.cps),
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		clustersEndpoint(b.kps, b.cps),
		decodeClustersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		deleteClusterEndpoint(b.kps, b.cps),
		decodeDeleteClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		nodesEndpoint(b.kps, b.cps),
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) createNodesHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		createNodesEndpoint(b.kps, b.cps),
		decodeCreateNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		deleteNodeEndpoint(b.kps, b.cps),
		decodeNodeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (b Routing) createAddonHandler() http.Handler {
	return httptransport.NewServer(
		b.ctx,
		createAddonEndpoint(b.kps, b.cps),
		decodeCreateAddonRequest,
		encodeJSON,
		httptransport.ServerErrorLogger(b.logger),
		defaultHTTPErrorEncoder(),
	)
}
