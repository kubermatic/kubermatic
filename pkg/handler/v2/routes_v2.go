/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"k8c.io/kubermatic/v2/pkg/handler"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/addon"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/handler/v2/constraint"
	constrainttemplate "k8c.io/kubermatic/v2/pkg/handler/v2/constraint_template"
	externalcluster "k8c.io/kubermatic/v2/pkg/handler/v2/external_cluster"
	"k8c.io/kubermatic/v2/pkg/handler/v2/gatekeeperconfig"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/handler/v2/kubernetes-dashboard"
	"k8c.io/kubermatic/v2/pkg/handler/v2/machine"
	"k8c.io/kubermatic/v2/pkg/handler/v2/preset"
	"k8c.io/kubermatic/v2/pkg/handler/v2/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v2/serviceaccount"
)

// RegisterV2 declares all router paths for v2
func (r Routing) RegisterV2(mux *mux.Router, metrics common.ServerMetrics) {

	// Defines a set of HTTP endpoints for cluster that belong to a project.
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters").
		Handler(r.createCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters").
		Handler(r.listClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}").
		Handler(r.getCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}").
		Handler(r.deleteCluster())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/clusters/{cluster_id}").
		Handler(r.patchCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/events").
		Handler(r.getClusterEvents())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/health").
		Handler(r.getClusterHealth())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/kubeconfig").
		Handler(r.getClusterKubeconfig())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/token").
		Handler(r.revokeClusterAdminToken())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/viewertoken").
		Handler(r.revokeClusterViewerToken())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/oidckubeconfig").
		Handler(r.getOidcClusterKubeconfig())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/metrics").
		Handler(r.getClusterMetrics())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/namespaces").
		Handler(r.listNamespace())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/upgrades").
		Handler(r.getClusterUpgrades())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/nodes/upgrades").
		Handler(r.upgradeClusterNodeDeployments())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/clusterroles").
		Handler(r.listClusterRole())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/clusterrolenames").
		Handler(r.listClusterRoleNames())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/roles").
		Handler(r.listRole())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/rolenames").
		Handler(r.listRoleNames())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings").
		Handler(r.bindUserToRole())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings").
		Handler(r.bindUserToClusterRole())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings").
		Handler(r.unbindUserFromRoleBinding())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings").
		Handler(r.unbindUserFromClusterRoleBinding())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/bindings").
		Handler(r.listRoleBinding())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/clusterbindings").
		Handler(r.listClusterRoleBinding())

	// Defines a set of HTTP endpoint for machine deployments that belong to a cluster
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments").
		Handler(r.createMachineDeployment())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id}").
		Handler(r.deleteMachineDeploymentNode())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments").
		Handler(r.listMachineDeployments())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}").
		Handler(r.getMachineDeployment())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes").
		Handler(r.listMachineDeploymentNodes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/nodes").
		Handler(r.listNodesForCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/metrics").
		Handler(r.listMachineDeploymentMetrics())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}").
		Handler(r.patchMachineDeployment())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events").
		Handler(r.listMachineDeploymentNodesEvents())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}").
		Handler(r.deleteMachineDeployment())

	// Defines set of HTTP endpoints for SSH Keys that belong to a cluster
	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id}").
		Handler(r.assignSSHKeyToCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id}").
		Handler(r.detachSSHKeyFromCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/sshkeys").
		Handler(r.listSSHKeysAssignedToCluster())

	// Defines a set of HTTP endpoints for external cluster that belong to a project.
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/kubernetes/clusters").
		Handler(r.createExternalCluster())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}").
		Handler(r.deleteExternalCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters").
		Handler(r.listExternalClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}").
		Handler(r.getExternalCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/metrics").
		Handler(r.getExternalClusterMetrics())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}").
		Handler(r.updateExternalCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodes").
		Handler(r.listExternalClusterNodes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.getExternalClusterNode())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodesmetrics").
		Handler(r.listExternalClusterNodesMetrics())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/events").
		Handler(r.listExternalClusterEvents())

	// Define a set of endpoints for gatekeeper constraint templates
	mux.Methods(http.MethodGet).
		Path("/constrainttemplates").
		Handler(r.listConstraintTemplates())

	mux.Methods(http.MethodGet).
		Path("/constrainttemplates/{ct_name}").
		Handler(r.getConstraintTemplate())

	mux.Methods(http.MethodPost).
		Path("/constrainttemplates").
		Handler(r.createConstraintTemplate())

	mux.Methods(http.MethodPatch).
		Path("/constrainttemplates/{ct_name}").
		Handler(r.patchConstraintTemplate())

	mux.Methods(http.MethodDelete).
		Path("/constrainttemplates/{ct_name}").
		Handler(r.deleteConstraintTemplate())

	// Define a set of endpoints for gatekeeper constraints
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/constraints").
		Handler(r.listConstraints())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}").
		Handler(r.getConstraint())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}").
		Handler(r.deleteConstraint())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/constraints").
		Handler(r.createConstraint())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name}").
		Handler(r.patchConstraint())

	// Defines a set of HTTP endpoints for managing gatekeeper config
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config").
		Handler(r.getGatekeeperConfig())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config").
		Handler(r.deleteGatekeeperConfig())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config").
		Handler(r.createGatekeeperConfig())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config").
		Handler(r.patchGatekeeperConfig())

	// Defines a set of HTTP endpoints for managing addons
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/installableaddons").
		Handler(r.listInstallableAddons())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/addons").
		Handler(r.createAddon())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/addons").
		Handler(r.listAddons())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}").
		Handler(r.getAddon())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}").
		Handler(r.patchAddon())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}").
		Handler(r.deleteAddon())

	// Defines a set of HTTP endpoints for various cloud providers
	// Note that these endpoints don't require credentials as opposed to the ones defined under /providers/*
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/aws/sizes").
		Handler(r.listAWSSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/aws/subnets").
		Handler(r.listAWSSubnetsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/gcp/disktypes").
		Handler(r.listGCPDiskTypesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/gcp/sizes").
		Handler(r.listGCPSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/gcp/zones").
		Handler(r.listGCPZonesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/gcp/networks").
		Handler(r.listGCPNetworksNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/gcp/subnetworks").
		Handler(r.listGCPSubnetworksNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/hetzner/sizes").
		Handler(r.listHetznerSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/digitalocean/sizes").
		Handler(r.listDigitaloceanSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/openstack/sizes").
		Handler(r.listOpenstackSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/openstack/tenants").
		Handler(r.listOpenstackTenantsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/openstack/networks").
		Handler(r.listOpenstackNetworksNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/openstack/securitygroups").
		Handler(r.listOpenstackSecurityGroupsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/openstack/subnets").
		Handler(r.listOpenstackSubnetsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/openstack/availabilityzones").
		Handler(r.listOpenstackAvailabilityZonesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/azure/sizes").
		Handler(r.listAzureSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/azure/availabilityzones").
		Handler(r.listAzureAvailabilityZonesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/vsphere/networks").
		Handler(r.listVSphereNetworksNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/vsphere/folders").
		Handler(r.listVSphereFoldersNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/instancetypes").
		Handler(r.listAlibabaInstanceTypesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/zones").
		Handler(r.listAlibabaZonesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/packet/sizes").
		Handler(r.listPacketSizesNoCredentials())

	// Defines a set of kubernetes-dashboard-specific endpoints
	mux.PathPrefix("/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy").
		Handler(r.kubernetesDashboardProxy())

	// Defines a set of HTTP endpoint for interacting with
	// various cloud providers
	mux.Methods(http.MethodGet).
		Path("/providers/azure/securitygroups").
		Handler(r.listAzureSecurityGroups())

	mux.Methods(http.MethodGet).
		Path("/providers/azure/resourcegroups").
		Handler(r.listAzureResourceGroups())

	mux.Methods(http.MethodGet).
		Path("/providers/azure/routetables").
		Handler(r.listAzureRouteTables())

	mux.Methods(http.MethodGet).
		Path("/providers/azure/subnets").
		Handler(r.listAzureSubnets())

	mux.Methods(http.MethodGet).
		Path("/providers/azure/vnets").
		Handler(r.listAzureVnets())

	mux.Methods(http.MethodGet).
		Path("/providers/vsphere/datastores").
		Handler(r.listVSphereDatastores())

	// Define a set of endpoints for preset management
	mux.Methods(http.MethodGet).
		Path("/presets").
		Handler(r.listPresets())

	mux.Methods(http.MethodPut).
		Path("/presets/{preset_name}/status").
		Handler(r.updatePresetStatus())

	mux.Methods(http.MethodGet).
		Path("/providers/{provider_name}/presets").
		Handler(r.listProviderPresets())

	mux.Methods(http.MethodPost).
		Path("/providers/{provider_name}/presets").
		Handler(r.createPreset())

	mux.Methods(http.MethodPut).
		Path("/providers/{provider_name}/presets").
		Handler(r.updatePreset())

	// Define a set of endpoints for service accounts management
	mux.Methods(http.MethodPost).
		Path("/serviceaccounts").
		Handler(r.createMainServiceAccount())
	mux.Methods(http.MethodGet).
		Path("/serviceaccounts").
		Handler(r.listMainServiceAccounts())
	mux.Methods(http.MethodPut).
		Path("/serviceaccounts/{serviceaccount_id}").
		Handler(r.updateMainServiceAccount())
	mux.Methods(http.MethodDelete).
		Path("/serviceaccounts/{serviceaccount_id}").
		Handler(r.deleteMainServiceAccount())

	// Defines set of HTTP endpoints for tokens of the given service account
	mux.Methods(http.MethodPost).
		Path("/serviceaccounts/{serviceaccount_id}/tokens").
		Handler(r.addTokenToMainServiceAccount())
	mux.Methods(http.MethodGet).
		Path("/serviceaccounts/{serviceaccount_id}/tokens").
		Handler(r.listMainServiceAccountTokens())
	mux.Methods(http.MethodPut).
		Path("/serviceaccounts/{serviceaccount_id}/tokens/{token_id}").
		Handler(r.updateMainServiceAccountToken())
	mux.Methods(http.MethodPatch).
		Path("/serviceaccounts/{serviceaccount_id}/tokens/{token_id}").
		Handler(r.patchMainServiceAccountToken())
	mux.Methods(http.MethodDelete).
		Path("/serviceaccounts/{serviceaccount_id}/tokens/{token_id}").
		Handler(r.deleteMainServiceAccountToken())
}

// swagger:route POST /api/v2/projects/{project_id}/clusters project createClusterV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.CreateEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.presetsProvider, r.exposeStrategy, r.userInfoGetter, r.settingsProvider, r.updateManager)),
		cluster.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters project listClustersV2
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
func (r Routing) listClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(cluster.ListEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.clusterProviderGetter, r.userInfoGetter)),
		common.DecodeGetProject,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id} project getClusterV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id} project deleteClusterV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.DeleteEndpoint(r.sshKeyProvider, r.privilegedSSHKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeDeleteReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/clusters/{cluster_id} project patchClusterV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.PatchEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		cluster.DecodePatchReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// getClusterEvents returns events related to the cluster.
// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/events project getClusterEventsV2
//
//     Gets the events related to the specified cluster.
//
//     Produces:
//     - application/yaml
//
//     Responses:
//       default: errorResponse
//       200: []Event
//       401: empty
//       403: empty
func (r Routing) getClusterEvents() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetClusterEventsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterEvents,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/health project getClusterHealthV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.HealthEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// getClusterKubeconfig returns the kubeconfig for the cluster.
// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/kubeconfig project getClusterKubeconfigV2
//
//     Gets the kubeconfig for the specified cluster.
//
//     Produces:
//     - application/octet-stream
//
//     Responses:
//       default: errorResponse
//       200: Kubeconfig
//       401: empty
//       403: empty
func (r Routing) getClusterKubeconfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetAdminKubeconfigEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		cluster.EncodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// getOidcClusterKubeconfig returns the oidc kubeconfig for the cluster.
// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/oidckubeconfig project getOidcClusterKubeconfigV2
//
//     Gets the kubeconfig for the specified cluster with oidc authentication.
//
//     Produces:
//     - application/octet-stream
//
//     Responses:
//       default: errorResponse
//       200: Kubeconfig
//       401: empty
//       403: empty
func (r Routing) getOidcClusterKubeconfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetOidcKubeconfigEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		cluster.EncodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/metrics project getClusterMetricsV2
//
//    Gets cluster metrics
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterMetrics
//       401: empty
//       403: empty
func (r Routing) getClusterMetrics() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetMetricsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/namespaces project listNamespaceV2
//
//     Lists all namespaces in the cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Namespace
//       401: empty
//       403: empty
func (r Routing) listNamespace() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListNamespaceEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/upgrades project getClusterUpgradesV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetUpgradesEndpoint(r.updateManager, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/nodes/upgrades project upgradeClusterNodeDeploymentsV2
//
//    Upgrades node deployments in a cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) upgradeClusterNodeDeployments() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.UpgradeNodeDeploymentsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeUpgradeNodeDeploymentsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id} project assignSSHKeyToClusterV2
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
//       201: SSHKey
//       401: empty
//       403: empty
func (r Routing) assignSSHKeyToCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.AssignSSHKeyEndpoint(r.sshKeyProvider, r.privilegedSSHKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeAssignSSHKeyReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/projects/{project_id}/clusters/{cluster_id}/sshkeys/{key_id} project detachSSHKeyFromClusterV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.DetachSSHKeyEndpoint(r.sshKeyProvider, r.privilegedSSHKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeAssignSSHKeyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/sshkeys project listSSHKeysAssignedToClusterV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListSSHKeysEndpoint(r.sshKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeListSSHKeysReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/kubernetes/clusters project createExternalCluster
//
//     Creates an external cluster for the given project.
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
func (r Routing) createExternalCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// Delete the external cluster
// swagger:route DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id} project deleteExternalCluster
//
//     Deletes the specified external cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteExternalCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeDeleteReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters project listExternalClusters
//
//     Lists external clusters for the specified project.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterList
//       401: empty
//       403: empty
func (r Routing) listExternalClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id} project getExternalCluster
//
//     Gets an external cluster for the given project.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Cluster
//       401: empty
//       403: empty
func (r Routing) getExternalCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id} project updateExternalCluster
//
//     Updates an external cluster for the given project.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Cluster
//       401: empty
//       403: empty
func (r Routing) updateExternalCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.UpdateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeUpdateReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodes project listExternalClusterNodes
//
//     Gets an external cluster nodes.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Node
//       401: empty
//       403: empty
func (r Routing) listExternalClusterNodes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListNodesEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeListNodesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodes/{node_id} project getExternalClusterNode
//
//     Gets an external cluster node.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Node
//       401: empty
//       403: empty
func (r Routing) getExternalClusterNode() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GetNodeEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetNodeReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/metrics project getExternalClusterMetrics
//
//     Gets cluster metrics
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterMetrics
//       401: empty
//       403: empty
func (r Routing) getExternalClusterMetrics() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GetMetricsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodesmetrics project listExternalClusterNodesMetrics
//
//     Gets an external cluster nodes metrics.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []NodeMetric
//       401: empty
//       403: empty
func (r Routing) listExternalClusterNodesMetrics() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListNodesMetricsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeListNodesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/events project listExternalClusterEvents
//
//     Gets an external cluster events.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Event
//       401: empty
//       403: empty
func (r Routing) listExternalClusterEvents() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEventsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeListEventsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/constrainttemplates constrainttemplates listConstraintTemplates
//
//     List constraint templates.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []ConstraintTemplate
//       401: empty
//       403: empty
func (r Routing) listConstraintTemplates() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constrainttemplate.ListEndpoint(r.constraintTemplateProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/constrainttemplates/{ct_name} constrainttemplates getConstraintTemplate
//
//     Get constraint templates specified by name
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ConstraintTemplate
//       401: empty
//       403: empty
func (r Routing) getConstraintTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constrainttemplate.GetEndpoint(r.constraintTemplateProvider)),
		constrainttemplate.DecodeConstraintTemplateRequest,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/constrainttemplates constrainttemplates createConstraintTemplate
//
//     Create constraint template
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ConstraintTemplate
//       401: empty
//       403: empty
func (r Routing) createConstraintTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constrainttemplate.CreateEndpoint(r.userInfoGetter, r.constraintTemplateProvider)),
		constrainttemplate.DecodeCreateConstraintTemplateRequest,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/constrainttemplates/{ct_name} constrainttemplates patchConstraintTemplate
//
//     Patch a specified constraint template
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ConstraintTemplate
//       401: empty
//       403: empty
func (r Routing) patchConstraintTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constrainttemplate.PatchEndpoint(r.userInfoGetter, r.constraintTemplateProvider)),
		constrainttemplate.DecodePatchConstraintTemplateReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/constrainttemplates/{ct_name} constrainttemplates deleteConstraintTemplate
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
func (r Routing) deleteConstraintTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constrainttemplate.DeleteEndpoint(r.userInfoGetter, r.constraintTemplateProvider)),
		constrainttemplate.DecodeConstraintTemplateRequest,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints project listConstraints
//
//     Lists constraints for the specified cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Constraint
//       401: empty
//       403: empty
func (r Routing) listConstraints() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(constraint.ListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.constraintProvider)),
		constraint.DecodeListConstraintsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name} project getConstraint
//
//     Gets an specified constraint for the given cluster.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Constraint
//       401: empty
//       403: empty
func (r Routing) getConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(constraint.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.constraintProvider)),
		constraint.DecodeConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name} project deleteConstraint
//
//     Deletes a specified constraint for the given cluster.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(constraint.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.constraintProvider, r.privilegedConstraintProvider)),
		constraint.DecodeConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints project createConstraint
//
//     Creates a given constraint for the specified cluster.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Constraint
//       401: empty
//       403: empty
func (r Routing) createConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(constraint.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.constraintProvider, r.privilegedConstraintProvider, r.constraintTemplateProvider)),
		constraint.DecodeCreateConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name} project patchConstraint
//
//     Patches a given constraint for the specified cluster.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Constraint
//       401: empty
//       403: empty
func (r Routing) patchConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(constraint.PatchEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.constraintProvider, r.privilegedConstraintProvider)),
		constraint.DecodePatchConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config project getGatekeeperConfig
//
//     Gets the gatekeeper sync config for the specified cluster.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GatekeeperConfig
//       401: empty
//       403: empty
func (r Routing) getGatekeeperConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(gatekeeperconfig.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		gatekeeperconfig.DecodeGatkeeperConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config project deleteGatekeeperConfig
//
//     Deletes the gatekeeper sync config for the specified cluster.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteGatekeeperConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(gatekeeperconfig.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		gatekeeperconfig.DecodeGatkeeperConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config project createGatekeeperConfig
//
//     Creates a gatekeeper config for the given cluster
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: GatekeeperConfig
//       401: empty
//       403: empty
func (r Routing) createGatekeeperConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(gatekeeperconfig.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		gatekeeperconfig.DecodeCreateGatkeeperConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config project patchGatekeeperConfig
//
//     Patches the gatekeeper config for the specified cluster.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GatekeeperConfig
//       401: empty
//       403: empty
func (r Routing) patchGatekeeperConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(gatekeeperconfig.PatchEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		gatekeeperconfig.DecodePatchGatekeeperConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments project createMachineDeployment
//
//     Creates a machine deployment that will belong to the given cluster
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
func (r Routing) createMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.CreateMachineDeployment(r.sshKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		machine.DecodeCreateMachineDeployment,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/nodes/{node_id} project deleteMachineDeploymentNode
//
//    Deletes the given node that belongs to the machine deployment.
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteMachineDeploymentNode() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.DeleteMachineDeploymentNode(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeDeleteMachineDeploymentNode,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments project listMachineDeployments
//
//     Lists machine deployments that belong to the given cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []NodeDeployment
//       401: empty
//       403: empty
func (r Routing) listMachineDeployments() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.ListMachineDeployments(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeListMachineDeployments,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id} project getMachineDeployment
//
//     Gets a machine deployment that is assigned to the given cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: NodeDeployment
//       401: empty
//       403: empty
func (r Routing) getMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.GetMachineDeployment(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeGetMachineDeployment,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes project listMachineDeploymentNodes
//
//     Lists nodes that belong to the given machine deployment.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Node
//       401: empty
//       403: empty
func (r Routing) listMachineDeploymentNodes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.ListMachineDeploymentNodes(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeListMachineDeploymentNodes,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/nodes project listNodesForCluster
//
//
//     This endpoint is used for kubeadm cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Node
//       401: empty
//       403: empty
func (r Routing) listNodesForCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.ListNodesForCluster(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeListNodesForCluster,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/metrics metric listMachineDeploymentMetrics
//
//     Lists metrics that belong to the given machine deployment.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []NodeMetric
//       401: empty
//       403: empty
func (r Routing) listMachineDeploymentMetrics() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.ListMachineDeploymentMetrics(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeListMachineDeploymentMetrics,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id} project patchMachineDeployment
//
//     Patches a machine deployment that is assigned to the given cluster. Please note that at the moment only
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
func (r Routing) patchMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.PatchMachineDeployment(r.sshKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		machine.DecodePatchMachineDeployment,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events project listMachineDeploymentNodesEvents
//
//     Lists machine deployment events. If query parameter `type` is set to `warning` then only warning events are retrieved.
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
func (r Routing) listMachineDeploymentNodesEvents() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.ListMachineDeploymentNodesEvents(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeListNodeDeploymentNodesEvents,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id} project deleteMachineDeployment
//
//    Deletes the given machine deployment that belongs to the cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.DeleteMachineDeployment(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeDeleteMachineDeployment,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/clusterroles project listClusterRoleV2
//
//     Lists all ClusterRoles
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []ClusterRole
//       401: empty
//       403: empty
func (r Routing) listClusterRole() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListClusterRoleEndpoint(r.userInfoGetter)),
		cluster.DecodeListClusterRoleReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/clusterrolenames project listClusterRoleNamesV2
//
//     Lists all ClusterRoles
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []ClusterRoleName
//       401: empty
//       403: empty
func (r Routing) listClusterRoleNames() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListClusterRoleNamesEndpoint(r.userInfoGetter)),
		cluster.DecodeListClusterRoleReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/roles project listRoleV2
//
//     Lists all Roles
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Role
//       401: empty
//       403: empty
func (r Routing) listRole() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListRoleEndpoint(r.userInfoGetter)),
		cluster.DecodeListClusterRoleReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rolenames project listRoleNamesV2
//
//     Lists all Role names with namespaces
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []RoleName
//       401: empty
//       403: empty
func (r Routing) listRoleNames() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListRoleNamesEndpoint(r.userInfoGetter)),
		cluster.DecodeListClusterRoleReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings project bindUserToRoleV2
//
//    Binds user to the role
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: RoleBinding
//       401: empty
//       403: empty
func (r Routing) bindUserToRole() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.BindUserToRoleEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeRoleUserReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings project bindUserToClusterRoleV2
//
//    Binds user to cluster role
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterRoleBinding
//       401: empty
//       403: empty
func (r Routing) bindUserToClusterRole() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.BindUserToClusterRoleEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeClusterRoleUserReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings project unbindUserFromRoleBindingV2
//
//    Unbinds user from the role binding
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: RoleBinding
//       401: empty
//       403: empty
func (r Routing) unbindUserFromRoleBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.UnbindUserFromRoleBindingEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeRoleUserReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/clusterroles/{role_id}/clusterbindings project unbindUserFromClusterRoleBindingV2
//
//    Unbinds user from cluster role binding
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ClusterRoleBinding
//       401: empty
//       403: empty
func (r Routing) unbindUserFromClusterRoleBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.UnbindUserFromClusterRoleBindingEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeClusterRoleUserReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/clusterbindings project listClusterRoleBindingV2
//
//    List cluster role binding
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []ClusterRoleBinding
//       401: empty
//       403: empty
func (r Routing) listClusterRoleBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListClusterRoleBindingEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeListBindingReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/bindings project listRoleBindingV2
//
//    List role binding
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []RoleBinding
//       401: empty
//       403: empty
func (r Routing) listRoleBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListRoleBindingEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeListBindingReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/installableaddons addon listInstallableAddonsV2
//
//     Lists names of addons that can be installed inside the user cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AccessibleAddons
//       401: empty
//       403: empty
func (r Routing) listInstallableAddons() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Addons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
			middleware.PrivilegedAddons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
		)(addon.ListInstallableAddonEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.accessibleAddons)),
		addon.DecodeListAddons,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/addons addon createAddonV2
//
//     Creates an addon that will belong to the given cluster
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: Addon
//       401: empty
//       403: empty
func (r Routing) createAddon() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Addons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
			middleware.PrivilegedAddons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
		)(addon.CreateAddonEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		addon.DecodeCreateAddon,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/addons addon listAddonsV2
//
//     Lists addons that belong to the given cluster
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Addon
//       401: empty
//       403: empty
func (r Routing) listAddons() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Addons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
			middleware.PrivilegedAddons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
		)(addon.ListAddonEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		addon.DecodeListAddons,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id} addon getAddonV2
//
//     Gets an addon that is assigned to the given cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Addon
//       401: empty
//       403: empty
func (r Routing) getAddon() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Addons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
			middleware.PrivilegedAddons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
		)(addon.GetAddonEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		addon.DecodeGetAddon,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id} addon patchAddonV2
//
//     Patches an addon that is assigned to the given cluster.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Addon
//       401: empty
//       403: empty
func (r Routing) patchAddon() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Addons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
			middleware.PrivilegedAddons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
		)(addon.PatchAddonEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		addon.DecodePatchAddon,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id} addon deleteAddonV2
//
//    Deletes the given addon that belongs to the cluster.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteAddon() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Addons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
			middleware.PrivilegedAddons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
		)(addon.DeleteAddonEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		addon.DecodeGetAddon,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/aws/sizes aws listAWSSizesNoCredentialsV2
//
// Lists available AWS sizes
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AWSSizeList
func (r Routing) listAWSSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AWSSizeNoCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.settingsProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/aws/subnets aws listAWSSubnetsNoCredentialsV2
//
// Lists available AWS subnets
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AWSSubnetList
func (r Routing) listAWSSubnetsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AWSSubnetNoCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/sizes gcp listGCPSizesNoCredentialsV2
//
// Lists machine sizes from GCP
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GCPMachineSizeList
func (r Routing) listGCPSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.GCPSizeWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeGCPTypesNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/disktypes gcp listGCPDiskTypesNoCredentialsV2
//
// Lists disk types from GCP
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GCPDiskTypeList
func (r Routing) listGCPDiskTypesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.GCPDiskTypesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		provider.DecodeGCPTypesNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/zones gcp listGCPZonesNoCredentialsV2
//
// Lists available GCP zones
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GCPZoneList
func (r Routing) listGCPZonesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.GCPZoneWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/networks gcp listGCPNetworksNoCredentialsV2
//
// Lists available GCP networks
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GCPNetworkList
func (r Routing) listGCPNetworksNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.GCPNetworkWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/gcp/subnetworks gcp listGCPSubnetworksNoCredentialsV2
//
// Lists available GCP subnetworks
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GCPSubnetworkList
func (r Routing) listGCPSubnetworksNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.GCPSubnetworkWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeGCPSubnetworksNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/token project revokeClusterAdminTokenV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.RevokeAdminTokenEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeAdminTokenReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/viewertoken project revokeClusterViewerTokenV2
//
//     Revokes the current viewer token
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) revokeClusterViewerToken() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.RevokeViewerTokenEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeAdminTokenReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/hetzner/sizes hetzner listHetznerSizesNoCredentialsV2
//
// Lists sizes from hetzner
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: HetznerSizeList
func (r Routing) listHetznerSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.HetznerSizeWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.settingsProvider)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/digitalocean/sizes digitalocean listDigitaloceanSizesNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.DigitaloceanSizeWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.settingsProvider)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/sizes openstack listOpenstackSizesNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackSizeWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/tenants openstack listOpenstackTenantsNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackTenantWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/networks openstack listOpenstackNetworksNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackNetworkWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/securitygroups openstack listOpenstackSecurityGroupsNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackSecurityGroupWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/subnets openstack listOpenstackSubnetsNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackSubnetsWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeOpenstackSubnetNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/availabilityzones openstack listOpenstackAvailabilityZonesNoCredentialsV2
//
// Lists availability zones from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackAvailabilityZone
func (r Routing) listOpenstackAvailabilityZonesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/azure/sizes azure listAzureSizesNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AzureSizeWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeAzureSizesNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/azure/availabilityzones azure listAzureAvailabilityZonesNoCredentialsV2
//
// Lists available VM availability zones in an Azure region
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureAvailabilityZonesList
func (r Routing) listAzureAvailabilityZonesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AzureAvailabilityZonesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeAzureAvailabilityZonesNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vsphere/networks vsphere listVSphereNetworksNoCredentialsV2
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VsphereNetworksWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVSphereNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vsphere/folders vsphere listVSphereFoldersNoCredentialsV2
//
// Lists folders from vsphere datacenter
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []VSphereFolder
func (r Routing) listVSphereFoldersNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VsphereFoldersWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVSphereNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/instancetypes alibaba listAlibabaInstanceTypesNoCredentialsV2
//
// Lists available Alibaba Instance Types
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AlibabaInstanceTypeList
func (r Routing) listAlibabaInstanceTypesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AlibabaInstanceTypesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeAlibabaNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/zones alibaba listAlibabaZonesNoCredentialsV2
//
// Lists available Alibaba Instance Types
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AlibabaZoneList
func (r Routing) listAlibabaZonesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AlibabaZonesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeAlibabaNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/packet/sizes packet listPacketSizesNoCredentialsV2
//
// Lists sizes from packet
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []PacketSizeList
func (r Routing) listPacketSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.PacketSizesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.settingsProvider)),
		provider.DecodePacketSizesNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy
//
//    Proxies the Kubernetes Dashboard. Requires a valid bearer token. The token can be obtained
//    using the /api/v1/projects/{project_id}/clusters/{cluster_id}/dashboard/login
//    endpoint.
//
//     Responses:
//       default: empty
func (r Routing) kubernetesDashboardProxy() http.Handler {
	return kubernetesdashboard.ProxyEndpoint(
		r.log,
		middleware.TokenExtractor(r.tokenExtractors),
		r.projectProvider,
		r.privilegedProjectProvider,
		r.userInfoGetter,
		r.settingsProvider,
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			// TODO: Instead of using an admin client to talk to the seed, we should provide a seed
			// client that allows access to the cluster namespace only
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		),
	)
}

// swagger:route GET /api/v2/providers/azure/securitygroups azure listAzureSecurityGroups
//
// Lists available VM security groups
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureSecurityGroupsList
func (r Routing) listAzureSecurityGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureSecurityGroupsEndpoint(r.presetsProvider, r.userInfoGetter)),
		provider.DecodeAzureSecurityGroupsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/resourcegroups azure listAzureResourceGroups
//
// Lists available VM resource groups
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureResourceGroupsList
func (r Routing) listAzureResourceGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureResourceGroupsEndpoint(r.presetsProvider, r.userInfoGetter)),
		provider.DecodeAzureResourceGroupsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/routetables azure listAzureRouteTables
//
// Lists available VM route tables
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureRouteTablesList
func (r Routing) listAzureRouteTables() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureRouteTablesEndpoint(r.presetsProvider, r.userInfoGetter)),
		provider.DecodeAzureRouteTablesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/vnets azure listAzureVnets
//
// Lists available VM virtual networks
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureVirtualNetworksList
func (r Routing) listAzureVnets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureVirtualNetworksEndpoint(r.presetsProvider, r.userInfoGetter)),
		provider.DecodeAzureVirtualNetworksReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/vsphere/datastores vsphere listVSphereDatastores
