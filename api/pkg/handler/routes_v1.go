package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/dc"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/project"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/provider"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/user"
)

// RegisterV1 declares all router paths for v1
func (r Routing) RegisterV1(mux *mux.Router) {
	//
	// no-op endpoint that always returns HTTP 200
	mux.Methods(http.MethodGet).
		Path("/healthz").
		HandlerFunc(StatusOK)

	//
	// Defines endpoints for managing data centers
	mux.Methods(http.MethodGet).
		Path("/dc").
		Handler(r.datacentersHandler())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}").
		Handler(r.datacenterHandler())

	//
	// Defines a set of HTTP endpoint for interacting with
	// various cloud providers
	mux.Methods(http.MethodGet).
		Path("/providers/digitalocean/sizes").
		Handler(r.listDigitaloceanSizes())

	mux.Methods(http.MethodGet).
		Path("/providers/azure/sizes").
		Handler(r.listAzureSizes())

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/sizes").
		Handler(r.listOpenstackSizes())

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/tenants").
		Handler(r.listOpenstackTenants())

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/networks").
		Handler(r.listOpenstackNetworks())

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/securitygroups").
		Handler(r.listOpenstackSecurityGroups())

	mux.Methods(http.MethodGet).
		Path("/providers/openstack/subnets").
		Handler(r.listOpenstackSubnets())

	mux.Methods(http.MethodGet).
		Path("/versions").
		Handler(r.getMasterVersions())

	mux.Methods(http.MethodGet).
		Path("/version").
		Handler(r.getKubermaticVersion())

	mux.Methods(http.MethodGet).
		Path("/providers/vsphere/networks").
		Handler(r.listVSphereNetworks())

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
		Handler(r.createSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/sshkeys/{key_id}").
		Handler(r.deleteSSHKey())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/sshkeys").
		Handler(r.listSSHKeys())

	//
	// Defines a set of HTTP endpoints for cluster that belong to a project.
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters").
		Handler(r.listClustersForProject())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters").
		Handler(r.createCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters").
		Handler(r.listClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}").
		Handler(r.getCluster())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}").
		Handler(r.patchCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig").
		Handler(r.getClusterKubeconfig())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}").
		Handler(r.deleteCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health").
		Handler(r.getClusterHealth())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/upgrades").
		Handler(r.getClusterUpgrades())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/upgrades").
		Handler(r.getClusterNodeUpgrades())

	//
	// Defines set of HTTP endpoints for SSH Keys that belong to a cluster
	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id}").
		Handler(r.assignSSHKeyToCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys").
		Handler(r.listSSHKeysAssignedToCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id}").
		Handler(r.detachSSHKeyFromCluster())

	//
	// Defines a set of HTTP endpoints for nodes that belong to a cluster
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.getNodeForClusterLegacy())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes").
		Handler(r.createNodeForClusterLegacy())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes").
		Handler(r.listNodesForClusterLegacy())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.deleteNodeForClusterLegacy())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/token").
		Handler(r.revokeClusterAdminToken())

	//
	// Defines a set of HTTP endpoint for node deployments that belong to a cluster
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments").
		Handler(r.createNodeDeployment())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments").
		Handler(r.listNodeDeployments())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}").
		Handler(r.getNodeDeployment())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes").
		Handler(r.listNodeDeploymentNodes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events").
		Handler(r.listNodeDeploymentNodesEvents())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}").
		Handler(r.patchNodeDeployment())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}").
		Handler(r.deleteNodeDeployment())

	//
	// Defines a set of HTTP endpoints for various cloud providers
	// Note that these endpoints don't require credentials as opposed to the ones defined under /providers/*
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/digitalocean/sizes").
		Handler(r.listDigitaloceanSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/azure/sizes").
		Handler(r.listAzureSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/sizes").
		Handler(r.listOpenstackSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/tenants").
		Handler(r.listOpenstackTenantsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/networks").
		Handler(r.listOpenstackNetworksNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/securitygroups").
		Handler(r.listOpenstackSecurityGroupsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/subnets").
		Handler(r.listOpenstackSubnetsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/vsphere/networks").
		Handler(r.listVSphereNetworksNoCredentials())

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

// swagger:route GET /api/v1/projects/{project_id}/sshkeys project listSSHKeys
//
//     Lists SSH Keys that belong to the given project.
//     The returned collection is sorted by creation timestamp.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []SSHKey
//       401: empty
//       403: empty
func (r Routing) listSSHKeys() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(listSSHKeyEndpoint(r.sshKeyProvider, r.projectProvider)),
		decodeListSSHKeyReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/sshkeys project createSSHKey
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
//       200: SSHKey
//       401: empty
//       403: empty
func (r Routing) createSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(createSSHKeyEndpoint(r.sshKeyProvider, r.projectProvider)),
		decodeCreateSSHKeyReq,
		setStatusCreatedHeader(EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/sshkeys/{key_id} project deleteSSHKey
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
func (r Routing) deleteSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(deleteSSHKeyEndpoint(r.sshKeyProvider, r.projectProvider)),
		decodeDeleteSSHKeyReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.DigitaloceanSizeEndpoint()),
		provider.DecodeDoSizesReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureSizeEndpoint()),
		provider.DecodeAzureSizesReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.OpenstackSizeEndpoint(r.cloudProviders)),
		provider.DecodeOpenstackReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.VsphereNetworksEndpoint(r.cloudProviders)),
		provider.DecodeVSphereNetworksReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.OpenstackTenantEndpoint(r.cloudProviders)),
		provider.DecodeOpenstackTenantReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.OpenstackNetworkEndpoint(r.cloudProviders)),
		provider.DecodeOpenstackReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.OpenstackSubnetsEndpoint(r.cloudProviders)),
		provider.DecodeOpenstackSubnetReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(provider.OpenstackSecurityGroupEndpoint(r.cloudProviders)),
		provider.DecodeOpenstackReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(dc.ListEndpoint(r.datacenters)),
		dc.DecodeDatacentersReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(dc.GetEndpoint(r.datacenters)),
		dc.DecodeLegacyDcReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(getMasterVersions(r.updateManager)),
		decodeEmptyReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/version versions getKubermaticVersion
//
// Get versions of running Kubermatic components.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: KubermaticVersions
func (r Routing) getKubermaticVersion() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(getKubermaticVersion()),
		decodeEmptyReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(project.ListEndpoint(r.projectProvider, r.userProjectMapper, r.projectMemberProvider, r.userProvider)),
		decodeEmptyReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(project.GetEndpoint(r.projectProvider, r.projectMemberProvider, r.userProvider)),
		common.DecodeGetProject,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(project.CreateEndpoint(r.projectProvider)),
		project.DecodeCreate,
		setStatusCreatedHeader(EncodeJSON),
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
//       201: Project
//       400: empty
//       404: empty
//       500: empty
//       501: empty
func (r Routing) updateProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(project.UpdateEndpoint(r.projectProvider, r.projectMemberProvider, r.userProvider)),
		project.DecodeUpdateRq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(project.DeleteEndpoint(r.projectProvider)),
		project.DecodeDelete,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/dc/{dc}/clusters project createCluster
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
func (r Routing) createCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(createClusterEndpoint(r.cloudProviders, r.projectProvider, r.datacenters)),
		decodeCreateClusterReq,
		setStatusCreatedHeader(EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters project listClusters
//
//     Lists clusters for the specified project and data center.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterList
//       401: empty
//       403: empty
func (r Routing) listClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(listClusters(r.projectProvider)),
		decodeListClustersReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/clusters project listClustersForProject
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
func (r Routing) listClustersForProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(listClustersForProject(r.projectProvider, r.clusterProviders)),
		common.DecodeGetProject,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id} project getCluster
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
func (r Routing) getCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(getCluster(r.projectProvider)),
		common.DecodeGetClusterReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id} project patchCluster
