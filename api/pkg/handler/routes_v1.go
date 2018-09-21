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

	mux.Methods(http.MethodPost).
		Path("/ssh-keys").
		Handler(r.createSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/ssh-keys/{meta_name}").
		Handler(r.deleteSSHKey())

	mux.Methods(http.MethodGet).
		Path("/providers/digitalocean/sizes").
		Handler(r.listDigitaloceanSizes())
	mux.Methods(http.MethodGet).
		Path("/digitalocean/sizes").
		Handler(r.redirectTo("/api/v1/providers/digitalocean/sizes"))

	mux.Methods(http.MethodGet).
		Path("/providers/azure/sizes").
		Handler(r.listAzureSizes())
	mux.Methods(http.MethodGet).
		Path("/azure/sizes").
		Handler(r.redirectTo("/api/v1/providers/azure/sizes"))

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/sizes").
		Handler(r.listOpenstackSizes())
	mux.Methods(http.MethodGet).
		Path("/openstack/sizes").
		Handler(r.redirectTo("/api/v1/providers/openstack/sizes"))

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/tenants").
		Handler(r.listOpenstackTenants())
	mux.Methods(http.MethodGet).
		Path("/openstack/tenants").
		Handler(r.redirectTo("/api/v1/providers/openstack/tenants"))

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/networks").
		Handler(r.listOpenstackNetworks())
	mux.Methods(http.MethodGet).
		Path("/openstack/networks").
		Handler(r.redirectTo("/api/v1/providers/openstack/networks"))

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/securitygroups").
		Handler(r.listOpenstackSecurityGroups())
	mux.Methods(http.MethodGet).
		Path("/openstack/securitygroups").
		Handler(r.redirectTo("/api/v1/providers/openstack/securitygroups"))

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/subnets").
		Handler(r.listOpenstackSubnets())
	mux.Methods(http.MethodGet).
		Path("/openstack/subnets").
		Handler(r.redirectTo("/api/v1/providers/openstack/subnets"))

	mux.Methods(http.MethodGet).
		Path("/versions").
		Handler(r.getMasterVersions())

	//
	// VSphere related endpoints
	mux.Methods(http.MethodGet).
		Path("/providers/vsphere/networks").
		Handler(r.listVSphereNetworks())
	mux.Methods(http.MethodGet).
		Path("/vsphere/networks").
		Handler(r.redirectTo("/api/v1/providers/vsphere/networks"))

	//
	// Defines a set of HTTP endpoints for project resource
	mux.Methods(http.MethodGet).
		Path("/projects").
		Handler(r.listProjects())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}").
		Handler(r.getProject())

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
	// Defines a set of HTTP endpoints for SSH Keys that belong to a project
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/sshkeys").
		Handler(r.newCreateSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/sshkeys/{key_id}").
		Handler(r.newDeleteSSHKey())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/sshkeys").
		Handler(r.newListSSHKeys())

	//
	// Defines a set of HTTP endpoints for cluster that belong to a project.
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters").
		Handler(r.newCreateCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters").
		Handler(r.newListClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}").
		Handler(r.newGetCluster())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}").
		Handler(r.newUpdateCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig").
		Handler(r.newGetClusterKubeconfig())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}").
		Handler(r.newDeleteCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health").
		Handler(r.newGetClusterHealth())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/upgrades").
		Handler(r.getClusterUpgrades())

	//
	// Defines set of HTTP endpoints for SSH Keys that belong to a cluster

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id}").
		Handler(r.newAssignSSHKeyToCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys").
		Handler(r.newListSSHKeysAssignedToCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id}").
		Handler(r.newDetachSSHKeyFromCluster())

	//
	// Defines a set of HTTP endpoints for nodes that belong to a cluster
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.newGetNodeForCluster())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes").
		Handler(r.newCreateNodeForCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes").
		Handler(r.newListNodesForCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.newDeleteNodeForCluster())

	//
	// Defines set of HTTP endpoints for the admin token that belongs to a cluster
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/token").
		Handler(r.getClusterAdminToken())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/token").
		Handler(r.revokeClusterAdminToken())

	//
	// Defines set of HTTP endpoints for Users of the given project
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/users").
		Handler(r.addUserToProject())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/users").
		Handler(r.getUsersForProject())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/users/{user_id}").
		Handler(r.editUserInProject())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/users/{user_id}").
		Handler(r.deleteUserFromProject())

	//
	// Defines an endpoint to retrieve information about the current token owner
	mux.Methods(http.MethodGet).
		Path("/me").
		Handler(r.getCurrentUser())
}

func (r Routing) redirectTo(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, path, http.StatusMovedPermanently)
	})
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
//     The returned collection is sorted by creation timestamp.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []NewSSHKey
//       401: empty
//       403: empty
func (r Routing) newListSSHKeys() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.userInfoMiddleware(),
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
			r.userInfoMiddleware(),
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

