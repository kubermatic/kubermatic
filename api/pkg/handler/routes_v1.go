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
		Path("/azure/sizes").
		Handler(r.listAzureSizes())

	mux.Methods(http.MethodGet).
		Path("/openstack/sizes").
		Handler(r.listOpenstackSizes())

	mux.Methods(http.MethodGet).
		Path("/openstack/tenants").
		Handler(r.listOpenstackTenants())

	mux.Methods(http.MethodGet).
		Path("/versions").
		Handler(r.getMasterVersions())

	//
	// Project management
	mux.Methods(http.MethodGet).
		Path("/projects").
		Handler(r.getProjects())

	mux.Methods(http.MethodPost).
		Path("/projects").
		Handler(r.createProject())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}").
		Handler(r.updateProject())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}").
		Handler(r.deleteProject())

	//
	// SSH keys that belong to a project
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/sshkeys").
		Handler(r.newCreateSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/sshkeys/{key_name}").
		Handler(r.newDeleteSSHKey())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/sshkeys").
		Handler(r.newListSSHKeys())

	//
	// Clusters that belong to a project
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters").
		Handler(r.newCreateCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters").
		Handler(r.newListClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}").
		Handler(r.newGetCluster())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}").
		Handler(r.newUpdateCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/kubeconfig").
		Handler(r.newGetClusterKubeconfig())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}").
		Handler(r.newDeleteCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/health").
		Handler(r.newGetClusterHealth())

	//
	// Defines set of endpoints that manipulate SSH keys of a cluster
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/sshkeys").
		Handler(r.assignSSHKeyToCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/sshkeys").
		Handler(r.listSSHKeysAssignedToCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/sshkeys").
		Handler(r.detachSSHKeysFromCluster())
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

// swagger:route GET /api/v1/projects/{project_id}/sshkeys project newListSSHKeys
//
//     Lists SSH Keys that belong to the given project.
//     The returned collection of keys is sorted by creation timestamp.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NewSSHKeyList
//       401: empty
//       403: empty
func (r Routing) newListSSHKeys() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(newListSSHKeyEndpoint(r.newSSHKeyProvider, r.projectProvider)),
		newDecodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/sshkeys sshkeys createSSHKey
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
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/sshkeys project newCreateSSHKey
//
//    Adds the given SSH key to the specified project.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NewSSHKey
//       401: empty
//       403: empty
func (r Routing) newCreateSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(newCreateSSHKeyEndpoint(r.newSSHKeyProvider, r.projectProvider)),
		newDecodeCreateSSHKeyReq,
		setStatusCreatedHeader(encodeJSON),
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

// swagger:route DELETE /api/v1/projects/{project_id}/sshkeys/{key_name} project newDeleteSSHKey
//
//     Removes the given SSH Key from the system.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) newDeleteSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(newDeleteSSHKeyEndpoint(r.newSSHKeyProvider, r.projectProvider)),
		newDecodeDeleteSSHKeyReq,
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

// swagger:route GET /api/v1/azure/sizes azure listAzureSizes
//
// Lists available VM sizes in an Azure region
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureSizeList
func (r Routing) listAzureSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(azureSizeEndpoint()),
		decodeAzureSizesReq,
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
		)(openstackSizeEndpoint(r.cloudProviders)),
		decodeOpenstackSizeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/openstack/tenants
//
// Lists tenants from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: OpenstackTenantList
func (r Routing) listOpenstackTenants() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(openstackTenantEndpoint(r.cloudProviders)),
		decodeOpenstackTenantReq,
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

// swagger:route GET /api/v1/versions versions getMasterVersions
//
// Lists all versions which don't result in automatic updates
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []MasterVersion
func (r Routing) getMasterVersions() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getMasterVersions(r.updateManager)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects project getProjects
//
//     Lists projects that an authenticated user is a member of.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       501: empty
func (r Routing) getProjects() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getProjectsEndpoint()),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects project createProject
//
//     Creates a brand new project.
//
//     Note that this endpoint can be consumed by every authenticated user.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: Project
//       401: empty
//       409: empty
func (r Routing) createProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(createProjectEndpoint(r.projectProvider)),
		decodeCreateProject,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id} project updateProject
//
//    Updates the given project
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       501: empty
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

// swagger:route DELETE /api/v1/projects/{project_id} project deleteProject
//
//    Deletes the project with the given ID.
//
//     Produces:
//     - application/json
//
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteProjectEndpoint(r.projectProvider)),
		decodeDeleteProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/dc/{dc}/clusters project newCreateCluster
//
//     Creates a cluster for the given project.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: Cluster
//       401: empty
//       403: empty
func (r Routing) newCreateCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(newCreateClusterEndpoint(r.newSSHKeyProvider, r.cloudProviders, r.projectProvider)),
		newDecodeCreateClusterReq,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters project newListClusters
//
//     Lists clusters for the specified project.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterList
//       401: empty
//       403: empty
func (r Routing) newListClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(newListClusters(r.projectProvider)),
		newDecodeListClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_name} project newGetCluster
//
//     Gets the cluster with the given name
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Cluster
//       401: empty
//       403: empty
func (r Routing) newGetCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(newGetCluster(r.projectProvider)),
		newDecodeGetClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_name} project newUpdateCluster
//
//     Updates the given cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Cluster
//       401: empty
//       403: empty
func (r Routing) newUpdateCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(newUpdateCluster(r.cloudProviders, r.projectProvider)),
		newDecodeUpdateClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// newGetClusterKubeconfig returns the kubeconfig for the cluster.
// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/kubeconfig project newGetClusterKubeconfig
//
//     Gets the kubeconfig for the specified cluster.
//
//     Produces:
//     - application/yaml
//
//     Responses:
//       default: errorResponse
//       200: Kubeconfig
//       401: empty
//       403: empty
func (r Routing) newGetClusterKubeconfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(newGetClusterKubeconfig(r.projectProvider)),
		newDecodeGetClusterKubeconfig,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v1/project/{project_id}/dc/{dc}/clusters/{cluster_name} project newDeleteCluster
//
//     Deletes the specified cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) newDeleteCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(newDeleteCluster(r.newSSHKeyProvider, r.projectProvider)),
		newDecodeGetClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/project/{project_id}/dc/{dc}/clusters/{cluster_name}/health project newGetClusterHealth
//
//     Returns the cluster's component health status
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterHealth
//       401: empty
//       403: empty
func (r Routing) newGetClusterHealth() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(getClusterHealth(r.projectProvider)),
		newDecodeGetClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Assign an existing ssh key to a cluster
// swagger:route POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/sshkeys
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) assignSSHKeyToCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(assignSSHKeyToCluster(r.newSSHKeyProvider, r.projectProvider)),
		decodeAssignSSHKeysToClusterReq,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/sshkeys
//
//     List ssh keys that are assigned to the cluster
//     The returned collection is returned by creation timestamp.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: v2.NewSSHKey
//       401: empty
//       403: empty
func (r Routing) listSSHKeysAssignedToCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(listSSHKeysAssingedToCluster(r.newSSHKeyProvider, r.projectProvider)),
		decodeListSSHKeysAssignedToCluster,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Unassing a set of ssh keys from the given cluster
// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_name}/sshkeys
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: Unauthorized
//       403: Forbidden
func (r Routing) detachSSHKeysFromCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
		)(detachSSHKeysFromCluster(r.newSSHKeyProvider, r.projectProvider)),
		decodeDetachSSHKeysFromCluster,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
