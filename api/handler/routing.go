package handler

import (
	"context"
	"net/http"
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api"
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/provider"
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

// Register declare router paths
func (r Routing) Register(mux *mux.Router) {
	mux.
		Methods(http.MethodGet).
		Path("/").
		HandlerFunc(StatusOK)
	mux.
		Methods(http.MethodGet).
		Path("/api/").
		HandlerFunc(APIDescriptionHandler)
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
		PathPrefix("/swagger-ui/").
		Handler(http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir("./swagger-ui"))))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc").
		Handler(r.authenticator.IsAuthenticated(r.datacentersHandler()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}").
		Handler(r.authenticator.IsAuthenticated(r.datacenterHandler()))

	mux.
		Methods(http.MethodPost).
		Path("/api/v1/cluster").
		Handler(r.authenticator.IsAuthenticated(r.newClusterHandlerV2()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster").
		Handler(r.authenticator.IsAuthenticated(r.clustersHandler()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(r.authenticator.IsAuthenticated(r.clusterHandler()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/kubeconfig").
		Handler(r.authenticator.IsAuthenticated(r.kubeconfigHandler()))

	mux.
		Methods(http.MethodDelete).
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(r.authenticator.IsAuthenticated(r.deleteClusterHandler()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(r.authenticator.IsAuthenticated(r.nodesHandler()))

	mux.
		Methods(http.MethodPost).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(r.authenticator.IsAuthenticated(r.createNodesHandler()))

	mux.
		Methods(http.MethodDelete).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node/{node}").
		Handler(r.authenticator.IsAuthenticated(r.deleteNodeHandler()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/upgrades").
		Handler(r.authenticator.IsAuthenticated(r.getPossibleClusterUpgrades()))

	mux.
		Methods(http.MethodPut).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/upgrade").
		Handler(r.authenticator.IsAuthenticated(r.performClusterUpgrade()))

	mux.
		Methods(http.MethodPost).
		Path("/api/v1/ext/{dc}/keys").
		Handler(r.authenticator.IsAuthenticated(r.getAWSKeyHandler()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/k8s/nodes").
		Handler(r.authenticator.IsAuthenticated(r.nodesHandler()))

	mux.
		Methods(http.MethodGet).
		Path("/api/v1/ssh-keys").
		Handler(r.authenticator.IsAuthenticated(r.listSSHKeys()))

	mux.
		Methods(http.MethodPost).
		Path("/api/v1/ssh-keys").
		Handler(r.authenticator.IsAuthenticated(r.createSSHKey()))

	mux.
		Methods(http.MethodDelete).
		Path("/api/v1/ssh-keys/{meta_name}").
		Handler(r.authenticator.IsAuthenticated(r.deleteSSHKey()))
}

// @Title listSSHKeys
// @Description listSSHKeys return list of ssh keys.
// @Accept  json
// @Produce  json
// @Success 200 {object} string
// @Failure 400 {object} APIError "Bad parameters, add user credentials"
// @Router /api/v1/ssh-keys [get]
func (r Routing) listSSHKeys() http.Handler {
	return httptransport.NewServer(
		listSSHKeyEndpoint(r.masterCrdClient),
		decodeListSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// @Title createSSHKey
// @Description createSSHKey add ssh key.
// @Accept  json
// @Produce  json
// @Success 200 {object} string
// @Failure 400 {object} APIError "Bad parameters, add user credentials"
// @Router /api/v1/ssh-keys [post]
func (r Routing) createSSHKey() http.Handler {
	return httptransport.NewServer(
		createSSHKeyEndpoint(r.masterCrdClient),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		httptransport.ServerErrorLogger(r.logger),
	)
}

// @Title deleteSSHKey
// @Description deleteSSHKey delete ssh key.
// @Accept  json
// @Produce  json
// @Success 200 {object} string
// @Failure 400 {object} APIError "Bad parameters"
// @Router /api/v1/ssh-keys/{meta_name} [delete]
func (r Routing) deleteSSHKey() http.Handler {
	return httptransport.NewServer(
		deleteSSHKeyEndpoint(r.masterCrdClient),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

func (r Routing) getAWSKeyHandler() http.Handler {
	return httptransport.NewServer(
		datacenterKeyEndpoint(r.datacenters),
		decodeDcKeyListRequest,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// datacentersHandler serves a list of datacenters.
// @Title DataCenterHandler
// @Description datacentersHandler serves a list of datacenters.
// @Accept  json
// @Produce  json
// @Param   some_id     path    int     true        "Some ID"
// @Success 200 {object} string
// @Failure 400 {object} APIError "We need ID!!"
// @Failure 404 {object} APIError "Can not find ID"
// @Router /api/v1/dc [get]
func (r Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		datacentersEndpoint(r.datacenters, r.kubernetesProviders, r.cloudProviders),
		decodeDatacentersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// datacenterHandler server information for a datacenter.
// Admin only!
// @Title datacenterHandler
// @Description datacenterHandler server information for a datacenter.
// @Accept  json
// @Produce  json
// @Param   some_id     path    int     true        "Some ID"
// @Success 200 {object} string
// @Failure 400 {object} APIError "We need datacenter"
// @Failure 404 {object} APIError "Can not find datacenter"
// @Router /api/v1/dc/{dc} [get]
func (r Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		datacenterEndpoint(r.datacenters, r.kubernetesProviders, r.cloudProviders),
		decodeDcReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// newClusterHandlerV2 creates a new cluster with the new single request strategy (#165).
func (r Routing) newClusterHandlerV2() http.Handler {
	return httptransport.NewServer(
		newClusterEndpointV2(r.kubernetesProviders, r.datacenters, r.masterCrdClient),
		decodeNewClusterReqV2,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// clusterHandler returns a cluster object.
func (r Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		clusterEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// kubeconfigHandler returns the cubeconfig for the cluster.
func (r Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		kubeconfigEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeKubeconfigReq,
		encodeKubeconfig,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// clustersHandler lists all clusters from a user.
func (r Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		clustersEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeClustersReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// deleteClusterHandler let's you create nodes.
func (r Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		deleteClusterEndpoint(r.kubernetesProviders, r.cloudProviders, r.masterCrdClient),
		decodeDeleteClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// nodesHandler returns all nodes from a user.
// @Title createNodesHandler
// @Description createNodesHandler create nodes.
// @Accept  json
// @Produce  json
// @Param   dc     path    int     true        "Some ID"
// @Param   cluster     path    string     true        "Some ID"
// @Success 200 {object} string
// @Failure 400 {object} APIError "unknown kubernetes datacenter"
// @Router /api/v1/dc/{dc}/cluster/{cluster}/node [get]
// createNodesHandler let's you create nodes.
// nodesHandler returns all nodes from a user.
func (r Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		nodesEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// createNodesHandler create nodes.
// @Title createNodesHandler
// @Description createNodesHandler create nodes.
// @Accept  json
// @Produce  json
// @Param   dc     path    int     true        "Some ID"
// @Success 200 {object} string
// @Failure 400 {object} APIError "cannot create nodes without cloud provider"
// @Failure 400 {object} APIError "unknown kubernetes datacenter"
// @Router /api/v1/dc/{dc}/cluster/{cluster}/node [post]
// createNodesHandler let's you create nodes.
func (r Routing) createNodesHandler() http.Handler {
	return httptransport.NewServer(
		createNodesEndpoint(r.kubernetesProviders, r.cloudProviders, r.masterCrdClient, r.versions),
		decodeCreateNodesReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// deleteClusterHandler let's you delete nodes.
// @Title deleteClusterHandler
// @Description deleteClusterHandler let's you delete nodes.
// @Accept json
// @Produce json
// @Param   dc     path    int     true        "Some ID"
// @Param   cluster     path    string     true        "Some ID"
// @Success 200 {object} string
// @Failure 400 {object} APIError "unknown kubernetes datacenter"
// @Router /api/v1/dc/{dc}/cluster/{cluster} [delete]
// deleteNodeHandler let's you delete nodes.
func (r Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		deleteNodeEndpoint(r.kubernetesProviders, r.cloudProviders),
		decodeNodeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// getPossibleClusterUpgrades returns a list of possible cluster upgrades
func (r Routing) getPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		getClusterUpgrades(r.kubernetesProviders, r.versions, r.updates),
		decodeClusterReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}

// performClusterUpgrade starts a cluster upgrade to a specific version
func (r Routing) performClusterUpgrade() http.Handler {
	return httptransport.NewServer(
		performClusterUpgrade(r.kubernetesProviders, r.versions, r.updates),
		decodeUpgradeReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
	)
}
