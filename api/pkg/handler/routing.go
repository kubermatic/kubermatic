package handler

import (
	"context"
	"net/http"
	"os"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

const (
	rawToken                  = iota
	apiUserContextKey         = iota
	userCRContextKey          = iota
	datacenterContextKey      = iota
	clusterProviderContextKey = iota
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	ctx              context.Context
	datacenters      map[string]provider.DatacenterMeta
	cloudProviders   map[string]provider.CloudProvider
	sshKeyProvider   provider.SSHKeyProvider
	userProvider     provider.UserProvider
	logger           log.Logger
	authenticator    Authenticator
	versions         map[string]*apiv1.MasterVersion
	updates          []apiv1.MasterUpdate
	clusterProviders map[string]provider.ClusterProvider
}

// NewRouting creates a new Routing.
func NewRouting(
	ctx context.Context,
	datacenters map[string]provider.DatacenterMeta,
	clusterProviders map[string]provider.ClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	sshKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	authenticator Authenticator,
	versions map[string]*apiv1.MasterVersion,
	updates []apiv1.MasterUpdate,
) Routing {
	return Routing{
		ctx:              ctx,
		datacenters:      datacenters,
		clusterProviders: clusterProviders,
		sshKeyProvider:   sshKeyProvider,
		userProvider:     userProvider,
		cloudProviders:   cloudProviders,
		logger:           log.NewLogfmtLogger(os.Stderr),
		authenticator:    authenticator,
		versions:         versions,
		updates:          updates,
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	return []httptransport.ServerOption{
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	}
}

// Register declare router paths
func (r Routing) Register(mux *mux.Router) {
	// swagger:route GET /api/healthz healthz
	//
	// Health endpoint
	//
	//     Responses:
	//       200: empty
	mux.Methods(http.MethodGet).
		Path("/api/healthz").
		HandlerFunc(StatusOK)

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc").
		Handler(r.datacentersHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}").
		Handler(r.datacenterHandler())

	mux.Methods(http.MethodPost).
		Path("/api/v1/dc/{dc}/cluster").
		Handler(r.newClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster").
		Handler(r.clustersHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}").
		Handler(r.clusterHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}/kubeconfig").
		Handler(r.kubeconfigHandler())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/cluster/{cluster}").
		Handler(r.deleteClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}/node").
		Handler(r.nodesHandler())

	mux.Methods(http.MethodPost).
		Path("/api/v1/cluster/{cluster}/node").
		Handler(r.createNodesHandler())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/cluster/{cluster}/node/{node}").
		Handler(r.deleteNodeHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/cluster/{cluster}/upgrades").
		Handler(r.getPossibleClusterUpgrades())

	mux.Methods(http.MethodPut).
		Path("/api/v1/cluster/{cluster}/upgrade").
		Handler(r.performClusterUpgrade())

	mux.Methods(http.MethodGet).
		Path("/api/v1/dc/{dc}/cluster/{cluster}/k8s/nodes").
		Handler(r.nodesHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/ssh-keys").
		Handler(r.listSSHKeys())

	mux.Methods(http.MethodGet).
		Path("/api/v1/user").
		Handler(r.getUser())

	mux.Methods(http.MethodPost).
		Path("/api/v1/ssh-keys").
		Handler(r.createSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/ssh-keys/{meta_name}").
		Handler(r.deleteSSHKey())

	mux.Methods(http.MethodGet).
		Path("/api/v1/digitalocean/sizes").
		Handler(r.listDigitaloceanSizes())

	// New project endpoints
	mux.Methods(http.MethodPost).
		Path("/api/v1/projects/{project_id}/cluster").
		Handler(r.newProjectClusterHandlerV2())

	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/cluster").
		Handler(r.getProjectClustersHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}").
		Handler(r.getProjectClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}/kubeconfig").
		Handler(r.getProjectClusterKubeconfigHandler())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}").
		Handler(r.deleteProjectClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}/node").
		Handler(r.getProjectClusterNodesHandler())

	mux.Methods(http.MethodPost).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}/node").
		Handler(r.createProjectClusterNodesHandler())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}/node/{node}").
		Handler(r.deleteProjectClusterNodeHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}/upgrades").
		Handler(r.getProjectClusterPossibleClusterUpgrades())

	mux.Methods(http.MethodPut).
		Path("/api/v1/projects/{project_id}/cluster/{cluster}/upgrade").
		Handler(r.performProjectClusterUpgrade())

	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/dc/{dc}/cluster/{cluster}/k8s/nodes").
		Handler(r.getProjectClusterK8sNodesHandler())

	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/ssh-keys").
		Handler(r.listProjectSSHKeys())

	mux.Methods(http.MethodPost).
		Path("/api/v1/projects/{project_id}/ssh-keys").
		Handler(r.createProjectSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/projects/{project_id}/ssh-keys/{meta_name}").
		Handler(r.deleteProjectSSHKey())

	// Member and organization endpoints
	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/me").
		Handler(r.getProjectMe())

	// Project management
	mux.Methods(http.MethodGet).
		Path("/api/v1/projects").
		Handler(r.getProjects())

	mux.Methods(http.MethodPost).
		Path("/api/v1/projects").
		Handler(r.createProject())

	mux.Methods(http.MethodPut).
		Path("/api/v1/projects/{project_id}").
		Handler(r.updateProject())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/projects/{project_id}").
		Handler(r.deleteProject())

	// Members in project
	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/members").
		Handler(r.getProjectMembers())

	mux.Methods(http.MethodPut).
		Path("/api/v1/projects/{project_id}/members").
		Handler(r.updateProjectMember())

	mux.Methods(http.MethodPost).
		Path("/api/v1/projects/{project_id}/member").
		Handler(r.addProjectMember())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/projects/{project_id}/member/{member_id}").
		Handler(r.deleteProjectMember())

	mux.Methods(http.MethodPost).
		Path("/api/v2/cluster/{cluster}/nodes").
		Handler(r.createNodeHandlerV2())

	mux.Methods(http.MethodGet).
		Path("/api/v2/cluster/{cluster}/nodes").
		Handler(r.getNodesHandlerV2())

	mux.Methods(http.MethodGet).
		Path("/api/v2/cluster/{cluster}/nodes/{node}").
		Handler(r.getNodeHandlerV2())

	mux.Methods(http.MethodDelete).
		Path("/api/v2/cluster/{cluster}/nodes/{node}").
		Handler(r.deleteNodeHandlerV2())
}

type endpointChainer func(e endpoint.Endpoint) endpoint.Endpoint

func (r Routing) auth(e endpoint.Endpoint) endpoint.Endpoint {
	return endpoint.Chain(r.authenticator.Verifier())(e)
}

func (r Routing) userStorer(e endpoint.Endpoint) endpoint.Endpoint {
	return endpoint.Chain(r.userSaverMiddleware())(e)
}

func (r Routing) datacenterLoader(e endpoint.Endpoint) endpoint.Endpoint {
	return endpoint.Chain(r.datacenterMiddleware())(e)
}

func newNotImplementedEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		return nil, errors.NewNotImplemented()
	}
}

// NotImplemented return a "Not Implemented" error.
func (r Routing) NotImplemented() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(newNotImplementedEndpoint())),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/ssh-keys ssh-keys listSSHKeys
//
// Lists SSH keys from the user
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: SSHKey
func (r Routing) listSSHKeys() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(listSSHKeyEndpoint(r.sshKeyProvider))),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/ssh-keys ssh-keys createSSHKey
//
// Creates a SSH keys for the user
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: SSHKey
func (r Routing) createSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createSSHKeyEndpoint(r.sshKeyProvider))),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/ssh-keys/{meta_name} ssh-keys deleteSSHKey
//
// Deletes a SSH keys for the user
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteSSHKeyEndpoint(r.sshKeyProvider))),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/digitalocean/sizes digitalocean listDigitaloceanSizes
//
// Lists sizes from digitalocean
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: DigitaloceanSizeList
func (r Routing) listDigitaloceanSizes() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(digitaloceanSizeEndpoint())),
		decodeDoSizesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/dc datacenter listDatacenters
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: DatacenterList
func (r Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(datacentersEndpoint(r.datacenters))),
		decodeDatacentersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get the datacenter