//
//     Patches the given cluster using JSON Merge Patch method (https://tools.ietf.org/html/rfc7396).
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Cluster
//       401: empty
//       403: empty
func (r Routing) patchCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(patchCluster(r.cloudProviders, r.projectProvider, r.datacenters)),
		decodePatchClusterReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// getClusterKubeconfig returns the kubeconfig for the cluster.
// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/kubeconfig project getClusterKubeconfig
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
func (r Routing) getClusterKubeconfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(getClusterKubeconfig(r.projectProvider)),
		decodeGetClusterKubeconfig,
		encodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id} project deleteCluster
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
func (r Routing) deleteCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(deleteCluster(r.sshKeyProvider, r.projectProvider)),
		DecodeDeleteClusterReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/health project getClusterHealth
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
func (r Routing) getClusterHealth() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(getClusterHealth(r.projectProvider)),
		common.DecodeGetClusterReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id} project assignSSHKeyToCluster
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
func (r Routing) assignSSHKeyToCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(assignSSHKeyToCluster(r.sshKeyProvider, r.projectProvider)),
		decodeAssignSSHKeyToClusterReq,
		setStatusCreatedHeader(EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys project listSSHKeysAssignedToCluster
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
//       200: []SSHKey
//       401: empty
//       403: empty
func (r Routing) listSSHKeysAssignedToCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(listSSHKeysAssingedToCluster(r.sshKeyProvider, r.projectProvider)),
		decodeListSSHKeysAssignedToCluster,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/sshkeys/{key_id} project detachSSHKeyFromCluster
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
func (r Routing) detachSSHKeyFromCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(detachSSHKeyFromCluster(r.sshKeyProvider, r.projectProvider)),
		decodeDetachSSHKeysFromCluster,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/token project revokeClusterAdminToken
//
//     Revokes the current admin token
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) revokeClusterAdminToken() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(revokeClusterAdminToken(r.projectProvider)),
		decodeClusterAdminTokenReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id} project getNodeForClusterLegacy