// swagger:route DELETE /api/v1/projects/{project_id}/sshkeys/{key_id} project newDeleteSSHKey
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
			r.userInfoMiddleware(),
		)(newDeleteSSHKeyEndpoint(r.newSSHKeyProvider, r.projectProvider)),
		newDecodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/providers/digitalocean/sizes digitalocean listDigitaloceanSizes
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

// swagger:route GET /api/v1/providers/azure/sizes azure listAzureSizes
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

// swagger:route GET /api/v1/providers/openstack/sizes openstack listOpenstackSizes
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
		decodeOpenstackReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/providers/vsphere/networks vsphere listVSphereNetworks
//
// Lists networks from vsphere datacenter
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []VSphereNetwork
func (r Routing) listVSphereNetworks() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(vsphereNetworksEndpoint(r.cloudProviders)),
		decodeVSphereNetworksReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/providers/openstack/tenants openstack listOpenstackTenants
//
// Lists tenants from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackTenant
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

// swagger:route GET /api/v1/providers/openstack/networks openstack listOpenstackNetworks
//
// Lists networks from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackNetwork
func (r Routing) listOpenstackNetworks() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(openstackNetworkEndpoint(r.cloudProviders)),
		decodeOpenstackReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/providers/openstack/subnets openstack listOpenstackSubnets
//
// Lists subnets from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackSubnet
func (r Routing) listOpenstackSubnets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(openstackSubnetsEndpoint(r.cloudProviders)),
		decodeOpenstackSubnetReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/providers/openstack/securitygroups openstack listOpenstackSecurityGroups
//
// Lists security groups from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackSecurityGroup
func (r Routing) listOpenstackSecurityGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(openstackSecurityGroupEndpoint(r.cloudProviders)),
		decodeOpenstackReq,
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
		decodeLegacyDcReq,
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

// swagger:route GET /api/v1/projects project listProjects
//
//     Lists projects that an authenticated user is a member of.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Project
//       401: empty
//       409: empty
func (r Routing) listProjects() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(listProjectsEndpoint(r.projectProvider, r.userProjectMapper)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id} project getProject
//
//     Gets the project with the given ID
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Project
//       401: empty
//       409: empty
func (r Routing) getProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.userInfoMiddleware(),
		)(getProjectEndpoint(r.projectProvider)),
		decodeGetProject,
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
			r.userInfoMiddleware(),
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
			r.userInfoMiddleware(),
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
			r.userInfoMiddleware(),
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
			r.userInfoMiddleware(),
		)(newListClusters(r.projectProvider)),
		newDecodeListClustersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id} project newGetCluster
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
			r.userInfoMiddleware(),
		)(newGetCluster(r.projectProvider)),
		newDecodeGetClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id} project newUpdateCluster
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
			r.userInfoMiddleware(),
		)(newUpdateCluster(r.cloudProviders, r.projectProvider)),
		newDecodeUpdateClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// newGetClusterKubeconfig returns the kubeconfig for the cluster.
// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig project newGetClusterKubeconfig
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
			r.userInfoMiddleware(),
		)(newGetClusterKubeconfig(r.projectProvider)),
		newDecodeGetClusterKubeconfig,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id} project newDeleteCluster
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
			r.userInfoMiddleware(),
		)(newDeleteCluster(r.newSSHKeyProvider, r.projectProvider)),
		newDecodeGetClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health project newGetClusterHealth
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
			r.userInfoMiddleware(),
		)(getClusterHealth(r.projectProvider)),
		newDecodeGetClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id} project newAssignSSHKeyToCluster