// swagger:route GET /api/v1/dc/{dc} datacenter getDatacenter
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Datacenter
func (r Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(datacenterEndpoint(r.datacenters))),
		decodeDcReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Creates a cluster
// swagger:route POST /api/v1/dc/{dc}/cluster cluster createCluster
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: ClusterV1
func (r Routing) newClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(newClusterEndpoint(r.sshKeyProvider)))),
		decodeNewClusterReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Get the cluster
// swagger:route GET /api/v1/dc/{dc}/cluster/{cluster} cluster getCluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterV1
func (r Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(clusterEndpoint()))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// kubeconfigHandler returns the kubeconfig for the cluster.
// swagger:route GET /api/v1/dc/{dc}/cluster/{cluster}/kubeconfig cluster getClusterKubeconfig
//
//     Produces:
//     - application/yaml
//
//     Responses:
//       default: errorResponse
//       200: Kubeconfig
func (r Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(kubeconfigEndpoint()))),
		decodeKubeconfigReq,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// List clusters
// swagger:route GET /api/v1/dc/{dc}/cluster cluster listClusters
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterListV1
func (r Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(clustersEndpoint()))),
		decodeClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v1/dc/{dc}/cluster/{cluster} cluster deleteCluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(deleteClusterEndpoint()))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get nodes