//
// Lists datastores from vsphere datacenter
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []VSphereDatastoreList
func (r Routing) listVSphereDatastores() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.VsphereDatastoreEndpoint(r.seedsGetter, r.presetsProvider, r.userInfoGetter)),
		provider.DecodeVSphereDatastoresReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/subnets azure listAzureSubnets
//
// Lists available VM subnets
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AzureSubnetsList
func (r Routing) listAzureSubnets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureSubnetsEndpoint(r.presetsProvider, r.userInfoGetter)),
		provider.DecodeAzureSubnetsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/presets preset listPresets
//
//     Lists presets
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: PresetList
//       401: empty
//       403: empty
func (r Routing) listPresets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.ListPresets(r.presetsProvider, r.userInfoGetter)),
		preset.DecodeListPresets,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/presets/{preset_name}/status preset updatePresetStatus
//
//     Updates the status of a preset. It can enable or disable it, so that it won't be listed by the list endpoints.
//
//
//     Consumes:
//	   - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) updatePresetStatus() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.UpdatePresetStatus(r.presetsProvider, r.userInfoGetter)),
		preset.DecodeUpdatePresetStatus,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/{provider_name}/presets preset listProviderPresets
//
//     Lists presets for the provider
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: PresetList
//       401: empty
//       403: empty
func (r Routing) listProviderPresets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.ListProviderPresets(r.presetsProvider, r.userInfoGetter)),
		preset.DecodeListProviderPresets,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/providers/{provider_name}/presets preset createPreset