//
//     Deprecated:
//     Gets a node that is assigned to the given cluster.
//
//     This endpoint is deprecated, please create a Node Deployment instead.
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
func (r Routing) getNodeForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(getNodeForClusterLegacy(r.projectProvider)),
		decodeGetNodeForClusterLegacy,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes project createNodeForClusterLegacy
//
//     Deprecated:
//     Creates a node that will belong to the given cluster
//
//     This endpoint is deprecated, please create a Node Deployment instead.
//     Use POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments
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
func (r Routing) createNodeForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(createNodeForClusterLegacy(r.sshKeyProvider, r.projectProvider, r.datacenters)),
		decodeCreateNodeForClusterLegacy,
		setStatusCreatedHeader(EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes project listNodesForClusterLegacy
//
//     Deprecated:
//     Lists nodes that belong to the given cluster
//
//     This endpoint is deprecated, please create a Node Deployment instead.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Node
//       401: empty
//       403: empty
func (r Routing) listNodesForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(listNodesForClusterLegacy(r.projectProvider)),
		decodeListNodesForClusterLegacy,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id} project deleteNodeForClusterLegacy
//
//    Deprecated:
//    Deletes the given node that belongs to the cluster.
//
//     This endpoint is deprecated, please create a Node Deployment instead.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteNodeForClusterLegacy() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(deleteNodeForClusterLegacy(r.projectProvider)),
		decodeDeleteNodeForClusterLegacy,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(getClusterUpgrades(r.updateManager, r.projectProvider)),
		common.DecodeGetClusterReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/upgrades project getClusterNodeUpgrades
//
//    Gets possible node upgrades for a cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []MasterVersion
//       401: empty
//       403: empty
func (r Routing) getClusterNodeUpgrades() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(getClusterNodeUpgrades(r.updateManager, r.projectProvider)),
		common.DecodeGetClusterReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(user.AddEndpoint(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		user.DecodeAddReq,
		setStatusCreatedHeader(EncodeJSON),
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(user.ListEndpoint(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		common.DecodeGetProject,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(user.EditEndpoint(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		user.DecodeEditReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.UserInfo(r.userProjectMapper),
		)(user.DeleteEndpoint(r.projectProvider, r.userProvider, r.projectMemberProvider)),
		user.DecodeDeleteReq,
		EncodeJSON,
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
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
		)(user.GetEndpoint(r.userProjectMapper)),
		decodeEmptyReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/digitalocean/sizes digitalocean listDigitaloceanSizesNoCredentials
//
// Lists sizes from digitalocean
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: DigitaloceanSizeList
func (r Routing) listDigitaloceanSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.DigitaloceanSizeNoCredentialsEndpoint(r.projectProvider)),
		provider.DecodeDoSizesNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/azure/sizes azure listAzureSizesNoCredentials
//
// Lists available VM sizes in an Azure region
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureSizeList
func (r Routing) listAzureSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.AzureSizeNoCredentialsEndpoint(r.projectProvider, r.datacenters)),
		provider.DecodeAzureSizesNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/sizes openstack listOpenstackSizesNoCredentials
//
// Lists sizes from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackSize
func (r Routing) listOpenstackSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.OpenstackSizeNoCredentialsEndpoint(r.projectProvider, r.cloudProviders)),
		provider.DecodeOpenstackNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/tenants openstack listOpenstackTenantsNoCredentials
//
// Lists tenants from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackTenant
func (r Routing) listOpenstackTenantsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.OpenstackTenantNoCredentialsEndpoint(r.projectProvider, r.cloudProviders)),
		provider.DecodeOpenstackNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/networks openstack listOpenstackNetworksNoCredentials