//
//     Assigns an existing ssh key to the given cluster
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
//       401: empty
//       403: empty
func (r Routing) newAssignSSHKeyToCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(assignSSHKeyToCluster(r.newSSHKeyProvider, r.projectProvider)),
		decodeAssignSSHKeyToClusterReq,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys project newListSSHKeysAssignedToCluster
//
//     Lists ssh keys that are assigned to the cluster
//     The returned collection is sorted by creation timestamp.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []NewSSHKey
//       401: empty
//       403: empty
func (r Routing) newListSSHKeysAssignedToCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(listSSHKeysAssingedToCluster(r.newSSHKeyProvider, r.projectProvider)),
		decodeListSSHKeysAssignedToCluster,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id} project newDetachSSHKeyFromCluster
//
//     Unassignes an ssh key from the given cluster
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
func (r Routing) newDetachSSHKeyFromCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(detachSSHKeyFromCluster(r.newSSHKeyProvider, r.projectProvider)),
		decodeDetachSSHKeysFromCluster,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/token project getClusterAdminToken
//
//     Returns the current admin token for the given cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterAdminToken
//       401: empty
//       403: empty
func (r Routing) getClusterAdminToken() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(getClusterAdminToken(r.projectProvider)),
		decodeClusterAdminTokenReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/token project revokeClusterAdminToken
//
//     Revokes the current admin token and returns a newly generated one.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterAdminToken
//       401: empty
//       403: empty
func (r Routing) revokeClusterAdminToken() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(revokeClusterAdminToken(r.projectProvider)),
		decodeClusterAdminTokenReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id} project newGetNodeForCluster
//
//     Gets a node that is assigned to the given cluster.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Node
//       401: empty
//       403: empty
func (r Routing) newGetNodeForCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(newGetNodeForCluster(r.projectProvider)),
		decodeGetNodeForCluster,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes project newCreateNodeForCluster
//
//     Creates a node that will belong to the given cluster
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: Node
//       401: empty
//       403: empty
func (r Routing) newCreateNodeForCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(newCreateNodeForCluster(r.newSSHKeyProvider, r.projectProvider, r.datacenters)),
		decodeCreateNodeForCluster,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes project newListNodesForCluster
//
//
//     Lists nodes that belong to the given cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Node
//       401: empty
//       403: empty
func (r Routing) newListNodesForCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(newListNodesForCluster(r.projectProvider)),
		decodeListNodesForCluster,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id} project newDeleteNodeForCluster
//
//    Deletes the given node that belongs to the cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) newDeleteNodeForCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(newDeleteNodeForCluster(r.projectProvider)),
		decodeDeleteNodeForCluster,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/upgrades project getClusterUpgrades
//
//    Gets possible cluster upgrades
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []MasterVersion
//       401: empty
//       403: empty
func (r Routing) getClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.newDatacenterMiddleware(),
			r.userInfoMiddleware(),
		)(getClusterUpgrades(r.updateManager, r.projectProvider)),
		decodeClusterReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/users users addUserToProject
//
//     Adds the given user to the given project
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: User
//       401: empty
//       403: empty
func (r Routing) addUserToProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.userInfoMiddleware(),
		)(addUserToProject(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		decodeAddUserToProject,
		setStatusCreatedHeader(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/users users getUsersForProject
//
//     Get list of users for the given project
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []User
//       401: empty
//       403: empty
func (r Routing) getUsersForProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.userInfoMiddleware(),
		)(listUsersFromProject(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		decodeGetProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id}/users/{user_id} users editUserInProject
//
//     Changes membership of the given user for the given project
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: User
//       401: empty
//       403: empty
func (r Routing) editUserInProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.userInfoMiddleware(),
		)(editMemberOfProject(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		decodeEditUserToProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/users/{user_id} users deleteUserFromProject
//
//     Removes the given member from the project
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: User
//       401: empty
//       403: empty
func (r Routing) deleteUserFromProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
			r.userInfoMiddleware(),
		)(deleteMemberFromProject(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		decodeDeleteUserFromProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/me users getCurrentUser
//
// Returns information about the current user.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: User
//       401: empty
func (r Routing) getCurrentUser() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getCurrentUserEndpoint(r.userProvider, r.userProjectMapper)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