//
//     Creates the preset
//
//     Consumes:
//	   - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Preset
//       401: empty
//       403: empty
func (r Routing) createPreset() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.CreatePreset(r.presetsProvider, r.userInfoGetter)),
		preset.DecodeCreatePreset,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/providers/{provider_name}/presets preset updatePreset
//
//	   Updates provider preset
//
//     Consumes:
//	   - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Preset
//       401: empty
//       403: empty
func (r Routing) updatePreset() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.UpdatePreset(r.presetsProvider, r.userInfoGetter)),
		preset.DecodeUpdatePreset,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/serviceaccounts mainserviceaccounts createMainServiceAccount
//
//     Creates the given service account
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: ServiceAccount
//       401: empty
//       403: empty
func (r Routing) createMainServiceAccount() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.CreateEndpoint(r.serviceAccountProvider, r.userInfoGetter)),
		serviceaccount.DecodeAddReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/serviceaccounts mainserviceaccounts listMainServiceAccounts
//
//     List main service accounts
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []ServiceAccount
//       401: empty
//       403: empty
func (r Routing) listMainServiceAccounts() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.ListEndpoint(r.serviceAccountProvider, r.userInfoGetter)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/V2/serviceaccounts/{serviceaccount_id} mainserviceaccounts updateMainServiceAccount
//
//     Updates main service account
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ServiceAccount
//       401: empty
//       403: empty
func (r Routing) updateMainServiceAccount() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.UpdateEndpoint(r.serviceAccountProvider, r.userInfoGetter)),
		serviceaccount.DecodeUpdateReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/serviceaccounts/{serviceaccount_id} mainserviceaccounts deleteMainServiceAccount
