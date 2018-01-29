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
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	ctx            context.Context
	datacenters    map[string]provider.DatacenterMeta
	cloudProviders map[string]provider.CloudProvider
	provider       provider.DataProvider
	logger         log.Logger
	authenticator  auth.Authenticator
	versions       map[string]*apiv1.MasterVersion
	updates        []apiv1.MasterUpdate
}

// NewRouting creates a new Routing.
func NewRouting(
	ctx context.Context,
	dcs map[string]provider.DatacenterMeta,
	kp provider.DataProvider,
	cps map[string]provider.CloudProvider,
	authenticator auth.Authenticator,
	versions map[string]*apiv1.MasterVersion,
	updates []apiv1.MasterUpdate,
) Routing {
	return Routing{
		ctx:            ctx,
		datacenters:    dcs,
		provider:       kp,
		cloudProviders: cps,
		logger:         log.NewLogfmtLogger(os.Stderr),
		authenticator:  authenticator,
		versions:       versions,
		updates:        updates,
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
	mux.Methods(http.MethodGet).
		Path("/").
		HandlerFunc(StatusOK)

	mux.Methods(http.MethodGet).
		Path("/healthz").
		HandlerFunc(StatusOK)

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
		Path("/api/v1/cluster").
		Handler(r.newClusterHandlerV2())

	mux.Methods(http.MethodGet).
		Path("/api/v1/cluster").
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
		Handler(r.performClusterUpgrage())

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
}

func (r Routing) auth(e endpoint.Endpoint) endpoint.Endpoint {
	return endpoint.Chain(r.authenticator.Verifier())(e)
}

func (r Routing) userStorer(e endpoint.Endpoint) endpoint.Endpoint {
	return endpoint.Chain(r.userSaverMiddleware())(e)
}

// swagger:route GET /api/v1/ssh-keys ssh keys list listSSHKeys
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
//       200: UserSSHKey
func (r Routing) listSSHKeys() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(listSSHKeyEndpoint(r.provider))),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
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

// swagger:route POST /api/v1/ssh-keys ssh keys create createSSHKey
//
// Creates a SSH keys for the user
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) createSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createSSHKeyEndpoint(r.provider))),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/ssh-keys/{meta_name} ssh keys delete deleteSSHKey
//
// Deletes a SSH keys for the user
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) deleteSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteSSHKeyEndpoint(r.provider))),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/dc datacenter list datacentersHandler
//
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(datacentersEndpoint(r.datacenters))),
		decodeDatacentersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/dc/{dc} datacenter list datacenterHandler
//
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(datacenterEndpoint(r.datacenters))),
		decodeDcReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// newClusterHandlerV2 creates a new cluster with the new single request strategy (#165).
// swagger:route POST /api/v1/cluster cluster list newClusterHandlerV2
//
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) newClusterHandlerV2() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(newClusterEndpointV2(r.provider, r.provider))),
		decodeNewClusterReqV2,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// clusterHandler returns a cluster object.
// swagger:route POST /api/v1/cluster cluster list clusterHandler
//
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(clusterEndpoint(r.provider))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// kubeconfigHandler returns the cubeconfig for the cluster.
// swagger:route GET /api/v1/cluster/{cluster}/kubeconfig kubeconfig get kubeconfigHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: KubeConfig
func (r Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(kubeconfigEndpoint(r.provider))),
		decodeKubeconfigReq,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// clustersHandler lists all clusters from a user.
// swagger:route GET /api/v1/cluster cluster get clustersHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: KubeCluster
func (r Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(clustersEndpoint(r.provider))),
		decodeClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// deleteClusterHandler deletes a cluster.
// swagger:route DELETE /api/v1/cluster/{cluster} cluster delete deleteClusterHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteClusterEndpoint(r.provider, r.cloudProviders))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// nodesHandler returns all nodes from a user.
// swagger:route GET /api/v1/cluster/{cluster}/node nodes get nodesHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: NodeList
func (r Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(nodesEndpoint(r.provider))),
		decodeNodesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// createNodesHandler let's you create nodes.