//
// Lists networks from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackNetwork
func (r Routing) listOpenstackNetworksNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.OpenstackNetworkNoCredentialsEndpoint(r.projectProvider, r.cloudProviders)),
		provider.DecodeOpenstackNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/securitygroups openstack listOpenstackSecurityGroupsNoCredentials
//
// Lists security groups from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackSecurityGroup
func (r Routing) listOpenstackSecurityGroupsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.OpenstackSecurityGroupNoCredentialsEndpoint(r.projectProvider, r.cloudProviders)),
		provider.DecodeOpenstackNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/openstack/subnets openstack listOpenstackSubnetsNoCredentials
//
// Lists subnets from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackSubnet
func (r Routing) listOpenstackSubnetsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.OpenstackSubnetsNoCredentialsEndpoint(r.projectProvider, r.cloudProviders)),
		provider.DecodeOpenstackSubnetNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/vsphere/networks vsphere listVSphereNetworksNoCredentials
//
// Lists networks from vsphere datacenter
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []VSphereNetwork
func (r Routing) listVSphereNetworksNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(provider.VsphereNetworksNoCredentialsEndpoint(r.projectProvider, r.cloudProviders)),
		provider.DecodeVSphereNetworksNoCredentialsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments project createNodeDeployment
//
//     Creates a node deployment that will belong to the given cluster
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: NodeDeployment
//       401: empty
//       403: empty
func (r Routing) createNodeDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(createNodeDeployment(r.sshKeyProvider, r.projectProvider, r.datacenters)),
		decodeCreateNodeDeployment,
		setStatusCreatedHeader(EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments project listNodeDeployments
//
//     Lists node deployments that belong to the given cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []NodeDeployment
//       401: empty
//       403: empty
func (r Routing) listNodeDeployments() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(listNodeDeployments(r.projectProvider)),
		decodeListNodeDeployments,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id} project getNodeDeployment
//
//     Gets a node deployment that is assigned to the given cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeDeployment
//       401: empty
//       403: empty
func (r Routing) getNodeDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(getNodeDeployment(r.projectProvider)),
		decodeGetNodeDeployment,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes project listNodeDeploymentNodes
//
//     Lists nodes that belong to the given node deployment.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Node
//       401: empty
//       403: empty
func (r Routing) listNodeDeploymentNodes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(listNodeDeploymentNodes(r.projectProvider)),
		decodeListNodeDeploymentNodes,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/nodes/events project listNodeDeploymentNodesEvents
//
//     Lists node deployment events. If query parameter `type` is set to `warning` then only warning events are retrieved.
//     If the value is 'normal' then normal events are returned. If the query parameter is missing method returns all events.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Event
//       401: empty
//       403: empty
func (r Routing) listNodeDeploymentNodesEvents() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(listNodeDeploymentNodesEvents()),
		decodeListNodeDeploymentNodesEvents,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id} project patchNodeDeployment
//
//     Patches a node deployment that is assigned to the given cluster. Please note that at the moment only
//	   node deployment's spec can be updated by a patch, no other fields can be changed using this endpoint.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeDeployment
//       401: empty
//       403: empty
func (r Routing) patchNodeDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(patchNodeDeployment(r.sshKeyProvider, r.projectProvider, r.datacenters)),
		decodePatchNodeDeployment,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id} project deleteNodeDeployment
//
//    Deletes the given node deployment that belongs to the cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteNodeDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.oidcAuthenticator.Verifier(),
			middleware.UserSaver(r.userProvider),
			middleware.Datacenter(r.clusterProviders, r.datacenters),
			middleware.UserInfo(r.userProjectMapper),
		)(deleteNodeDeployment(r.projectProvider)),
		decodeDeleteNodeDeployment,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}
