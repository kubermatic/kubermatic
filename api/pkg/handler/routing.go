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
	versions       map[string]*api.MasterVersion
	updates        []api.MasterUpdate
}

// NewRouting creates a new Routing.
func NewRouting(
	ctx context.Context,
	dcs map[string]provider.DatacenterMeta,
	kp provider.DataProvider,
	cps map[string]provider.CloudProvider,
	authenticator auth.Authenticator,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
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

	// Member and organization endpoints
	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/me").
		Handler(r.notImplemented())

	// Project management
	mux.Methods(http.MethodGet).
		Path("/api/v1/projects").
		Handler(r.notImplemented())

	mux.Methods(http.MethodPost).
		Path("/api/v1/projects").
		Handler(r.notImplemented())

	mux.Methods(http.MethodPut).
		Path("/api/v1/projects/{project_id}").
		Handler(r.notImplemented())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/projects/{project_id}").
		Handler(r.notImplemented())

	// Members in project
	mux.Methods(http.MethodGet).
		Path("/api/v1/projects/{project_id}/members").
		Handler(r.notImplemented())

	mux.Methods(http.MethodPut).
		Path("/api/v1/projects/{project_id}/members").
		Handler(r.notImplemented())

	mux.Methods(http.MethodPost).
		Path("/api/v1/projects/{project_id}/member").
		Handler(r.notImplemented())

	mux.Methods(http.MethodDelete).
		Path("/api/v1/projects/{project_id}/member/{member_id}").
		Handler(r.notImplemented())
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) notImplemented() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(
			func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return nil, errors.NewBadRequest("Not implemented")
			})),
		decodeListSSHKeyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
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
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) getProjectMe() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectMeEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) getProjects() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectsEndpoint())),
		// We don't have to write a decoder only for a request without incoming information
		decodeEmptyReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) createProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(createProjectEndpoint())),
		decodeCreateProject,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) updateProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(updateProjectEndpoint())),
		decodeUpdateProject,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) deleteProject() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteProjectEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) getProjectMembers() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(getProjectMembersEndpoint())),
		decodeProjectPathReq,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) addProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(addProjectMemberEndpoint())),
		decodeAddProjectMember,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) deleteProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(deleteProjectMemberEndpoint())),
		decodeDeleteProjectMember,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}

func (r Routing) updateProjectMember() http.Handler {
	return httptransport.NewServer(
		r.auth(r.userStorer(updateProjectMemberEndpoint())),
		decodeUpdateProjectMember,
		encodeJSON,
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	)
}
