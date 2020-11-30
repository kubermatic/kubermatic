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
	"github.com/prometheus/client_golang/prometheus"

	"k8c.io/kubermatic/v2/pkg/handler"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/addon"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/handler/v2/constraint"
	constrainttemplate "k8c.io/kubermatic/v2/pkg/handler/v2/constraint_template"
	externalcluster "k8c.io/kubermatic/v2/pkg/handler/v2/external_cluster"
	"k8c.io/kubermatic/v2/pkg/handler/v2/machine"
	"k8c.io/kubermatic/v2/pkg/handler/v2/provider"
)

// RegisterV2 declares all router paths for v2
func (r Routing) RegisterV2(mux *mux.Router, metrics common.ServerMetrics) {

	// Defines a set of HTTP endpoints for cluster that belong to a project.
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters").
		Handler(r.createCluster(metrics.InitNodeDeploymentFailures))

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
func (r Routing) createCluster(initNodeDeploymentFailures *prometheus.CounterVec) http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.CreateEndpoint(r.sshKeyProvider, r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, initNodeDeploymentFailures, r.eventRecorderProvider, r.presetsProvider, r.exposeStrategy, r.userInfoGetter, r.settingsProvider, r.updateManager)),
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
		)(provider.AWSSizeNoCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
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
