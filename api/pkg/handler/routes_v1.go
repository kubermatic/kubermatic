package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// RegisterV1 declares all router paths for v1
func (r Routing) RegisterV1(mux *mux.Router) {
	// swagger:route GET /api/v1/healthz healthz
	//
	// Health endpoint
	//
	//     Responses:
	//       default: empty
	mux.Methods(http.MethodGet).
		Path("/healthz").
		HandlerFunc(StatusOK)

	mux.Methods(http.MethodGet).
		Path("/dc").
		Handler(r.datacentersHandler())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}").
		Handler(r.datacenterHandler())

	mux.Methods(http.MethodPost).
		Path("/cluster").
		Handler(r.newClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/cluster").
		Handler(r.clustersHandler())

	mux.Methods(http.MethodGet).
		Path("/cluster/{cluster}").
		Handler(r.clusterHandler())

	mux.Methods(http.MethodGet).
		Path("/cluster/{cluster}/kubeconfig").
		Handler(r.kubeconfigHandler())

	mux.Methods(http.MethodDelete).
		Path("/cluster/{cluster}").
		Handler(r.deleteClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/cluster/{cluster}/node").
		Handler(r.nodesHandler())

	mux.Methods(http.MethodPost).
		Path("/cluster/{cluster}/node").
		Handler(r.createNodesHandler())

	mux.Methods(http.MethodDelete).
		Path("/cluster/{cluster}/node/{node}").
		Handler(r.deleteNodeHandler())

	mux.Methods(http.MethodGet).
		Path("/cluster/{cluster}/upgrades").
		Handler(r.getPossibleClusterUpgrades())

	mux.Methods(http.MethodPut).
		Path("/cluster/{cluster}/upgrade").
		Handler(r.performClusterUpgrade())

	mux.Methods(http.MethodGet).
		Path("/ssh-keys").
		Handler(r.listSSHKeys())

	mux.Methods(http.MethodGet).
		Path("/user").
		Handler(r.getUser())

	mux.Methods(http.MethodPost).
		Path("/ssh-keys").
		Handler(r.createSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/ssh-keys/{meta_name}").
		Handler(r.deleteSSHKey())

	mux.Methods(http.MethodGet).
		Path("/digitalocean/sizes").
		Handler(r.listDigitaloceanSizes())

	mux.Methods(http.MethodGet).
		Path("/openstack/sizes").
		Handler(r.listOpenstackSizes())

	// New project endpoints
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/cluster").
		Handler(r.newProjectClusterHandlerV2())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/cluster").
		Handler(r.getProjectClustersHandler())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/cluster/{cluster}").
		Handler(r.getProjectClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/cluster/{cluster}/kubeconfig").
		Handler(r.getProjectClusterKubeconfigHandler())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/cluster/{cluster}").
		Handler(r.deleteProjectClusterHandler())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/cluster/{cluster}/node").
		Handler(r.getProjectClusterNodesHandler())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/cluster/{cluster}/node").
		Handler(r.createProjectClusterNodesHandler())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/cluster/{cluster}/node/{node}").
		Handler(r.deleteProjectClusterNodeHandler())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/cluster/{cluster}/upgrades").
		Handler(r.getProjectClusterPossibleClusterUpgrades())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/cluster/{cluster}/upgrade").
		Handler(r.performProjectClusterUpgrade())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/ssh-keys").
		Handler(r.listProjectSSHKeys())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/ssh-keys").
		Handler(r.createProjectSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/ssh-keys/{meta_name}").
		Handler(r.deleteProjectSSHKey())

	// Member and organization endpoints
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/me").
		Handler(r.getProjectMe())

	// Project management
	mux.Methods(http.MethodGet).
		Path("/projects").
		Handler(r.getProjects())

	mux.Methods(http.MethodPost).
		Path("projects/{project_id}/projects").
		Handler(r.createProject())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}").
		Handler(r.updateProject())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}").
		Handler(r.deleteProject())

	// Members in project
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/members").
		Handler(r.getProjectMembers())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/members").
		Handler(r.updateProjectMember())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/member").
		Handler(r.addProjectMember())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/member/{member_id}").
		Handler(r.deleteProjectMember())
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(listSSHKeyEndpoint(r.sshKeyProvider)),
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(createSSHKeyEndpoint(r.sshKeyProvider)),
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteSSHKeyEndpoint(r.sshKeyProvider)),
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(digitaloceanSizeEndpoint()),
		decodeDoSizesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/openstack/sizes openstack listOpenstackSizes
//
// Lists sizes from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackSize
func (r Routing) listOpenstackSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(openstackSizeEndpoint()),
		decodeOpenstackSizesReq,
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(datacentersEndpoint(r.datacenters)),
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(datacenterEndpoint(r.datacenters)),
		decodeDcReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Creates a cluster
// swagger:route POST /api/v1/cluster cluster createCluster
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(newClusterEndpoint(r.sshKeyProvider, r.cloudProviders)),
		decodeNewClusterReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Get the cluster
// swagger:route GET /api/v1/cluster/{cluster} cluster getCluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterV1
func (r Routing) clusterHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(clusterEndpoint()),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// kubeconfigHandler returns the kubeconfig for the cluster.
// swagger:route GET /api/v1/cluster/{cluster}/kubeconfig cluster getClusterKubeconfig
//
//     Produces:
//     - application/yaml
//
//     Responses:
//       default: errorResponse
//       200: Kubeconfig
func (r Routing) kubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(kubeconfigEndpoint()),
		decodeKubeconfigReq,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// List clusters
// swagger:route GET /api/v1/cluster cluster listClusters
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterListV1
func (r Routing) clustersHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(clustersEndpoint()),
		decodeClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v1/cluster/{cluster} cluster deleteCluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteClusterHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(deleteClusterEndpoint()),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get nodes
// swagger:route GET /api/v1/cluster/{cluster}/node cluster getClusterNodes
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeListV1
func (r Routing) nodesHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(nodesEndpoint()),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Create nodes
// swagger:route POST /api/v1/cluster/{cluster}/node cluster createClusterNodes
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
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(createNodesEndpoint(r.cloudProviders, r.sshKeyProvider, r.versions)),
		decodeCreateNodesReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// Delete's the node
// swagger:route DELETE /api/v1/cluster/{cluster}/node/{node} cluster deleteClusterNode
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteNodeHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(deleteNodeEndpoint()),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(getClusterUpgrades(r.versions, r.updates)),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) performClusterUpgrade() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(performClusterUpgrade(r.versions, r.updates)),
		decodeUpgradeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getUser() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getUserHandler()),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectMe() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getProjectMeEndpoint()),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjects() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getProjectsEndpoint()),
		// We don't have to write a decoder only for a request without incoming information
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(createProjectEndpoint()),
		decodeCreateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) updateProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(updateProjectEndpoint()),
		decodeUpdateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteProjectEndpoint()),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectMembers() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getProjectMembersEndpoint()),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) addProjectMember() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(addProjectMemberEndpoint()),
		decodeAddProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectMember() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteProjectMemberEndpoint()),
		decodeDeleteProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) updateProjectMember() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(updateProjectMemberEndpoint()),
		decodeUpdateProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) newProjectClusterHandlerV2() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(newClusterEndpoint(r.sshKeyProvider, r.cloudProviders)),
		decodeNewClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClustersHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(clustersEndpoint()),
		decodeClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(clusterEndpoint()),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterKubeconfigHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(kubeconfigEndpoint()),
		decodeKubeconfigReq,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectClusterHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(deleteClusterEndpoint()),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterNodesHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(nodesEndpoint()),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProjectClusterNodesHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(createNodesEndpoint(r.cloudProviders, r.sshKeyProvider, r.versions)),
		decodeCreateNodesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectClusterNodeHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(deleteNodeEndpoint()),
		decodeNodeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectClusterPossibleClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(getClusterUpgrades(r.versions, r.updates)),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) performProjectClusterUpgrade() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.optimisticDatacenterMiddleware(),
		)(performClusterUpgrade(r.versions, r.updates)),
		decodeUpgradeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) listProjectSSHKeys() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(listSSHKeyEndpoint(r.sshKeyProvider)),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(createSSHKeyEndpoint(r.sshKeyProvider)),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteSSHKeyEndpoint(r.sshKeyProvider)),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