//
//     Deletes main service account
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteMainServiceAccount() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.DeleteEndpoint(r.serviceAccountProvider, r.privilegedServiceAccountProvider, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		serviceaccount.DecodeDeleteReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/serviceaccounts/{serviceaccount_id}/tokens tokens addTokenToMainServiceAccount
//
//     Generates a token for the given main service account
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: ServiceAccountToken
//       401: empty
//       403: empty
func (r Routing) addTokenToMainServiceAccount() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.CreateTokenEndpoint(r.serviceAccountProvider, r.privilegedServiceAccountTokenProvider, r.saTokenAuthenticator, r.saTokenGenerator, r.userInfoGetter)),
		serviceaccount.DecodeAddTokenReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/serviceaccounts/{serviceaccount_id}/tokens tokens listMainServiceAccountTokens
//
//     List tokens for the given main service account
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []PublicServiceAccountToken
//       401: empty
//       403: empty
func (r Routing) listMainServiceAccountTokens() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.ListTokenEndpoint(r.serviceAccountProvider, r.privilegedServiceAccountTokenProvider, r.saTokenAuthenticator, r.userInfoGetter)),
		serviceaccount.DecodeTokenReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/serviceaccounts/{serviceaccount_id}/tokens/{token_id} tokens updateMainServiceAccountToken