// swagger:route POST /api/v1/cluster/{cluster}/node ssh keys list createNodesHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) createNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createNodesEndpoint(r.provider, r.cloudProviders, r.provider, r.versions))),
		decodeCreateNodesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// deleteNodeHandler let's you delete nodes.
// swagger:route DELETE /api/v1/cluster/{cluster}/node/{node} nodes delete deleteNodeHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteNodeEndpoint(r.provider))),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getPossibleClusterUpgrades returns a list of possible cluster upgrades
// swagger:route GET /api/v1/cluster/{cluster}/upgrades cluster upgrade versions getPossibleClusterUpgrades
//
//     Produces:
//     - application/json
//
//     Consumes:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: Versions
func (r Routing) getPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getClusterUpgrades(r.provider, r.versions, r.updates))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// performClusterUpgrage starts a cluster upgrade to a specific version
// swagger:route PUT /api/v1/cluster/{cluster}/upgrade cluster upgrade performClusterUpgrage
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) performClusterUpgrage() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(performClusterUpgrade(r.provider, r.versions, r.updates))),
		decodeUpgradeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getUser starts a cluster upgrade to a specific version
// swagger:route GET /api/v1/user user get getUser
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: User
func (r Routing) getUser() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getUserHandler())),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectMe returns the member in the context of a project.
// swagger:route GET /api/v1/projects/{project_id}/me user get project getProjectMe
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: Member
func (r Routing) getProjectMe() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectMeEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjects lists all projects visible for a user.
// swagger:route GET /api/v1/projects get project getProjects
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: ProjectList
func (r Routing) getProjects() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectsEndpoint())),
		// We don't have to write a decoder only for a request without incoming information
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// createProject create a new project this might take a while,
// a owner member will be created for the acting user.
// swagger:route POST /api/v1/projects project create createProject
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: Project
func (r Routing) createProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createProjectEndpoint())),
		decodeCreateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// updateProject update project preferences
// swagger:route PUT /api/v1/projects/{project_id} project update updateProject
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: Project
func (r Routing) updateProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(updateProjectEndpoint())),
		decodeUpdateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// deleteProject close a project
// swagger:route DELETE /api/v1/projects/{project_id} project delete deleteProject
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200:
func (r Routing) deleteProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteProjectEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectMembers list all members of a project
// swagger:route GET /api/v1/projects/{project_id}/members get project member getProjectMembers
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: MemberList
func (r Routing) getProjectMembers() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectMembersEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// addProjectMember invite a user with matching mail to the project
// swagger:route POST /api/v1/projects/{project_id}/members get project member addProjectMember
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: Member
func (r Routing) addProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(addProjectMemberEndpoint())),
		decodeAddProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// deleteProjectMember remove a member from a project
// swagger:route DELETE /api/v1/projects/{project_id}/member/{member_id} get project member deleteProjectMember
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200:
func (r Routing) deleteProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteProjectMemberEndpoint())),
		decodeDeleteProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// updateProjectMember update a member in a project, this should be used to change groups
// swagger:route PUT /api/v1/projects/{project_id}/member/{member_id} get project member updateProjectMember
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: Member
func (r Routing) updateProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(updateProjectMemberEndpoint())),
		decodeUpdateProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// newProjectClusterHandlerV2 creates a new cluster with the new single request strategy (#165).
// swagger:route POST /api/v1/projects/{project_id}/cluster cluster list newProjectClusterHandlerV2
//
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) newProjectClusterHandlerV2() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(newClusterEndpointV2(r.provider, r.provider))),
		decodeNewClusterReqV2,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectClustersHandler lists all clusters from a user.
// swagger:route GET /api/v1/projects/{project_id}/cluster cluster get getProjectClustersHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: KubeCluster
func (r Routing) getProjectClustersHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(clustersEndpoint(r.provider))),
		decodeClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectClusterHandler returns a cluster object.