// swagger:route GET /api/v1/dc/{dc}/cluster/{cluster}/node cluster getClusterNodes
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeListV1
func (r Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(nodesEndpoint()))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Create nodes
// swagger:route POST /api/v1/dc/{dc}/cluster/{cluster}/node cluster createClusterNodes
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: empty
func (r Routing) createNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(createNodesEndpoint(r.cloudProviders, r.sshKeyProvider, r.versions)))),
		decodeCreateNodesReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Delete's the node
// swagger:route DELETE /api/v1/dc/{dc}/cluster/{cluster}/node/{node} cluster deleteClusterNode
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(deleteNodeEndpoint()))),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(getClusterUpgrades(r.versions, r.updates)))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) performClusterUpgrade() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(performClusterUpgrade(r.versions, r.updates)))),
		decodeUpgradeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getUser() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getUserHandler())),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectMe() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectMeEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjects() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectsEndpoint())),
		// We don't have to write a decoder only for a request without incoming information
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createProjectEndpoint())),
		decodeCreateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) updateProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(updateProjectEndpoint())),
		decodeUpdateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteProjectEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectMembers() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectMembersEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) addProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(addProjectMemberEndpoint())),
		decodeAddProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteProjectMemberEndpoint())),
		decodeDeleteProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) updateProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(updateProjectMemberEndpoint())),
		decodeUpdateProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) newProjectClusterHandlerV2() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(newClusterEndpoint(r.sshKeyProvider)))),
		decodeNewClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClustersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(clustersEndpoint()))),
		decodeClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(clusterEndpoint()))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterKubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(kubeconfigEndpoint()))),
		decodeKubeconfigReq,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(deleteClusterEndpoint()))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(nodesEndpoint()))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProjectClusterNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(createNodesEndpoint(r.cloudProviders, r.sshKeyProvider, r.versions)))),
		decodeCreateNodesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectClusterNodeHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(deleteNodeEndpoint()))),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(getClusterUpgrades(r.versions, r.updates)))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) performProjectClusterUpgrade() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(performClusterUpgrade(r.versions, r.updates)))),
		decodeUpgradeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterK8sNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(r.datacenterLoader(nodesEndpoint()))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) listProjectSSHKeys() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(listSSHKeyEndpoint(r.sshKeyProvider))),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createSSHKeyEndpoint(r.sshKeyProvider))),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteSSHKeyEndpoint(r.sshKeyProvider))),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
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
		r.auth(r.userStorer(r.datacenterLoader(createNodeEndpointV2(r.datacenters, r.sshKeyProvider, r.versions)))),
		decodeCreateNodeReqV2,
		createStatusResource(encodeJSON),
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
		r.auth(r.userStorer(r.datacenterLoader(getNodesEndpointV2()))),
		decodeClusterReq,
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
		r.auth(r.userStorer(r.datacenterLoader(getNodeEndpointV2()))),
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
		r.auth(r.userStorer(r.datacenterLoader(deleteNodeEndpointV2()))),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
