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
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	ctx                 context.Context
	datacenters         map[string]provider.DatacenterMeta
	kubernetesProviders map[string]provider.KubernetesProvider
	cloudProviders      map[string]provider.CloudProvider
	logger              log.Logger
	masterCrdClient     crdclient.Interface
	authenticator       Authenticator
	versions            map[string]*api.MasterVersion
	updates             []api.MasterUpdate
}

// NewRouting creates a new Routing.
func NewRouting(
	ctx context.Context,
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
	authenticator Authenticator,
	masterCrdClient crdclient.Interface,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
) Routing {
	return Routing{
		ctx:                 ctx,
		datacenters:         dcs,
		kubernetesProviders: kps,
		cloudProviders:      cps,
		logger:              log.NewLogfmtLogger(os.Stderr),
		masterCrdClient:     masterCrdClient,
		authenticator:       authenticator,
		versions:            versions,
		updates:             updates,
	}
}

func (r Routing) withDefaultAuthenticatedChain(e endpoint.Endpoint, dec httptransport.DecodeRequestFunc) http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Signer(),
		)(e),
		dec,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

// Register declare router paths
func (r Routing) Register(mux *mux.Router) {
	mux.Methods(http.MethodGet).
		Path("/").
		HandlerFunc(StatusOK)

	mux.Methods(http.MethodGet).
		Path("/api/").
		HandlerFunc(APIDescriptionHandler)

	mux.Methods(http.MethodGet).
		Path("/healthz").
		HandlerFunc(StatusOK)

	mux.Methods(http.MethodGet).
		Path("/api/healthz").
		HandlerFunc(StatusOK)

	mux.Methods(http.MethodGet).
		PathPrefix("/swagger-ui/").
		Handler(http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir("./swagger-ui"))))

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/kubeconfig").
		Handler(r.kubeconfigHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc").
		Handler(r.withDefaultAuthenticatedChain(datacentersEndpoint(r.datacenters, r.kubernetesProviders, r.cloudProviders), decodeDatacentersReq))

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}").
		Handler(r.withDefaultAuthenticatedChain(datacenterEndpoint(r.datacenters, r.kubernetesProviders, r.cloudProviders), decodeDcReq))

	mux.Methods(http.MethodPost).
		Path("/api/v1/cluster").
		Handler(r.withDefaultAuthenticatedChain(newClusterEndpointV2(r.kubernetesProviders, r.datacenters, r.masterCrdClient), decodeNewClusterReqV2))

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster").
		Handler(r.withDefaultAuthenticatedChain(clustersEndpoint(r.kubernetesProviders, r.cloudProviders), decodeClustersReq))

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(r.withDefaultAuthenticatedChain(clusterEndpoint(r.kubernetesProviders, r.cloudProviders), decodeClusterReq))

	mux.Methods(http.MethodDelete).
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(r.withDefaultAuthenticatedChain(deleteClusterEndpoint(r.kubernetesProviders, r.cloudProviders, r.masterCrdClient), decodeDeleteClusterReq))

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(r.withDefaultAuthenticatedChain(nodesEndpoint(r.kubernetesProviders, r.cloudProviders), decodeNodesReq))

	mux.Methods(http.MethodPost).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(r.withDefaultAuthenticatedChain(createNodesEndpoint(r.kubernetesProviders, r.cloudProviders, r.masterCrdClient, r.versions), decodeCreateNodesReq))

	mux.Methods(http.MethodDelete).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node/{node}").
		Handler(r.withDefaultAuthenticatedChain(deleteNodeEndpoint(r.kubernetesProviders, r.cloudProviders), decodeNodeReq))

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/upgrades").
		Handler(r.withDefaultAuthenticatedChain(getClusterUpgrades(r.kubernetesProviders, r.versions, r.updates), decodeClusterReq))

	mux.Methods(http.MethodPut).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/upgrade").
		Handler(r.withDefaultAuthenticatedChain(performClusterUpgrade(r.kubernetesProviders, r.versions, r.updates), decodeUpgradeReq))

	mux.Methods(http.MethodPost).
		Path("/api/v1/ext/{dc}/keys").
		Handler(r.withDefaultAuthenticatedChain(datacenterKeyEndpoint(r.datacenters), decodeDcKeyListRequest))

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/k8s/nodes").
		Handler(r.withDefaultAuthenticatedChain(nodesEndpoint(r.kubernetesProviders, r.cloudProviders), decodeNodesReq))

	mux.Methods(http.MethodGet).
		Path("/api/v1/ssh-keys").
		Handler(r.withDefaultAuthenticatedChain(listSSHKeyEndpoint(r.masterCrdClient), decodeListSSHKeyReq))

	mux.Methods(http.MethodPost).
		Path("/api/v1/ssh-keys").
		Handler(r.withDefaultAuthenticatedChain(createSSHKeyEndpoint(r.masterCrdClient), decodeCreateSSHKeyReq))

	mux.Methods(http.MethodDelete).
		Path("/api/v1/ssh-keys/{meta_name}").
		Handler(r.withDefaultAuthenticatedChain(deleteSSHKeyEndpoint(r.masterCrdClient), decodeDeleteSSHKeyReq))
}

// kubeconfigHandler returns the cubeconfig for the cluster.
func (r Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(r.authenticator.Signer())(kubeconfigEndpoint(r.kubernetesProviders, r.cloudProviders)),
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}