// swagger:route POST /api/v1/projects/{project_id}/cluster/{cluster} cluster list getProjectClusterHandler
//
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) getProjectClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(clusterEndpoint(r.provider))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectClusterKubeconfigHandler returns the cubeconfig for the cluster.
// swagger:route GET /api/v1/projects/{project_id}/cluster/{cluster}/kubeconfig kubeconfig get getProjectClusterKubeconfigHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: KubeConfig
func (r Routing) getProjectClusterKubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(kubeconfigEndpoint(r.provider))),
		decodeKubeconfigReq,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// deleteProjectClusterHandler deletes a cluster.
// swagger:route DELETE /api/v1/projects/{project_id}/cluster/{cluster} cluster delete deleteProjectClusterHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) deleteProjectClusterHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteClusterEndpoint(r.provider, r.cloudProviders))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectClusterNodesHandler returns all nodes from a user.
// swagger:route GET /api/v1/projects/{project_id}/cluster/{cluster}/node nodes get getProjectClusterNodesHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: NodeList
func (r Routing) getProjectClusterNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(nodesEndpoint(r.provider))),
		decodeNodesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// createProjectClusterNodesHandler let's you create nodes.
// swagger:route POST /api/v1/projects/{project_id}/cluster/{cluster}/node ssh keys list createProjectClusterNodesHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) createProjectClusterNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createNodesEndpoint(r.provider, r.cloudProviders, r.provider, r.versions))),
		decodeCreateNodesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// deleteProjectClusterNodeHandler let's you delete nodes.
// swagger:route DELETE /api/v1/projects/{project_id}/cluster/{cluster}/node/{node} nodes delete deleteProjectClusterNodeHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) deleteProjectClusterNodeHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteNodeEndpoint(r.provider))),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectClusterPossibleClusterUpgrades returns a list of possible cluster upgrades
// swagger:route GET /api/v1/projects/{project_id}/cluster/{cluster}/upgrades cluster upgrade versions getProjectClusterPossibleClusterUpgrades
//
//     Produces:
//     - application/json
//
//     Consumes:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: Versions
func (r Routing) getProjectClusterPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getClusterUpgrades(r.provider, r.versions, r.updates))),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// performProjectClusterUpgrade starts a cluster upgrade to a specific version
// swagger:route PUT /api/v1/projects/{project_id}/cluster/{cluster}/upgrade cluster upgrade performProjectClusterUpgrade
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) performProjectClusterUpgrade() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(performClusterUpgrade(r.provider, r.versions, r.updates))),
		decodeUpgradeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// getProjectClusterK8sNodesHandler returns all nodes from a user.
// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/cluster/{cluster}/k8s/nodes nodes get getProjectClusterK8sNodesHandler
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: NodeList
func (r Routing) getProjectClusterK8sNodesHandler() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(nodesEndpoint(r.provider))),
		decodeNodesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/ssh-keys ssh keys list listProjectSSHKeys
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
//       200: UserSSHKey
func (r Routing) listProjectSSHKeys() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(listSSHKeyEndpoint(r.provider))),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/ssh-keys ssh keys create createProjectSSHKey
//
// Creates a SSH keys for the user
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) createProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createSSHKeyEndpoint(r.provider))),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

//mux.Methods(http.MethodDelete).
//Path("/api/v1/projects/{project_id}/ssh-keys/{meta_name}").
//Handler(r.deleteProjectSSHKey())

// swagger:route DELETE /api/v1/projects/{project_id}/ssh-keys/{meta_name} ssh keys delete deleteProjectSSHKey
//
// Deletes a SSH keys for the user
//
//
//     Produces:
//     - application/json
//
//     Schemes: http, https
//
//     Responses:
//       default: APIError
//       200: UserSSHKey
func (r Routing) deleteProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteSSHKeyEndpoint(r.provider))),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