//
//     Updates and regenerates the token
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: ServiceAccountToken
//       401: empty
//       403: empty
func (r Routing) updateMainServiceAccountToken() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.UpdateTokenEndpoint(r.serviceAccountProvider, r.privilegedServiceAccountTokenProvider, r.saTokenAuthenticator, r.saTokenGenerator, r.userInfoGetter)),
		serviceaccount.DecodeUpdateTokenReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/serviceaccounts/{serviceaccount_id}/tokens/{token_id} tokens patchMainServiceAccountToken
//
//     Patches the token name
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: PublicServiceAccountToken
//       401: empty
//       403: empty
func (r Routing) patchMainServiceAccountToken() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.PatchTokenEndpoint(r.serviceAccountProvider, r.privilegedServiceAccountTokenProvider, r.saTokenAuthenticator, r.saTokenGenerator, r.userInfoGetter)),
		serviceaccount.DecodePatchTokenReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/serviceaccounts/{serviceaccount_id}/tokens/{token_id} tokens deleteMainServiceAccountToken
//
//     Deletes the token
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteMainServiceAccountToken() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(serviceaccount.DeleteTokenEndpoint(r.serviceAccountProvider, r.privilegedServiceAccountTokenProvider, r.userInfoGetter)),
		serviceaccount.DecodeDeleteTokenReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}
