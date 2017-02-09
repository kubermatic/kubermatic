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
func (r Routing) Register(mux *mux.Router) {
	mux.
		Methods("GET").
		Path("/").
		HandlerFunc(StatusOK)
	mux.
		Methods("GET").
		Path("/healthz").
		HandlerFunc(StatusOK)
	mux.
		Methods("GET").
		Path("/api/healthz").
		HandlerFunc(StatusOK)

	mux.
		Methods("GET").
		Path("/api/v1/dc").
		Handler(r.authenticated(r.datacentersHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}").
		Handler(r.authenticated(r.datacenterHandler()))

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster").
		Handler(r.authenticated(r.newClusterHandler()))

	mux.
		Methods("POST").
		Path("/api/v1/cluster").
		Handler(r.authenticated(r.newClusterHandlerV2()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster").
		Handler(r.authenticated(r.clustersHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(r.authenticated(r.clusterHandler()))

	mux.
		Methods("PUT").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/cloud").
		Handler(r.authenticated(r.setCloudHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/kubeconfig").
		Handler(r.getAuthenticated(r.kubeconfigHandler()))

	mux.
		Methods("DELETE").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(r.authenticated(r.deleteClusterHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(r.authenticated(r.nodesHandler()))

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(r.authenticated(r.createNodesHandler()))

	mux.
		Methods("DELETE").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node/{node}").
		Handler(r.authenticated(r.deleteNodeHandler()))

	mux.
		Methods("POST").
		Path("/api/v1/ext/{dc}/keys").
		Handler(r.authenticated(r.getAWSKeyHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/k8s/nodes").
		Handler(r.authenticated(r.getKubernetesNodesHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/k8s/nodes/{node}").
		Handler(r.authenticated(r.getKubernetesNodeInfoHandler()))

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/addon").
		Handler(r.authenticated(r.createAddonHandler()))

	mux.
		Methods("GET").
		Path("/api/v1/ssh-keys").
		Handler(r.authenticated(r.listSSHKeys()))
	mux.
		Methods("POST").
		Path("/api/v1/ssh-keys").
		Handler(r.authenticated(r.createSSHKey()))
	mux.
		Methods("DELETE").
		Path("/api/v1/ssh-keys/{fingerprint}").
		Handler(r.authenticated(r.deleteSSHKey()))
}

func (r Routing) listSSHKeys() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		listSSHKeyEndpoint(r.kubernetesProviders),
		decodeListSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (r Routing) createSSHKey() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		createSSHKeyEndpoint(r.kubernetesProviders),
		decodeCreateSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (r Routing) deleteSSHKey() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		deleteSSHKeyEndpoint(r.kubernetesProviders),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (r Routing) getAWSKeyHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		datacenterKeyEndpoint(r.datacenters),
		decodeDcKeyListRequest,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (r Routing) getKubernetesNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		kubernetesNodesEndpoint(r.kubernetesProviders),
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (r Routing) getKubernetesNodeInfoHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		kubernetesNodeInfoEndpoint(r.kubernetesProviders),
		decodeNodeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// datacentersHandler serves a list of datacenters.
// Admin only!
func (r Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		datacentersEndpoint(r.datacenters, r.kubernetesProviders, r.cloudProviders),
		decodeDatacentersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// datacenterHandler server information for a datacenter.
// Admin only!
func (r Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		datacenterEndpoint(r.datacenters, r.kubernetesProviders, r.cloudProviders),
		decodeDcReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// newClusterHandler creates a new cluster.
func (r Routing) newClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		newClusterEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeNewClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// newClusterHandlerV2 creates a new cluster with the new single request strategy (#165).
func (r Routing) newClusterHandlerV2() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		newClusterEndpointV2(r.kubernetesProviders, r.datacenters),
		decodeNewClusterReqV2,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// clusterHandler returns a cluster object.
func (r Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		clusterEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// setCloudHandler updates a cluster.
func (r Routing) setCloudHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		setCloudEndpoint(r.datacenters, r.kubernetesProviders, r.cloudProviders),
		decodeSetCloudReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// kubeconfigHandler returns the cubeconfig for the cluster.
func (r Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		kubeconfigEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// clustersHandler lists all clusters from a user.
func (r Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		clustersEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeClustersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// deleteClusterHandler deletes a cluster.
func (r Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		deleteClusterEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeDeleteClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// nodesHandler returns all nodes from a user.
func (r Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		nodesEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// createNodesHandler let's you create nodes.
func (r Routing) createNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		createNodesEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeCreateNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

// deleteNodeHandler let's you delete nodes.
func (r Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		deleteNodeEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeNodeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}

func (r Routing) createAddonHandler() http.Handler {
	return httptransport.NewServer(
		r.ctx,
		createAddonEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeCreateAddonRequest,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		defaultHTTPErrorEncoder(),
	)
}
