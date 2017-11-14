package handler

import (
	"context"
	"net/http"
	"os"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	ctx             context.Context
	datacenters     map[string]provider.DatacenterMeta
	cloudProviders  map[string]provider.CloudProvider
	clusterProvider provider.ClusterProvider
	logger          log.Logger
	dataProvider    provider.DataProvider
	authenticator   auth.Authenticator
	versions        map[string]*api.MasterVersion
	updates         []api.MasterUpdate
}

// NewRouting creates a new Routing.
func NewRouting(
	ctx context.Context,
	dcs map[string]provider.DatacenterMeta,
	kp provider.ClusterProvider,
	cps map[string]provider.CloudProvider,
	authenticator auth.Authenticator,
	dataProvider provider.DataProvider,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
) Routing {
	return Routing{
		ctx:             ctx,
		datacenters:     dcs,
		clusterProvider: kp,
		cloudProviders:  cps,
		logger:          log.NewLogfmtLogger(os.Stderr),
		dataProvider:    dataProvider,
		authenticator:   authenticator,
		versions:        versions,
		updates:         updates,
	}
}

// Register declare router paths
func (r Routing) Register(mux *mux.Router) {
	mux.
		Methods(http.MethodGet).
		Path("/").
		HandlerFunc(StatusOK)

	mux.
		Methods(http.MethodGet).
		Path("/healthz").
		HandlerFunc(StatusOK)

	mux.
		Methods(http.MethodGet).
		Path("/api/healthz").
		HandlerFunc(StatusOK)

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc").
		Handler(r.datacentersHandler())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}").
		Handler(r.datacenterHandler())

	mux.
		Methods(http.MethodPost).
		Path("/api/v1/cluster").
		Handler(r.newClusterHandlerV2())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/cluster").
		Handler(r.clustersHandler())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}").
		Handler(r.clusterHandler())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}/kubeconfig").
		Handler(r.kubeconfigHandler())

	mux.
		Methods(http.MethodDelete).
		Path("/api/v1/cluster/{cluster}").
		Handler(r.deleteClusterHandler())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}/node").
		Handler(r.nodesHandler())

	mux.
		Methods(http.MethodPost).
		Path("/api/v1/cluster/{cluster}/node").
		Handler(r.createNodesHandler())

	mux.
		Methods(http.MethodDelete).
		Path("/api/v1/cluster/{cluster}/node/{node}").
		Handler(r.deleteNodeHandler())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}/upgrades").
		Handler(r.getPossibleClusterUpgrades())

	mux.
		Methods(http.MethodPut).
		Path("/api/v1/cluster/{cluster}/upgrade").
		Handler(r.performClusterUpgrage())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/k8s/nodes").
		Handler(r.nodesHandler())

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/ssh-keys").
		Handler(r.listSSHKeys())

	mux.
		Methods(http.MethodPost).
		Path("/api/v1/ssh-keys").
		Handler(r.createSSHKey())

	mux.
		Methods(http.MethodDelete).
		Path("/api/v1/ssh-keys/{meta_name}").
		Handler(r.deleteSSHKey())
}

func (r Routing) auth(e endpoint.Endpoint) endpoint.Endpoint {
	return endpoint.Chain(r.authenticator.Verifier())(e)
}

// swagger:route GET /api/v1/ssh-keys ssh keys list getlistSSHKeys
//
// Lists SSH keys from the user
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKeys
func (r Routing) listSSHKeys() http.Handler {
	return httptransport.NewServer(
		r.auth(listSSHKeyEndpoint(r.dataProvider)),
		decodeListSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) createSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(createSSHKeyEndpoint(r.dataProvider)),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) deleteSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(deleteSSHKeyEndpoint(r.dataProvider)),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(datacentersEndpoint(r.datacenters)),
		decodeDatacentersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(datacenterEndpoint(r.datacenters)),
		decodeDcReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// newClusterHandlerV2 creates a new cluster with the new single request strategy (#165).
func (r Routing) newClusterHandlerV2() http.Handler {
	return httptransport.NewServer(
		r.auth(newClusterEndpointV2(r.clusterProvider, r.dataProvider)),
		decodeNewClusterReqV2,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// clusterHandler returns a cluster object.
func (r Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(clusterEndpoint(r.clusterProvider)),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// kubeconfigHandler returns the cubeconfig for the cluster.
func (r Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(kubeconfigEndpoint(r.clusterProvider)),
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// clustersHandler lists all clusters from a user.
func (r Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(clustersEndpoint(r.clusterProvider)),
		decodeClustersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// deleteClusterHandler deletes a cluster.
func (r Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(deleteClusterEndpoint(r.clusterProvider, r.cloudProviders)),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// nodesHandler returns all nodes from a user.
func (r Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(nodesEndpoint(r.clusterProvider)),
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// createNodesHandler let's you create nodes.
func (r Routing) createNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(createNodesEndpoint(r.clusterProvider, r.cloudProviders, r.dataProvider, r.versions)),
		decodeCreateNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// deleteNodeHandler let's you delete nodes.
func (r Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(deleteNodeEndpoint(r.clusterProvider)),
		decodeNodeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// getPossibleClusterUpgrades returns a list of possible cluster upgrades
func (r Routing) getPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		r.auth(getClusterUpgrades(r.clusterProvider, r.versions, r.updates)),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// performClusterUpgrage starts a cluster upgrade to a specific version
func (r Routing) performClusterUpgrage() http.Handler {
	return httptransport.NewServer(
		r.auth(performClusterUpgrade(r.clusterProvider, r.versions, r.updates)),
		decodeUpgradeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}
