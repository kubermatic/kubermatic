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
	"k8c.io/kubermatic/v2/pkg/handler/v2/alertmanager"
	allowedregistry "k8c.io/kubermatic/v2/pkg/handler/v2/allowed_registry"
	applicationdefinition "k8c.io/kubermatic/v2/pkg/handler/v2/application_definition"
	applicationinstallation "k8c.io/kubermatic/v2/pkg/handler/v2/application_installation"
	"k8c.io/kubermatic/v2/pkg/handler/v2/backupcredentials"
	"k8c.io/kubermatic/v2/pkg/handler/v2/backupdestinations"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	clustertemplate "k8c.io/kubermatic/v2/pkg/handler/v2/cluster_template"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cniversion"
	"k8c.io/kubermatic/v2/pkg/handler/v2/constraint"
	constrainttemplate "k8c.io/kubermatic/v2/pkg/handler/v2/constraint_template"
	"k8c.io/kubermatic/v2/pkg/handler/v2/etcdbackupconfig"
	"k8c.io/kubermatic/v2/pkg/handler/v2/etcdrestore"
	externalcluster "k8c.io/kubermatic/v2/pkg/handler/v2/external_cluster"
	featuregates "k8c.io/kubermatic/v2/pkg/handler/v2/feature_gates"
	"k8c.io/kubermatic/v2/pkg/handler/v2/gatekeeperconfig"
	groupprojectbinding "k8c.io/kubermatic/v2/pkg/handler/v2/group-project-binding"
	ipampool "k8c.io/kubermatic/v2/pkg/handler/v2/ipampool"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/handler/v2/kubernetes-dashboard"
	"k8c.io/kubermatic/v2/pkg/handler/v2/machine"
	mlaadminsetting "k8c.io/kubermatic/v2/pkg/handler/v2/mla_admin_setting"
	"k8c.io/kubermatic/v2/pkg/handler/v2/networkdefaults"
	operatingsystemprofile "k8c.io/kubermatic/v2/pkg/handler/v2/operatingsystemprofile"
	"k8c.io/kubermatic/v2/pkg/handler/v2/preset"
	"k8c.io/kubermatic/v2/pkg/handler/v2/provider"
	resourcequota "k8c.io/kubermatic/v2/pkg/handler/v2/resource_quota"
	"k8c.io/kubermatic/v2/pkg/handler/v2/rulegroup"
	rulegroupadmin "k8c.io/kubermatic/v2/pkg/handler/v2/rulegroup_admin"
	"k8c.io/kubermatic/v2/pkg/handler/v2/seedsettings"
	"k8c.io/kubermatic/v2/pkg/handler/v2/user"
	"k8c.io/kubermatic/v2/pkg/handler/v2/version"
	"k8c.io/kubermatic/v2/pkg/handler/v2/webterminal"
)

// RegisterV2 declares all router paths for v2.
func (r Routing) RegisterV2(mux *mux.Router, oidcKubeConfEndpoint bool, oidcCfg common.OIDCConfiguration) {
	// Defines a set of HTTP endpoint for generating kubeconfig secret for a cluster that will contain OIDC tokens
	if oidcKubeConfEndpoint {
		mux.Methods(http.MethodGet).
			Path("/kubeconfig/secret").
			Handler(r.createOIDCKubeconfigSecret(oidcCfg))
	}

	// Defines a set of HTTP endpoint for interacting with
	// various cloud providers
	mux.Methods(http.MethodGet).
		Path("/providers/gke/images").
		Handler(r.listGKEImages())

	mux.Methods(http.MethodGet).
		Path("/providers/gke/zones").
		Handler(r.listGKEZones())

	mux.Methods(http.MethodGet).
		Path("/providers/gke/vmsizes").
		Handler(r.listGKEVMSizes())

	mux.Methods(http.MethodGet).
		Path("/providers/gke/disktypes").
		Handler(r.listGKEDiskTypes())

	mux.Methods(http.MethodGet).
		Path("/providers/gke/versions").
		Handler(r.listGKEVersions())

	mux.Methods(http.MethodGet).
		Path("/providers/gke/validatecredentials").
		Handler(r.validateGKECredentials())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/validatecredentials").
		Handler(r.validateEKSCredentials())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/vpcs").
		Handler(r.listEKSVPCS())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/subnets").
		Handler(r.listEKSSubnets())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/securitygroups").
		Handler(r.listEKSSecurityGroups())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/regions").
		Handler(r.listEKSRegions())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/clusterroles").
		Handler(r.listEKSClusterRoles())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/versions").
		Handler(r.listEKSVersions())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/amitypes").
		Handler(r.listEKSAMITypes())

	mux.Methods(http.MethodGet).
		Path("/providers/eks/capacitytypes").
		Handler(r.listEKSCapacityTypes())

	mux.Methods(http.MethodGet).
		Path("/providers/aks/validatecredentials").
		Handler(r.validateAKSCredentials())

	mux.Methods(http.MethodGet).
		Path("/providers/aks/vmsizes").
		Handler(r.listAKSVMSizes())

	mux.Methods(http.MethodGet).
		Path("/providers/aks/resourcegroups").
		Handler(r.listAKSResourceGroups())

	mux.Methods(http.MethodGet).
		Path("/providers/aks/locations").
		Handler(r.listAKSLocations())

	mux.Methods(http.MethodGet).
		Path("/providers/aks/modes").
		Handler(r.listAKSNodePoolModes())

	mux.Methods(http.MethodGet).
		Path("/providers/aks/versions").
		Handler(r.listAKSVersions())

	mux.Methods(http.MethodGet).
		Path("/featuregates").
		Handler(r.getFeatureGates())

	// Defines a set of HTTP endpoints for interacting with KubeVirt clusters
	mux.Methods(http.MethodGet).
		Path("/providers/kubevirt/vmflavors").
		Handler(r.listKubeVirtVMIPresets())

	mux.Methods(http.MethodGet).
		Path("/providers/kubevirt/instancetypes").
		Handler(r.listKubeVirtInstancetypes())

	mux.Methods(http.MethodGet).
		Path("/providers/kubevirt/preferences").
		Handler(r.listKubeVirtPreferences())

	mux.Methods(http.MethodGet).
		Path("/providers/kubevirt/storageclasses").
		Handler(r.listKubevirtStorageClasses())

	// Defines a set of HTTP endpoints for cluster that belong to a project.
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/gke/clusters").
		Handler(r.listGKEClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/eks/clusters").
		Handler(r.listEKSClusters())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/aks/clusters").
		Handler(r.listAKSClusters())

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

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/externalccmmigration").
		Handler(r.migrateClusterToExternalCCM())

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
		Path("/projects/{project_id}/clusters/{cluster_id}/oidc").
		Handler(r.getClusterOidc())

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

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/restart").
		Handler(r.restartMachineDeployment())

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

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}").
		Handler(r.patchExternalCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/metrics").
		Handler(r.getExternalClusterMetrics())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}").
		Handler(r.updateExternalCluster())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/upgrades").
		Handler(r.getExternalClusterUpgrades())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}").
		Handler(r.getExternalClusterMachineDeployment())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments").
		Handler(r.listExternalClusterMachineDeployments())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments").
		Handler(r.createExternalClusterMachineDeployment())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}").
		Handler(r.deleteExternalClusterMachineDeployment())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes").
		Handler(r.listExternalClusterMachineDeploymentNodes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodes").
		Handler(r.listExternalClusterNodes())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}").
		Handler(r.patchExternalClusterMachineDeployments())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/upgrades").
		Handler(r.getExternalClusterMachineDeploymentUpgrades())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/metrics").
		Handler(r.listExternalClusterMachineDeploymentMetrics())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events").
		Handler(r.listExternalClusterMachineDeploymentEvents())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodes/{node_id}").
		Handler(r.getExternalClusterNode())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodesmetrics").
		Handler(r.listExternalClusterNodesMetrics())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/events").
		Handler(r.listExternalClusterEvents())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/kubeconfig").
		Handler(r.getExternalClusterKubeconfig())

	// Defines a set of HTTP endpoint for ApplicationInstallations that belong to a cluster
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/applicationinstallations").
		Handler(r.listApplicationInstallations())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/applicationinstallations").
		Handler(r.createApplicationInstallation())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/applicationinstallations/{namespace}/{appinstall_name}").
		Handler(r.deleteApplicationInstallation())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/applicationinstallations/{namespace}/{appinstall_name}").
		Handler(r.getApplicationInstallation())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/applicationinstallations/{namespace}/{appinstall_name}").
		Handler(r.updateApplicationInstallation())

	// Defines a set of HTTP endpoint for ApplicationDefinitions which are available in the KKP installation
	mux.Methods(http.MethodGet).
		Path("/applicationdefinitions").
		Handler(r.listApplicationDefinitions())

	mux.Methods(http.MethodGet).
		Path("/applicationdefinitions/{appdef_name}").
		Handler(r.getApplicationDefinition())

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

	// Define a set of endpoints for default constraints
	mux.Methods(http.MethodPost).
		Path("/constraints").
		Handler(r.createDefaultConstraint())

	mux.Methods(http.MethodGet).
		Path("/constraints").
		Handler(r.listDefaultConstraint())

	mux.Methods(http.MethodGet).
		Path("/constraints/{constraint_name}").
		Handler(r.getDefaultConstraint())

	mux.Methods(http.MethodDelete).
		Path("/constraints/{constraint_name}").
		Handler(r.deleteDefaultConstraint())

	mux.Methods(http.MethodPatch).
		Path("/constraints/{constraint_name}").
		Handler(r.patchDefaultConstraint())

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

	// Defines a set of HTTP endpoints for managing alertmanager
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/alertmanager/config").
		Handler(r.getAlertmanager())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/alertmanager/config").
		Handler(r.updateAlertmanager())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/alertmanager/config").
		Handler(r.resetAlertmanager())

	// Defines a set of HTTP endpoints for various cloud providers
	// These endpoints are required to use project-scoped presets as credentials

	// GKE endpoints
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/gke/images").
		Handler(r.listProjectGKEImages())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/gke/zones").
		Handler(r.listProjectGKEZones())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/gke/vmsizes").
		Handler(r.listProjectGKEVMSizes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/gke/disktypes").
		Handler(r.listProjectGKEDiskTypes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/gke/versions").
		Handler(r.listProjectGKEVersions())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/gke/validatecredentials").
		Handler(r.validateProjectGKECredentials())

		// TODO: implement provider-specific API endpoints and uncomment providers you implement.

		/*
			// EKS endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/validatecredentials").
				Handler(r.validateProjectEKSCredentials())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/vpcs").
				Handler(r.listProjectEKSVPCS())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/subnets").
				Handler(r.listProjectEKSSubnets())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/securitygroups").
				Handler(r.listProjectEKSSecurityGroups())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/regions").
				Handler(r.listProjectEKSRegions())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/clusterroles").
				Handler(r.listProjectEKSClusterRoles())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/versions").
				Handler(r.listProjectEKSVersions())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/amitypes").
				Handler(r.listProjectEKSAMITypes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/eks/capacitytypes").
				Handler(r.listProjectEKSCapacityTypes())

			// AKS credentials
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aks/validatecredentials").
				Handler(r.validateProjectAKSCredentials())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aks/vmsizes").
				Handler(r.listProjectAKSVMSizes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aks/resourcegroups").
				Handler(r.listProjectAKSResourceGroups())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aks/locations").
				Handler(r.listProjectAKSLocations())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aks/modes").
				Handler(r.listProjectAKSNodePoolModes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aks/versions").
				Handler(r.listProjectAKSVersions())

			// Kubevirt endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/kubevirt/vmflavors").
				Handler(r.listProjectKubeVirtVMIPresets())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/kubevirt/instancetypes").
				Handler(r.listProjectKubeVirtInstancetypes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/kubevirt/preferences").
				Handler(r.listProjectKubeVirtPreferences())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/kubevirt/storageclasses").
				Handler(r.listProjectKubevirtStorageClasses())

			// Azure endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/azure/sizes").
				Handler(r.listProjectAzureSizes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/azure/availabilityzones").
				Handler(r.listProjectAzureSKUAvailabilityZones())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/azure/securitygroups").
				Handler(r.listProjectAzureSecurityGroups())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/azure/resourcegroups").
				Handler(r.listProjectAzureResourceGroups())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/azure/routetables").
				Handler(r.listProjectAzureRouteTables())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/azure/subnets").
				Handler(r.listProjectAzureSubnets())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/azure/vnets").
				Handler(r.listProjectAzureVnets())

			// vSphere endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/vsphere/networks").
				Handler(r.listProjectVSphereNetworks())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/vsphere/folders").
				Handler(r.listProjectVSphereFolders())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/vsphere/datastores").
				Handler(r.listProjectVSphereDatastores())

			// Nutanix endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/nutanix/{dc}/clusters").
				Handler(r.listProjectNutanixClusters())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/nutanix/{dc}/projects").
				Handler(r.listProjectNutanixProjects())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/nutanix/{dc}/subnets").
				Handler(r.listProjectNutanixSubnets())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/nutanix/{dc}/categories").
				Handler(r.listProjectNutanixCategories())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/nutanix/{dc}/categories/{category}/values").
				Handler(r.listProjectNutanixCategoryValues())

			// VMware Cloud Director endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/vmwareclouddirector/{dc}/networks").
				Handler(r.listProjectVMwareCloudDirectorNetworks())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/vmwareclouddirector/{dc}/storageprofiles").
				Handler(r.listProjectVMwareCloudDirectorStorageProfiles())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/vmwareclouddirector/{dc}/catalogs").
				Handler(r.listProjectVMwareCloudDirectorCatalogs())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/vmwareclouddirector/{dc}/templates/{catalog_name}").
				Handler(r.listProjectVMwareCloudDirectorTemplates())

			// AWS endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aws/sizes").
				Handler(r.listProjectAWSSizes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aws/{dc}/subnets").
				Handler(r.listProjectAWSSubnets())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aws/{dc}/vpcs").
				Handler(r.listProjectAWSVPCS())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/aws/{dc}/securitygroups").
				Handler(r.listProjectAWSSecurityGroups())

			// GCP endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/gcp/disktypes").
				Handler(r.listProjectGCPDiskTypes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/gcp/sizes").
				Handler(r.listProjectGCPSizes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/gcp/{dc}/zones").
				Handler(r.listProjectGCPZones())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/gcp/networks").
				Handler(r.listProjectGCPNetworks())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/gcp/{dc}/subnetworks").
				Handler(r.listProjectGCPSubnetworks())

			// Digitalocean endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/digitalocean/sizes").
				Handler(r.listProjectDigitaloceanSizes())

			// Openstack endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/openstack/sizes").
				Handler(r.listProjectOpenstackSizes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/openstack/tenants").
				Handler(r.listProjectOpenstackTenants())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/openstack/networks").
				Handler(r.listProjectOpenstackNetworks())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/openstack/securitygroups").
				Handler(r.listProjectOpenstackSecurityGroups())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/openstack/subnets").
				Handler(r.listProjectOpenstackSubnets())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/openstack/availabilityzones").
				Handler(r.listProjectOpenstackAvailabilityZones())

			// Equinix Metal (Packet) endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/packet/sizes").
				Handler(r.listProjectPacketSizes())

			// Hetzner endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/hetzner/sizes").
				Handler(r.listProjectHetznerSizes())

			// Alibaba endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/alibaba/instancetypes").
				Handler(r.listProjectAlibabaInstanceTypes())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/alibaba/zones").
				Handler(r.listProjectAlibabaZones())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/alibaba/vswitches").
				Handler(r.listProjectAlibabaVSwitches())

			// Anexia endpoints
			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/anexia/vlans").
				Handler(r.listProjectAnexiaVlans())

			mux.Methods(http.MethodGet).
				Path("/projects/{project_id}/providers/anexia/templates").
				Handler(r.listProjectAnexiaTemplates())

		*/

	// Defines a set of HTTP endpoints for various cloud providers
	// Note that these endpoints don't require credentials as opposed to the ones defined under
	// /providers/* and /projects/{project_id}/providers/*.
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
		Path("/providers/openstack/subnetpools").
		Handler(r.listOpenstackSubnetPools())

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
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/vswitches").
		Handler(r.listAlibabaVSwitchesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/packet/sizes").
		Handler(r.listPacketSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/anexia/vlans").
		Handler(r.listAnexiaVlansNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/anexia/templates").
		Handler(r.listAnexiaTemplatesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/vmflavors").
		Handler(r.listKubeVirtVMIPresetsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/instancetypes").
		Handler(r.listKubeVirtInstancetypesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/preferences").
		Handler(r.listKubeVirtPreferencesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/storageclasses").
		Handler(r.listKubevirtStorageClassesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/subnets").
		Handler(r.listNutanixSubnetsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/categories").
		Handler(r.listNutanixCategoriesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/categories/{category}/values").
		Handler(r.listNutanixCategoryValuesNoCredentials())

	// Endpoints for VMware Cloud Director
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/networks").
		Handler(r.listVMwareCloudDirectorNetworksNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/storageprofiles").
		Handler(r.listVMwareCloudDirectorStorageProfilesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/catalogs").
		Handler(r.listVMwareCloudDirectorCatalogsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/templates/{catalog_name}").
		Handler(r.listVMwareCloudDirectorTemplatesNoCredentials())

	kubernetesdashboard.
		NewLoginHandler(oidcCfg, r.oidcIssuerVerifier, r.settingsProvider).
		Middlewares(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
		).
		Options(r.defaultServerOptions()...).
		Install(mux)

	kubernetesdashboard.
		NewProxyHandler(r.log, r.settingsProvider, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter).
		RequestFuncs(
			middleware.TokenExtractor(r.tokenExtractors),
			middleware.SetSeedsGetter(r.seedsGetter)).
		Middlewares(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		).
		Options(r.defaultServerOptions()...).
		Install(mux)

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

	mux.Methods(http.MethodGet).
		Path("/providers/nutanix/{dc}/clusters").
		Handler(r.listNutanixClusters())

	mux.Methods(http.MethodGet).
		Path("/providers/nutanix/{dc}/projects").
		Handler(r.listNutanixProjects())

	mux.Methods(http.MethodGet).
		Path("/providers/nutanix/{dc}/subnets").
		Handler(r.listNutanixSubnets())

	mux.Methods(http.MethodGet).
		Path("/providers/nutanix/{dc}/categories").
		Handler(r.listNutanixCategories())

	mux.Methods(http.MethodGet).
		Path("/providers/nutanix/{dc}/categories/{category}/values").
		Handler(r.listNutanixCategoryValues())

	// Endpoints for VMware Cloud Director
	mux.Methods(http.MethodGet).
		Path("/providers/vmwareclouddirector/{dc}/networks").
		Handler(r.listVMwareCloudDirectorNetworks())

	mux.Methods(http.MethodGet).
		Path("/providers/vmwareclouddirector/{dc}/storageprofiles").
		Handler(r.listVMwareCloudDirectorStorageProfiles())

	mux.Methods(http.MethodGet).
		Path("/providers/vmwareclouddirector/{dc}/catalogs").
		Handler(r.listVMwareCloudDirectorCatalogs())

	mux.Methods(http.MethodGet).
		Path("/providers/vmwareclouddirector/{dc}/templates/{catalog_name}").
		Handler(r.listVMwareCloudDirectorTemplates())

	// Define a set of endpoints for preset management
	mux.Methods(http.MethodGet).
		Path("/presets").
		Handler(r.listPresets())

	mux.Methods(http.MethodDelete).
		Path("/presets/{preset_name}").
		Handler(r.deletePreset())

	mux.Methods(http.MethodGet).
		Path("/presets/{preset_name}/stats").
		Handler(r.getPresetStats())

	mux.Methods(http.MethodPut).
		Path("/presets/{preset_name}/status").
		Handler(r.updatePresetStatus())

	mux.Methods(http.MethodDelete).
		Path("/presets/{preset_name}/provider/{provider_name}").
		Handler(r.deletePresetProvider())

	mux.Methods(http.MethodGet).
		Path("/providers/{provider_name}/presets").
		Handler(r.listProviderPresets())

	mux.Methods(http.MethodPost).
		Path("/providers/{provider_name}/presets").
		Handler(r.createPreset())

	mux.Methods(http.MethodPut).
		Path("/providers/{provider_name}/presets").
		Handler(r.updatePreset())

	mux.Methods(http.MethodDelete).
		Path("/providers/{provider_name}/presets/{preset_name}").
		Handler(r.deleteProviderPreset())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/presets").
		Handler(r.listProjectPresets())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/providers/{provider_name}/presets").
		Handler(r.listProjectProviderPresets())

	mux.Methods(http.MethodGet).
		Path("/seeds/{seed_name}/settings").
		Handler(r.getSeedSettings())

	// Define an endpoint to retrieve the Kubernetes versions supported by the given provider
	mux.Methods(http.MethodGet).
		Path("/providers/{provider_name}/versions").
		Handler(r.listVersionsByProvider())

	// Define a set of endpoints for cluster templates management
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clustertemplates").
		Handler(r.createClusterTemplate())
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clustertemplates/import").
		Handler(r.importClusterTemplate())
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clustertemplates").
		Handler(r.listClusterTemplates())
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clustertemplates/{template_id}").
		Handler(r.getClusterTemplate())
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clustertemplates/{template_id}/export").
		Handler(r.exportClusterTemplate())
	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clustertemplates/{template_id}").
		Handler(r.deleteClusterTemplate())
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clustertemplates/{template_id}/instances").
		Handler(r.createClusterTemplateInstance())

	// Defines a set of HTTP endpoints for managing rule groups
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/rulegroups/{rulegroup_id}").
		Handler(r.getRuleGroup())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/rulegroups").
		Handler(r.listRuleGroups())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/rulegroups").
		Handler(r.createRuleGroup())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/rulegroups/{rulegroup_id}").
		Handler(r.updateRuleGroup())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/rulegroups/{rulegroup_id}").
		Handler(r.deleteRuleGroup())

	// Defines a set of HTTP endpoints for managing allowed registries
	mux.Methods(http.MethodPost).
		Path("/allowedregistries").
		Handler(r.createAllowedRegistry())

	mux.Methods(http.MethodGet).
		Path("/allowedregistries").
		Handler(r.listAllowedRegistries())

	mux.Methods(http.MethodGet).
		Path("/allowedregistries/{allowed_registry}").
		Handler(r.getAllowedRegistry())

	mux.Methods(http.MethodDelete).
		Path("/allowedregistries/{allowed_registry}").
		Handler(r.deleteAllowedRegistry())

	mux.Methods(http.MethodPatch).
		Path("/allowedregistries/{allowed_registry}").
		Handler(r.patchAllowedRegistry())

	// Defines a set of HTTP endpoints for managing etcd backup configs
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs").
		Handler(r.createEtcdBackupConfig())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_id}").
		Handler(r.getEtcdBackupConfig())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs").
		Handler(r.listEtcdBackupConfig())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_id}").
		Handler(r.deleteEtcdBackupConfig())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_id}").
		Handler(r.patchEtcdBackupConfig())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/etcdbackupconfigs").
		Handler(r.listProjectEtcdBackupConfig())

	// Defines a set of HTTP endpoints for managing etcd backup restores
	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdrestores").
		Handler(r.createEtcdRestore())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name}").
		Handler(r.getEtcdRestore())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdrestores").
		Handler(r.listEtcdRestore())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name}").
		Handler(r.deleteEtcdRestore())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/etcdrestores").
		Handler(r.listProjectEtcdRestore())

	// Defines a set of HTTP endpoints for managing etcd backup restores
	mux.Methods(http.MethodPut).
		Path("/seeds/{seed_name}/backupcredentials").
		Handler(r.createOrUpdateBackupCredentials())

	// Defines a set of HTTP endpoints for managing MLA admin setting
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting").
		Handler(r.getMLAAdminSetting())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting").
		Handler(r.createMLAAdminSetting())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting").
		Handler(r.updateMLAAdminSetting())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting").
		Handler(r.deleteMLAAdminSetting())

	// Defines a set of HTTP endpoints for managing users
	mux.Methods(http.MethodGet).
		Path("/users").
		Handler(r.listUser())

	// Defines a set of HTTP endpoints for managing rule groups for admins
	mux.Methods(http.MethodGet).
		Path("/seeds/{seed_name}/rulegroups/{rulegroup_id}").
		Handler(r.getAdminRuleGroup())

	mux.Methods(http.MethodGet).
		Path("/seeds/{seed_name}/rulegroups").
		Handler(r.listAdminRuleGroups())

	mux.Methods(http.MethodPost).
		Path("/seeds/{seed_name}/rulegroups").
		Handler(r.createAdminRuleGroup())

	mux.Methods(http.MethodPut).
		Path("/seeds/{seed_name}/rulegroups/{rulegroup_id}").
		Handler(r.updateAdminRuleGroup())

	mux.Methods(http.MethodDelete).
		Path("/seeds/{seed_name}/rulegroups/{rulegroup_id}").
		Handler(r.deleteAdminRuleGroup())

	// Defines a set of HTTP endpoints for various cloud providers
	// Note that these endpoints don't require credentials as opposed to the ones defined under /providers/*
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/aks/versions").
		Handler(r.listAKSNodeVersionsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/aks/vmsizes").
		Handler(r.listAKSVMSizesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/images").
		Handler(r.listGKEClusterImages())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/zones").
		Handler(r.listGKEClusterZones())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes").
		Handler(r.listGKEClusterSizes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/disktypes").
		Handler(r.listGKEClusterDiskTypes())

	// Define an endpoint for getting seed backup destination names for a cluster
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/backupdestinations").
		Handler(r.getBackupDestinationNames())

	// Defines endpoints for CNI versions
	mux.Methods(http.MethodGet).
		Path("/cni/{cni_plugin_type}/versions").
		Handler(r.listVersionsByCNIPlugin())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/cniversions").
		Handler(r.listCNIPluginVersionsForCluster())

	// Defines a set of HTTP endpoints for various cloud providers
	// Note that these endpoints don't require credentials as opposed to the ones defined under /providers/*
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/instancetypes").
		Handler(r.listEKSInstanceTypesNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/subnets").
		Handler(r.listEKSSubnetsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/vpcs").
		Handler(r.listEKSVPCsNoCredentials())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/noderoles").
		Handler(r.listEKSNodeRolesNoCredentials())

	// Defines an endpoint to retrieve the cluster networking defaults for the given provider and CNI.
	mux.Methods(http.MethodGet).
		Path("/providers/{provider_name}/dc/{dc}/networkdefaults").
		Handler(r.getNetworkDefaults())

	// Defines endpoints to interact with resource quotas
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/quota").
		Handler(r.getProjectQuota())

	mux.Methods(http.MethodGet).
		Path("/quotas/{quota_name}").
		Handler(r.getResourceQuota())

	mux.Methods(http.MethodGet).
		Path("/quotas").
		Handler(r.listResourceQuotas())

	mux.Methods(http.MethodPost).
		Path("/quotas").
		Handler(r.createResourceQuota())

	mux.Methods(http.MethodPut).
		Path("/quotas/{quota_name}").
		Handler(r.putResourceQuota())

	mux.Methods(http.MethodDelete).
		Path("/quotas/{quota_name}").
		Handler(r.deleteResourceQuota())

	// Defines endpoints to interact with group project bindings
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/groupbindings").
		Handler(r.listGroupProjectBindings())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/groupbindings/{binding_name}").
		Handler(r.getGroupProjectBinding())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/groupbindings").
		Handler(r.createGroupProjectBinding())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/groupbindings/{binding_name}").
		Handler(r.deleteGroupProjectBinding())

	mux.Methods(http.MethodPatch).
		Path("/projects/{project_id}/groupbindings/{binding_name}").
		Handler(r.patchGroupProjectBinding())

	// Defines endpoints to manage IPAM pools
	mux.Methods(http.MethodGet).
		Path("/seeds/{seed_name}/ipampools").
		Handler(r.listIPAMPools())

	mux.Methods(http.MethodGet).
		Path("/seeds/{seed_name}/ipampools/{ipampool_name}").
		Handler(r.getIPAMPool())

	mux.Methods(http.MethodPost).
		Path("/seeds/{seed_name}/ipampools").
		Handler(r.createIPAMPool())

	mux.Methods(http.MethodPatch).
		Path("/seeds/{seed_name}/ipampools/{ipampool_name}").
		Handler(r.patchIPAMPool())

	mux.Methods(http.MethodDelete).
		Path("/seeds/{seed_name}/ipampools/{ipampool_name}").
		Handler(r.deleteIPAMPool())

	// Define endpoints to manage operating system profiles.
	mux.Methods(http.MethodGet).
		Path("/seeds/{seed_name}/operatingsystemprofiles").
		Handler(r.listOperatingSystemProfiles())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/operatingsystemprofiles").
		Handler(r.listOperatingSystemProfilesForCluster())

	// Define endpoints to manage cluster service accounts
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/serviceaccount").
		Handler(r.listClusterServiceAccount())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/clusters/{cluster_id}/serviceaccount").
		Handler(r.createClusterServiceAccount())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/clusters/{cluster_id}/serviceaccount/{namespace}/{service_account_id}/kubeconfig").
		Handler(r.getClusterServiceAccountKubeconfig())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/clusters/{cluster_id}/serviceaccount/{namespace}/{service_account_id}").
		Handler(r.deleteClusterServiceAccount())
}

// swagger:route POST /api/v2/projects/{project_id}/clusters project createClusterV2
//
//	Creates a cluster for the given project.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: Cluster
//	  401: empty
//	  403: empty
func (r Routing) createCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.CreateEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter,
			r.presetProvider, r.exposeStrategy, r.userInfoGetter, r.settingsProvider, r.caBundle, r.kubermaticConfigGetter, r.features)),
		cluster.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters project listClustersV2
//
//	Lists clusters for the specified project. If query parameter `show_dm_count` is set to `true` then the endpoint will also return the number of machine deployments of each cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ClusterList
//	  401: empty
//	  403: empty
func (r Routing) listClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(cluster.ListEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.clusterProviderGetter, r.userInfoGetter, r.kubermaticConfigGetter)),
		cluster.DecodeListClustersReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id} project getClusterV2
//
//	Gets the cluster with the given name
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Cluster
//	  401: empty
//	  403: empty
func (r Routing) getCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.kubermaticConfigGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// Delete the cluster
// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id} project deleteClusterV2
//
//	Deletes the specified cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
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
//	Patches the given cluster using JSON Merge Patch method (https://tools.ietf.org/html/rfc7396).
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Cluster
//	  401: empty
//	  403: empty
func (r Routing) patchCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.PatchEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.caBundle, r.kubermaticConfigGetter, r.features)),
		cluster.DecodePatchReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// getClusterEvents returns events related to the cluster.
// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/events project getClusterEventsV2
//
//	Gets the events related to the specified cluster.
//
//	Produces:
//	- application/yaml
//
//	Responses:
//	  default: errorResponse
//	  200: []Event
//	  401: empty
//	  403: empty
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
//	Returns the cluster's component health status
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ClusterHealth
//	  401: empty
//	  403: empty
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
//	Gets the kubeconfig for the specified cluster.
//
//	Produces:
//	- application/octet-stream
//
//	Responses:
//	  default: errorResponse
//	  200: Kubeconfig
//	  401: empty
//	  403: empty
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
//	Gets the kubeconfig for the specified cluster with oidc authentication.
//
//	Produces:
//	- application/octet-stream
//
//	Responses:
//	  default: errorResponse
//	  200: Kubeconfig
//	  401: empty
//	  403: empty
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

// getClusterOidc returns the OIDC spec for the user cluster.
// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/oidc project getClusterOidc
//
//	Gets the OIDC params for the specified cluster with OIDC authentication.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: OIDCSpec
//	  401: empty
//	  403: empty
func (r Routing) getClusterOidc() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetClusterOidcEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/metrics project getClusterMetricsV2
//
//	Gets cluster metrics
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: ClusterMetrics
//	   401: empty
//	   403: empty
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
//	Lists all namespaces in the cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Namespace
//	  401: empty
//	  403: empty
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
//	Gets possible cluster upgrades
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: []MasterVersion
//	   401: empty
//	   403: empty
func (r Routing) getClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetUpgradesEndpoint(r.kubermaticConfigGetter, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/nodes/upgrades project upgradeClusterNodeDeploymentsV2
//
//	Upgrades node deployments in a cluster
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
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
//	Assigns an existing ssh key to the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: SSHKey
//	  401: empty
//	  403: empty
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
//	Unassignes an ssh key from the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
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
//	Lists ssh keys that are assigned to the cluster
//	The returned collection is sorted by creation timestamp.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []SSHKey
//	  401: empty
//	  403: empty
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
//	Creates an external cluster for the given project.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: ExternalCluster
//	  401: empty
//	  403: empty
func (r Routing) createExternalCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider, r.presetProvider)),
		externalcluster.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// Delete the external cluster
// swagger:route DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id} project deleteExternalCluster
//
//	Deletes the specified external cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
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
//	Lists external clusters for the specified project.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ExternalCluster
//	  401: empty
//	  403: empty
func (r Routing) listExternalClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id} project getExternalCluster
//
//	Gets an external cluster for the given project.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ExternalCluster
//	  401: empty
//	  403: empty
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

// swagger:route PATCH /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id} project patchExternalCluster
//
//	Patches the given cluster using JSON Merge Patch method (https://tools.ietf.org/html/rfc7396).
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ExternalCluster
//	  401: empty
//	  403: empty
func (r Routing) patchExternalCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.PatchEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodePatchReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id} project updateExternalCluster
//
//	Updates an external cluster for the given project.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ExternalCluster
//	  401: empty
//	  403: empty
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

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/upgrades project getExternalClusterUpgrades
//
//	Gets an external cluster upgrades.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []MasterVersion
//	  401: empty
//	  403: empty
func (r Routing) getExternalClusterUpgrades() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GetUpgradesEndpoint(r.kubermaticConfigGetter, r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments project listExternalClusterMachineDeployments
//
//	Gets an external cluster machine deployments.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ExternalClusterMachineDeployment
//	  401: empty
//	  403: empty
func (r Routing) listExternalClusterMachineDeployments() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListMachineDeploymentEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeListMachineDeploymentReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments project createExternalClusterMachineDeployment
//
//	Create an external cluster machine deployments.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ExternalClusterMachineDeployment
//	  401: empty
//	  403: empty
func (r Routing) createExternalClusterMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.CreateMachineDeploymentEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeCreateMachineDeploymentReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id} project deleteExternalClusterMachineDeployment
//
//	Delete an external cluster machine deployment.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteExternalClusterMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.DeleteMachineDeploymentEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeGetMachineDeploymentReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/nodes project listExternalClusterNodes
//
//	Gets an external cluster nodes.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ExternalClusterNode
//	  401: empty
//	  403: empty
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
//	Gets an external cluster node.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ExternalClusterNode
//	  401: empty
//	  403: empty
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
//	Gets cluster metrics
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ClusterMetrics
//	  401: empty
//	  403: empty
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
//	Gets an external cluster nodes metrics.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []NodeMetric
//	  401: empty
//	  403: empty
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
//	Gets an external cluster events.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Event
//	  401: empty
//	  403: empty
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
//	List constraint templates.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ConstraintTemplate
//	  401: empty
//	  403: empty
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
//	Get constraint templates specified by name
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ConstraintTemplate
//	  401: empty
//	  403: empty
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
//	Create constraint template
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ConstraintTemplate
//	  401: empty
//	  403: empty
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
//	Patch a specified constraint template
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ConstraintTemplate
//	  401: empty
//	  403: empty
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
//	Deletes the specified cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
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
//	Lists constraints for the specified cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Constraint
//	  401: empty
//	  403: empty
func (r Routing) listConstraints() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Constraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
		)(constraint.ListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		constraint.DecodeListConstraintsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name} project getConstraint
//
//	Gets an specified constraint for the given cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Constraint
//	  401: empty
//	  403: empty
func (r Routing) getConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Constraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
		)(constraint.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		constraint.DecodeConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name} project deleteConstraint
//
//	Deletes a specified constraint for the given cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Constraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
			middleware.PrivilegedConstraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
		)(constraint.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		constraint.DecodeConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints project createConstraint
//
//	Creates a given constraint for the specified cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Constraint
//	  401: empty
//	  403: empty
func (r Routing) createConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Constraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
			middleware.PrivilegedConstraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
		)(constraint.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.constraintTemplateProvider)),
		constraint.DecodeCreateConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/constraints constraint createDefaultConstraint
//
//	Creates default constraint
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Constraint
//	  401: empty
//	  403: empty
func (r Routing) createDefaultConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constraint.CreateDefaultEndpoint(r.userInfoGetter, r.defaultConstraintProvider, r.constraintTemplateProvider)),
		constraint.DecodeCreateDefaultConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/constraints constraint listDefaultConstraint
//
//	List default constraint.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Constraint
//	  401: empty
//	  403: empty
func (r Routing) listDefaultConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constraint.ListDefaultEndpoint(r.defaultConstraintProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/constraints/{constraint_name} constraint getDefaultConstraint
//
//	Gets an specified default constraint
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Constraint
//	  401: empty
//	  403: empty
func (r Routing) getDefaultConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constraint.GetDefaultEndpoint(r.defaultConstraintProvider)),
		constraint.DecodeDefaultConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/constraints/{constraint_name} constraints deleteDefaultConstraint
//
//	Deletes a specified default constraint.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteDefaultConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constraint.DeleteDefaultEndpoint(r.userInfoGetter, r.defaultConstraintProvider)),
		constraint.DecodeDefaultConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/constraints/{constraint_name} constraint patchDefaultConstraint
//
//	Patch a specified default constraint
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Constraint
//	  401: empty
//	  403: empty
func (r Routing) patchDefaultConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(constraint.PatchDefaultEndpoint(r.userInfoGetter, r.defaultConstraintProvider, r.constraintTemplateProvider)),
		constraint.DecodePatchDefaultConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints/{constraint_name} project patchConstraint
//
//	Patches a given constraint for the specified cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Constraint
//	  401: empty
//	  403: empty
func (r Routing) patchConstraint() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Constraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
			middleware.PrivilegedConstraints(r.clusterProviderGetter, r.constraintProviderGetter, r.seedsGetter),
		)(constraint.PatchEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.constraintTemplateProvider)),
		constraint.DecodePatchConstraintReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/gatekeeper/config project getGatekeeperConfig
//
//	Gets the gatekeeper sync config for the specified cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GatekeeperConfig
//	  401: empty
//	  403: empty
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
//	Deletes the gatekeeper sync config for the specified cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
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
//	Creates a gatekeeper config for the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: GatekeeperConfig
//	  401: empty
//	  403: empty
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
//	Patches the gatekeeper config for the specified cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GatekeeperConfig
//	  401: empty
//	  403: empty
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
//	Creates a machine deployment that will belong to the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: NodeDeployment
//	  401: empty
//	  403: empty
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
//	Deletes the given node that belongs to the machine deployment.
//
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
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
//	Lists machine deployments that belong to the given cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []NodeDeployment
//	  401: empty
//	  403: empty
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
//	Gets a machine deployment that is assigned to the given cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: NodeDeployment
//	  401: empty
//	  403: empty
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
//	Lists nodes that belong to the given machine deployment.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Node
//	  401: empty
//	  403: empty
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
//	This endpoint is used for kubeadm cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Node
//	  401: empty
//	  403: empty
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
//	Lists metrics that belong to the given machine deployment.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []NodeMetric
//	  401: empty
//	  403: empty
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
//	    Patches a machine deployment that is assigned to the given cluster. Please note that at the moment only
//		   node deployment's spec can be updated by a patch, no other fields can be changed using this endpoint.
//
//	    Consumes:
//	    - application/json
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: NodeDeployment
//	      401: empty
//	      403: empty
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

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id} project restartMachineDeployment
//
//	Schedules rolling restart of a machine deployment that is assigned to the given cluster.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: NodeDeployment
//	  401: empty
//	  403: empty
func (r Routing) restartMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(machine.RestartMachineDeployment(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		machine.DecodeGetMachineDeployment,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events project listMachineDeploymentNodesEvents
//
//	Lists machine deployment events. If query parameter `type` is set to `warning` then only warning events are retrieved.
//	If the value is 'normal' then normal events are returned. If the query parameter is missing method returns all events.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Event
//	  401: empty
//	  403: empty
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
//	Deletes the given machine deployment that belongs to the cluster.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
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
//	Lists all ClusterRoles
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ClusterRole
//	  401: empty
//	  403: empty
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
//	Lists all ClusterRoles
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ClusterRoleName
//	  401: empty
//	  403: empty
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
//	Lists all Roles
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Role
//	  401: empty
//	  403: empty
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
//	Lists all Role names with namespaces
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []RoleName
//	  401: empty
//	  403: empty
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
//	Binds user to the role
//
//	 Consumes:
//	 - application/json
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: RoleBinding
//	   401: empty
//	   403: empty
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
//	Binds user to cluster role
//
//	 Consumes:
//	 - application/json
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: ClusterRoleBinding
//	   401: empty
//	   403: empty
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
//	Unbinds user from the role binding
//
//	 Consumes:
//	 - application/json
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: RoleBinding
//	   401: empty
//	   403: empty
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
//	Unbinds user from cluster role binding
//
//	 Consumes:
//	 - application/json
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: ClusterRoleBinding
//	   401: empty
//	   403: empty
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
//	List cluster role binding
//
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: []ClusterRoleBinding
//	   401: empty
//	   403: empty
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
//	List role binding
//
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: []RoleBinding
//	   401: empty
//	   403: empty
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
//	Lists names of addons that can be installed inside the user cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AccessibleAddons
//	  401: empty
//	  403: empty
func (r Routing) listInstallableAddons() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Addons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
			middleware.PrivilegedAddons(r.clusterProviderGetter, r.addonProviderGetter, r.seedsGetter),
		)(addon.ListInstallableAddonEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.kubermaticConfigGetter)),
		addon.DecodeListAddons,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/addons addon createAddonV2
//
//	Creates an addon that will belong to the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: Addon
//	  401: empty
//	  403: empty
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
//	Lists addons that belong to the given cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Addon
//	  401: empty
//	  403: empty
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
//	Gets an addon that is assigned to the given cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Addon
//	  401: empty
//	  403: empty
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
//	Patches an addon that is assigned to the given cluster.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Addon
//	  401: empty
//	  403: empty
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
//	Deletes the given addon that belongs to the cluster.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AWSSizeList
func (r Routing) listAWSSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AWSSizeNoCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.settingsProvider, r.userInfoGetter)),
		provider.DecodeAWSSizeNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/aws/subnets aws listAWSSubnetsNoCredentialsV2
//
// Lists available AWS subnets
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AWSSubnetList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPMachineSizeList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPDiskTypeList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPZoneList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPNetworkList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPSubnetworkList
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
//	Revokes the current admin token
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
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
//	Revokes the current viewer token
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: HetznerSizeList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: DigitaloceanSizeList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OpenstackSize
func (r Routing) listOpenstackSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackSizeWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider,
			r.seedsGetter, r.userInfoGetter, r.settingsProvider, r.caBundle)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/tenants openstack listOpenstackTenantsNoCredentialsV2
//
// Lists tenants from openstack
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OpenstackTenant
func (r Routing) listOpenstackTenantsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackTenantWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter,
			r.userInfoGetter, r.caBundle)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/networks openstack listOpenstackNetworksNoCredentialsV2
//
// Lists networks from openstack
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OpenstackNetwork
func (r Routing) listOpenstackNetworksNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackNetworkWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter,
			r.userInfoGetter, r.caBundle)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/securitygroups openstack listOpenstackSecurityGroupsNoCredentialsV2
//
// Lists security groups from openstack
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OpenstackSecurityGroup
func (r Routing) listOpenstackSecurityGroupsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackSecurityGroupWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider,
			r.seedsGetter, r.userInfoGetter, r.caBundle)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/subnets openstack listOpenstackSubnetsNoCredentialsV2
//
// Lists subnets from openstack
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OpenstackSubnet
func (r Routing) listOpenstackSubnetsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackSubnetsWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter,
			r.userInfoGetter, r.caBundle)),
		provider.DecodeOpenstackSubnetNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/openstack/availabilityzones openstack listOpenstackAvailabilityZonesNoCredentialsV2
//
// Lists availability zones from openstack
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OpenstackAvailabilityZone
func (r Routing) listOpenstackAvailabilityZonesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter,
			r.userInfoGetter, r.caBundle)),
		provider.DecodeOpenstackNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/openstack/subnetpools openstack listOpenstackSubnetPools
//
// Lists subnet pools from openstack
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OpenstackSubnetPool
func (r Routing) listOpenstackSubnetPools() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.OpenstackSubnetPoolEndpoint(r.seedsGetter, r.presetProvider, r.userInfoGetter, r.caBundle)),
		provider.DecodeOpenstackSubnetPoolReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/azure/sizes azure listAzureSizesNoCredentialsV2
//
// Lists available VM sizes in an Azure region
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureSizeList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureAvailabilityZonesList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []VSphereNetwork
func (r Routing) listVSphereNetworksNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VsphereNetworksWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.caBundle)),
		provider.DecodeVSphereNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vsphere/folders vsphere listVSphereFoldersNoCredentialsV2
//
// Lists folders from vsphere datacenter
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []VSphereFolder
func (r Routing) listVSphereFoldersNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VsphereFoldersWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.caBundle)),
		provider.DecodeVSphereNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/instancetypes alibaba listAlibabaInstanceTypesNoCredentialsV2
//
// Lists available Alibaba Instance Types
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AlibabaInstanceTypeList
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
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AlibabaZoneList
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

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/vswitches alibaba listAlibabaVSwitchesNoCredentialsV2
//
// Lists available Alibaba vSwitches
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AlibabaVSwitchList
func (r Routing) listAlibabaVSwitchesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AlibabaVswitchesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeAlibabaNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/packet/sizes packet listPacketSizesNoCredentialsV2
//
// Lists sizes from packet
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []PacketSizeList
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

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/anexia/vlans anexia listAnexiaVlansNoCredentialsV2
//
// Lists vlans from Anexia
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AnexiaVlanList
func (r Routing) listAnexiaVlansNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AnexiaVlansWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		provider.DecodeAnexiaNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/anexia/templates anexia listAnexiaTemplatesNoCredentialsV2
//
// Lists templates from Anexia
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AnexiaTemplateList
func (r Routing) listAnexiaTemplatesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.AnexiaTemplatesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.seedsGetter)),
		provider.DecodeAnexiaNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/vmflavors kubevirt listKubeVirtVMIPresetsNoCredentials
//
// Lists available VirtualMachineInstancePreset
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VirtualMachineInstancePresetList
//
// Deprecated: in favor of listKubeVirtInstancetypesNoCredentials.
func (r Routing) listKubeVirtVMIPresetsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.KubeVirtVMIPresetsWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeKubeVirtGenericNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/instancetypes kubevirt listKubeVirtInstancetypesNoCredentials
//
// Lists available VirtualMachineInstancetype
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VirtualMachineInstancetypeList
func (r Routing) listKubeVirtInstancetypesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.KubeVirtInstancetypesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeKubeVirtGenericNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/preferences kubevirt listKubeVirtPreferencesNoCredentials
//
// Lists available VirtualMachinePreference
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VirtualMachinePreferenceList
func (r Routing) listKubeVirtPreferencesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.KubeVirtPreferencesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeKubeVirtGenericNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/kubevirt/storageclasses kubevirt listKubevirtStorageClassesNoCredentials
//
// List Storage Classes
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: StorageClassList
func (r Routing) listKubevirtStorageClassesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.KubeVirtStorageClassesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeKubeVirtGenericNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/subnets nutanix listNutanixSubnetsNoCredentials
//
// Lists available Nutanix Subnets
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: NutanixSubnetList
func (r Routing) listNutanixSubnetsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.NutanixSubnetsWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/categories nutanix listNutanixCategoriesNoCredentials
//
// Lists available Nutanix categories
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: NutanixCategoryList
func (r Routing) listNutanixCategoriesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.NutanixCategoriesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/nutanix/categories/{category}/values nutanix listNutanixCategoryValuesNoCredentials
//
// Lists available Nutanix category values for a specific category
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: NutanixCategoryValueList
func (r Routing) listNutanixCategoryValuesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.NutanixCategoryValuesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixCategoryValuesNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/networks vmwareclouddirector listVMwareCloudDirectorNetworksNoCredentials
//
// List VMware Cloud Director OVDC Networks
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VMwareCloudDirectorNetworkList
func (r Routing) listVMwareCloudDirectorNetworksNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VMwareCloudDirectorNetworksWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVMwareCloudDirectorNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/storageprofiles vmwareclouddirector listVMwareCloudDirectorStorageProfilesNoCredentials
//
// List VMware Cloud Director Storage Profiles
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VMwareCloudDirectorStorageProfileList
func (r Routing) listVMwareCloudDirectorStorageProfilesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VMwareCloudDirectorStorageProfilesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVMwareCloudDirectorNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/catalogs vmwareclouddirector listVMwareCloudDirectorCatalogsNoCredentials
//
// List VMware Cloud Director Catalogs
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VMwareCloudDirectorCatalogList
func (r Routing) listVMwareCloudDirectorCatalogsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VMwareCloudDirectorCatalogsWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVMwareCloudDirectorNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vmwareclouddirector/templates/{catalog_name} vmwareclouddirector listVMwareCloudDirectorTemplatesNoCredentials
//
// List VMware Cloud Director Templates
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VMwareCloudDirectorTemplateList
func (r Routing) listVMwareCloudDirectorTemplatesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(provider.VMwareCloudDirectorTemplatesWithClusterCredentialsEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVMwareCloudDirectorTemplateNoCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/securitygroups azure listAzureSecurityGroups
//
// Lists available VM security groups
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureSecurityGroupsList
func (r Routing) listAzureSecurityGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureSecurityGroupsEndpoint(r.presetProvider, r.userInfoGetter)),
		provider.DecodeAzureSecurityGroupsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/resourcegroups azure listAzureResourceGroups
//
// Lists available VM resource groups
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureResourceGroupsList
func (r Routing) listAzureResourceGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureResourceGroupsEndpoint(r.presetProvider, r.userInfoGetter)),
		provider.DecodeAzureResourceGroupsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/routetables azure listAzureRouteTables
//
// Lists available VM route tables
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureRouteTablesList
func (r Routing) listAzureRouteTables() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureRouteTablesEndpoint(r.presetProvider, r.userInfoGetter)),
		provider.DecodeAzureRouteTablesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/vnets azure listAzureVnets
//
// Lists available VM virtual networks
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureVirtualNetworksList
func (r Routing) listAzureVnets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureVirtualNetworksEndpoint(r.presetProvider, r.userInfoGetter)),
		provider.DecodeAzureVirtualNetworksReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/vsphere/datastores vsphere listVSphereDatastores
//
// Lists datastores from vsphere datacenter
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []VSphereDatastoreList
func (r Routing) listVSphereDatastores() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.VsphereDatastoreEndpoint(r.seedsGetter, r.presetProvider, r.userInfoGetter, r.caBundle)),
		provider.DecodeVSphereDatastoresReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/azure/subnets azure listAzureSubnets
//
// Lists available VM subnets
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureSubnetsList
func (r Routing) listAzureSubnets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.AzureSubnetsEndpoint(r.presetProvider, r.userInfoGetter)),
		provider.DecodeAzureSubnetsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/nutanix/{dc}/clusters nutanix listNutanixClusters
//
// List clusters from Nutanix
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: NutanixClusterList
func (r Routing) listNutanixClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.NutanixClusterEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/nutanix/{dc}/projects nutanix listNutanixProjects
//
// List projects from Nutanix
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: NutanixProjectList
func (r Routing) listNutanixProjects() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.NutanixProjectEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/nutanix/{dc}/subnets nutanix listNutanixSubnets
//
// List subnets from Nutanix
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: NutanixSubnetList
func (r Routing) listNutanixSubnets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.NutanixSubnetEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixSubnetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/nutanix/{dc}/categories nutanix listNutanixCategories
//
// List category keys from Nutanix
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: NutanixCategoryList
func (r Routing) listNutanixCategories() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.NutanixCategoryEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/nutanix/{dc}/categories/{category}/values nutanix listNutanixCategoryValues
//
// List available category values for a specific category from Nutanix
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: NutanixCategoryValueList
func (r Routing) listNutanixCategoryValues() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.NutanixCategoryValuesEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeNutanixCategoryValueReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/vmwareclouddirector/{dc}/networks vmwareclouddirector listVMwareCloudDirectorNetworks
//
// List VMware Cloud Director OVDC Networks
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: VMwareCloudDirectorNetworkList
func (r Routing) listVMwareCloudDirectorNetworks() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.VMwareCloudDirectorNetworksEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVMwareCloudDirectorCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/vmwareclouddirector/{dc}/storageprofiles vmwareclouddirector listVMwareCloudDirectorStorageProfiles
//
// List VMware Cloud Director Storage Profiles
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: VMwareCloudDirectorStorageProfileList
func (r Routing) listVMwareCloudDirectorStorageProfiles() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.VMwareCloudDirectorStorageProfilesEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVMwareCloudDirectorCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/vmwareclouddirector/{dc}/catalogs vmwareclouddirector listVMwareCloudDirectorCatalogs
//
// List VMware Cloud Director Catalogs
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: VMwareCloudDirectorCatalogList
func (r Routing) listVMwareCloudDirectorCatalogs() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.VMwareCloudDirectorCatalogsEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeVMwareCloudDirectorCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/vmwareclouddirector/{dc}/templates/{catalog_name} vmwareclouddirector listVMwareCloudDirectorTemplates
//
// List VMware Cloud Director Templates
//
//	Produces:
//	- application/json
//
//	Responses:
//	default: errorResponse
//	200: VMwareCloudDirectorTemplateList
func (r Routing) listVMwareCloudDirectorTemplates() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.VMwareCloudDirectorTemplatesEndpoint(r.presetProvider, r.seedsGetter, r.userInfoGetter)),
		provider.DecodeListTemplatesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/presets preset listPresets
//
//	Lists presets
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: PresetList
//	  401: empty
//	  403: empty
func (r Routing) listPresets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.ListPresets(r.presetProvider, r.userInfoGetter)),
		preset.DecodeListPresets,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/presets/{preset_name}/status preset updatePresetStatus
//
//	    Updates the status of a preset. It can enable or disable it, so that it won't be listed by the list endpoints.
//
//
//	    Consumes:
//		   - application/json
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: empty
//	      401: empty
//	      403: empty
func (r Routing) updatePresetStatus() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.UpdatePresetStatus(r.presetProvider, r.userInfoGetter)),
		preset.DecodeUpdatePresetStatus,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/presets/{preset_name} preset deletePreset
//
//	    Removes preset.
//
//	    Consumes:
//		   - application/json
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: empty
//	      401: empty
//	      403: empty
//	      404: empty
func (r Routing) deletePreset() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.DeletePreset(r.presetProvider, r.userInfoGetter)),
		preset.DecodeDeletePreset,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/presets/{preset_name}/provider/{provider_name} preset deletePresetProvider
//
//	    Removes selected preset's provider.
//
//	    Consumes:
//		   - application/json
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: empty
//	      401: empty
//	      403: empty
//	      404: empty
func (r Routing) deletePresetProvider() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.DeletePresetProvider(r.presetProvider, r.userInfoGetter)),
		preset.DecodeDeletePresetProvider,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/{provider_name}/presets preset listProviderPresets
//
//	Lists presets for the provider
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: PresetList
//	  401: empty
//	  403: empty
func (r Routing) listProviderPresets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.ListProviderPresets(r.presetProvider, r.userInfoGetter)),
		preset.DecodeListProviderPresets,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/presets preset listProjectPresets
//
//	Lists presets in a specific project
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: PresetList
//	  401: empty
//	  403: empty
func (r Routing) listProjectPresets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.ListProjectPresets(r.presetProvider, r.userInfoGetter)),
		preset.DecodeListProjectPresets,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/{provider_name}/presets preset listProjectProviderPresets
//
//	Lists presets for the provider in a specific project
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: PresetList
//	  401: empty
//	  403: empty
func (r Routing) listProjectProviderPresets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.ListProjectProviderPresets(r.presetProvider, r.userInfoGetter)),
		preset.DecodeListProjectProviderPresets,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/providers/{provider_name}/presets preset createPreset
//
//	    Creates the preset
//
//	    Consumes:
//		   - application/json
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: Preset
//	      401: empty
//	      403: empty
func (r Routing) createPreset() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.CreatePreset(r.presetProvider, r.userInfoGetter)),
		preset.DecodeCreatePreset,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/providers/{provider_name}/presets preset updatePreset
//
//		   Updates provider preset
//
//	    Consumes:
//		   - application/json
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: Preset
//	      401: empty
//	      403: empty
func (r Routing) updatePreset() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.UpdatePreset(r.presetProvider, r.userInfoGetter)),
		preset.DecodeUpdatePreset,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/providers/{provider_name}/presets/{preset_name} preset deleteProviderPreset
//
//		   Deletes provider preset.
//
//	    This endpoint has been depreciated in favour of /presets/{presets_name} and /presets/{preset_name}/providers/{provider_name}.
//
//	    Consumes:
//		   - application/json
//
//	    Produces:
//	    - application/json
//
//	    Deprecated: true
//
//	    Responses:
//	      default: errorResponse
//	      200: empty
//	      401: empty
//	      403: empty
func (r Routing) deleteProviderPreset() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.DeleteProviderPreset(r.presetProvider, r.userInfoGetter)),
		preset.DecodeDeleteProviderPreset,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/presets/{preset_name}/stats preset getPresetStats
//
//	Gets presets stats.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: PresetStats
//	  401: empty
//	  403: empty
//	  404: empty
func (r Routing) getPresetStats() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(preset.GetPresetStats(r.presetProvider, r.userInfoGetter, r.clusterProviderGetter, r.seedsGetter, r.clusterTemplateProvider)),
		preset.DecodeGetPresetStats,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/{provider_name}/versions version listVersionsByProvider
//
// Lists all versions which don't result in automatic updates for a given provider
//
//	    Consumes:
//		   - application/json
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: VersionList
//	      401: empty
//	      403: empty
func (r Routing) listVersionsByProvider() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(version.ListVersions(r.kubermaticConfigGetter)),
		version.DecodeListProviderVersions,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config project getAlertmanager
//
//	Gets the alertmanager configuration for the specified cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Alertmanager
//	  401: empty
//	  403: empty
func (r Routing) getAlertmanager() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Alertmanagers(r.clusterProviderGetter, r.alertmanagerProviderGetter, r.seedsGetter),
			middleware.PrivilegedAlertmanagers(r.clusterProviderGetter, r.alertmanagerProviderGetter, r.seedsGetter),
		)(alertmanager.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		alertmanager.DecodeGetAlertmanagerReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config project updateAlertmanager
//
//	Updates an alertmanager configuration for the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Alertmanager
//	  401: empty
//	  403: empty
func (r Routing) updateAlertmanager() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Alertmanagers(r.clusterProviderGetter, r.alertmanagerProviderGetter, r.seedsGetter),
			middleware.PrivilegedAlertmanagers(r.clusterProviderGetter, r.alertmanagerProviderGetter, r.seedsGetter),
		)(alertmanager.UpdateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		alertmanager.DecodeUpdateAlertmanagerReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/alertmanager/config project resetAlertmanager
//
//	Resets the alertmanager configuration to default for the specified cluster.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) resetAlertmanager() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.Alertmanagers(r.clusterProviderGetter, r.alertmanagerProviderGetter, r.seedsGetter),
			middleware.PrivilegedAlertmanagers(r.clusterProviderGetter, r.alertmanagerProviderGetter, r.seedsGetter),
		)(alertmanager.ResetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		alertmanager.DecodeResetAlertmanagerReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/gke/images gke listProjectGKEImages
//
// Lists GKE image types.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEImageList
func (r Routing) listProjectGKEImages() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEImagesEndpoint(r.presetProvider, r.userInfoGetter, true)),
		externalcluster.DecodeGKEProjectVMReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/gke/zones gke listProjectGKEZones
//
// Lists GKE zones.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEZoneList
func (r Routing) listProjectGKEZones() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEZonesEndpoint(r.presetProvider, r.userInfoGetter, true)),
		externalcluster.DecodeGKEProjectCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/gke/vmsizes gke listProjectGKEVMSizes
//
// Lists GKE VM sizes.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPMachineSizeList
func (r Routing) listProjectGKEVMSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEVMSizesEndpoint(r.presetProvider, r.userInfoGetter, true)),
		externalcluster.DecodeGKEProjectVMReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/gke/disktypes gke listProjectGKEDiskTypes
//
// Lists GKE machine disk types.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEDiskTypeList
func (r Routing) listProjectGKEDiskTypes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEDiskTypesEndpoint(r.presetProvider, r.userInfoGetter, true)),
		externalcluster.DecodeGKEProjectVMReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/gke/versions gke listProjectGKEVersions
//
// Lists GKE versions.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []MasterVersion
func (r Routing) listProjectGKEVersions() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEVersionsEndpoint(r.presetProvider, r.userInfoGetter, true)),
		externalcluster.DecodeGKEProjectVersionsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/gke/validatecredentials gke validateProjectGKECredentials
//
// Validates GKE credentials.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
func (r Routing) validateProjectGKECredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEValidateCredentialsEndpoint(r.presetProvider, r.userInfoGetter, true)),
		externalcluster.DecodeGKEProjectCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/seeds/{seed_name}/settings seed getSeedSettings
//
//	Gets the seed settings.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: SeedSettings
//	  401: empty
//	  403: empty
func (r Routing) getSeedSettings() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(seedsettings.GetSeedSettingsEndpoint(r.seedsGetter)),
		seedsettings.DecodeGetSeedSettingsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clustertemplates project createClusterTemplate
//
//	Creates a cluster templates for the given project.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: ClusterTemplate
//	  401: empty
//	  403: empty
func (r Routing) createClusterTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(clustertemplate.CreateEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.clusterTemplateProvider, r.seedsGetter, r.presetProvider, r.caBundle, r.exposeStrategy, r.sshKeyProvider, r.kubermaticConfigGetter, r.features)),
		clustertemplate.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clustertemplates/import project importClusterTemplate
//
//	Import a cluster templates for the given project.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: ClusterTemplate
//	  401: empty
//	  403: empty
func (r Routing) importClusterTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(clustertemplate.ImportEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.clusterTemplateProvider, r.seedsGetter, r.presetProvider, r.caBundle, r.exposeStrategy, r.sshKeyProvider, r.kubermaticConfigGetter, r.features)),
		clustertemplate.DecodeImportReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clustertemplates project listClusterTemplates
//
//	List cluster templates for the given project.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ClusterTemplateList
//	  401: empty
//	  403: empty
func (r Routing) listClusterTemplates() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(clustertemplate.ListEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.clusterTemplateProvider)),
		clustertemplate.DecodeListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clustertemplates/{template_id} project getClusterTemplate
//
//	Get cluster template.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ClusterTemplate
//	  401: empty
//	  403: empty
func (r Routing) getClusterTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(clustertemplate.GetEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.clusterTemplateProvider)),
		clustertemplate.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clustertemplates/{template_id}/export project exportClusterTemplate
//
//	Export cluster template to file.
//
//
//	Produces:
//	- application/octet-stream
//
//	Responses:
//	  default: errorResponse
//	  200: ClusterTemplate
//	  401: empty
//	  403: empty
func (r Routing) exportClusterTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(clustertemplate.ExportEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.clusterTemplateProvider)),
		clustertemplate.DecodeExportReq,
		clustertemplate.EncodeClusterTemplate,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clustertemplates/{template_id} project deleteClusterTemplate
//
//	Delete cluster template.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteClusterTemplate() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(clustertemplate.DeleteEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.clusterTemplateProvider)),
		clustertemplate.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clustertemplates/{template_id}/instances project createClusterTemplateInstance
//
//	Create cluster template instance.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: ClusterTemplateInstance
//	  401: empty
//	  403: empty
func (r Routing) createClusterTemplateInstance() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(clustertemplate.CreateInstanceEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.clusterTemplateProvider, r.seedsGetter, r.clusterTemplateInstanceProviderGetter)),
		clustertemplate.DecodeCreateInstanceReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups/{rulegroup_id} rulegroup getRuleGroup
//
//	Gets a specified rule group for the given cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) getRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.RuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroup.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		rulegroup.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups rulegroup listRuleGroups
//
//	Lists rule groups that belong to the given cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) listRuleGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.RuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroup.ListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		rulegroup.DecodeListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups rulegroup createRuleGroup
//
//	Creates a rule group that will belong to the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) createRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.RuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroup.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		rulegroup.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups/{rulegroup_id} rulegroup updateRuleGroup
//
//	Updates the specified rule group for the given cluster.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) updateRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.RuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroup.UpdateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		rulegroup.DecodeUpdateReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/rulegroups/{rulegroup_id} rulegroup deleteRuleGroup
//
//	Deletes the given rule group that belongs to the cluster.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
func (r Routing) deleteRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.RuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroup.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		rulegroup.DecodeDeleteReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/externalccmmigration migrateClusterToExternalCCM
//
//	   Enable the migration to the external CCM for the given cluster
//
//		   Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: empty
//	      401: empty
//	      403: empty
func (r Routing) migrateClusterToExternalCCM() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.MigrateEndpointToExternalCCM(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter, r.kubermaticConfigGetter)),
		cluster.DecodeGetClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/allowedregistries allowedregistry createAllowedRegistry
//
//	Creates a allowed registry
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: AllowedRegistry
//	  401: empty
//	  403: empty
func (r Routing) createAllowedRegistry() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(allowedregistry.CreateEndpoint(r.userInfoGetter, r.privilegedAllowedRegistryProvider)),
		allowedregistry.DecodeCreateAllowedRegistryRequest,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/allowedregistries allowedregistry listAllowedRegistries
//
//	List allowed registries.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []AllowedRegistry
//	  401: empty
//	  403: empty
func (r Routing) listAllowedRegistries() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(allowedregistry.ListEndpoint(r.userInfoGetter, r.privilegedAllowedRegistryProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/allowedregistries/{allowed_registry} allowedregistries getAllowedRegistry
//
//	Get allowed registries specified by name
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AllowedRegistry
//	  401: empty
//	  403: empty
func (r Routing) getAllowedRegistry() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(allowedregistry.GetEndpoint(r.userInfoGetter, r.privilegedAllowedRegistryProvider)),
		allowedregistry.DecodeGetAllowedRegistryRequest,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/allowedregistries/{allowed_registry} allowedregistries deleteAllowedRegistry
//
//	Deletes the given allowed registry.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
func (r Routing) deleteAllowedRegistry() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(allowedregistry.DeleteEndpoint(r.userInfoGetter, r.privilegedAllowedRegistryProvider)),
		allowedregistry.DecodeGetAllowedRegistryRequest,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/allowedregistries/{allowed_registry} allowedregistries patchAllowedRegistry
//
//	Patch a specified allowed registry
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ConstraintTemplate
//	  401: empty
//	  403: empty
func (r Routing) patchAllowedRegistry() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(allowedregistry.PatchEndpoint(r.userInfoGetter, r.privilegedAllowedRegistryProvider)),
		allowedregistry.DecodePatchAllowedRegistryReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs etcdbackupconfig createEtcdBackupConfig
//
//	Creates a etcd backup config that will belong to the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: EtcdBackupConfig
//	  401: empty
//	  403: empty
func (r Routing) createEtcdBackupConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
		)(etcdbackupconfig.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdbackupconfig.DecodeCreateEtcdBackupConfigReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_id} etcdbackupconfig getEtcdBackupConfig
//
//	Gets a etcd backup config for a given cluster based on its id
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EtcdBackupConfig
//	  401: empty
//	  403: empty
func (r Routing) getEtcdBackupConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
		)(etcdbackupconfig.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdbackupconfig.DecodeGetEtcdBackupConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs etcdbackupconfig listEtcdBackupConfig
//
//	List etcd backup configs for a given cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []EtcdBackupConfig
//	  401: empty
//	  403: empty
func (r Routing) listEtcdBackupConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
		)(etcdbackupconfig.ListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdbackupconfig.DecodeListEtcdBackupConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_id} etcdbackupconfig deleteEtcdBackupConfig
//
//	Deletes a etcd backup config for a given cluster based on its id
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteEtcdBackupConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
		)(etcdbackupconfig.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdbackupconfig.DecodeGetEtcdBackupConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_id} etcdbackupconfig patchEtcdBackupConfig
//
//	Patches a etcd backup config for a given cluster based on its id
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EtcdBackupConfig
//	  401: empty
//	  403: empty
func (r Routing) patchEtcdBackupConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdBackupConfig(r.clusterProviderGetter, r.etcdBackupConfigProviderGetter, r.seedsGetter),
		)(etcdbackupconfig.PatchEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdbackupconfig.DecodePatchEtcdBackupConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/etcdbackupconfigs etcdbackupconfig listProjectEtcdBackupConfig
//
//	List etcd backup configs for a given project
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []EtcdBackupConfig
//	  401: empty
//	  403: empty
func (r Routing) listProjectEtcdBackupConfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.EtcdBackupConfigProject(r.etcdBackupConfigProjectProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdBackupConfigProject(r.etcdBackupConfigProjectProviderGetter, r.seedsGetter),
		)(etcdbackupconfig.ProjectListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdbackupconfig.DecodeListProjectEtcdBackupConfigReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores etcdrestore createEtcdRestore
//
//	Creates a etcd backup restore for a given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: EtcdBackupConfig
//	  401: empty
//	  403: empty
func (r Routing) createEtcdRestore() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
		)(etcdrestore.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdrestore.DecodeCreateEtcdRestoreReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name} etcdrestore getEtcdRestore
//
//	Gets a etcd backup restore for a given cluster based on its name
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EtcdRestore
//	  401: empty
//	  403: empty
func (r Routing) getEtcdRestore() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
		)(etcdrestore.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdrestore.DecodeGetEtcdRestoreReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores etcdrestore listEtcdRestore
//
//	List etcd backup restores for a given cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []EtcdRestore
//	  401: empty
//	  403: empty
func (r Routing) listEtcdRestore() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
		)(etcdrestore.ListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdrestore.DecodeListEtcdRestoreReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/etcdrestores/{er_name} etcdrestore deleteEtcdRestore
//
//	Deletes a etcd restore config for a given cluster based on its name
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
//	  409: errorResponse
func (r Routing) deleteEtcdRestore() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.EtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdRestore(r.clusterProviderGetter, r.etcdRestoreProviderGetter, r.seedsGetter),
		)(etcdrestore.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdrestore.DecodeGetEtcdRestoreReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/etcdrestores etcdrestore listProjectEtcdRestore
//
//	List etcd backup restores for a given project
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []EtcdRestore
//	  401: empty
//	  403: empty
func (r Routing) listProjectEtcdRestore() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.EtcdRestoreProject(r.etcdRestoreProjectProviderGetter, r.seedsGetter),
			middleware.PrivilegedEtcdRestoreProject(r.etcdRestoreProjectProviderGetter, r.seedsGetter),
		)(etcdrestore.ProjectListEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		etcdrestore.DecodeListProjectEtcdRestoreReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/seeds/{seed_name}/backupcredentials backupcredentials createOrUpdateBackupCredentials
//
//	Creates or updates backup credentials for a given seed
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) createOrUpdateBackupCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.BackupCredentials(r.backupCredentialsProviderGetter, r.seedsGetter),
		)(backupcredentials.CreateOrUpdateEndpoint(r.userInfoGetter, r.seedsGetter, r.seedProvider)),
		backupcredentials.DecodeBackupCredentialsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting mlaadminsetting getMLAAdminSetting
//
//	Gets MLA Admin settings for the given cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: MLAAdminSetting
//	  401: empty
//	  403: empty
func (r Routing) getMLAAdminSetting() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.PrivilegedMLAAdminSetting(r.clusterProviderGetter, r.privilegedMLAAdminSettingProviderGetter, r.seedsGetter),
		)(mlaadminsetting.GetEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		mlaadminsetting.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting mlaadminsetting createMLAAdminSetting
//
//	Creates MLA admin setting that will belong to the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: MLAAdminSetting
//	  401: empty
//	  403: empty
func (r Routing) createMLAAdminSetting() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.PrivilegedMLAAdminSetting(r.clusterProviderGetter, r.privilegedMLAAdminSettingProviderGetter, r.seedsGetter),
		)(mlaadminsetting.CreateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		mlaadminsetting.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting mlaadminsetting updateMLAAdminSetting
//
//	Updates the MLA admin setting for the given cluster.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: MLAAdminSetting
//	  401: empty
//	  403: empty
func (r Routing) updateMLAAdminSetting() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.PrivilegedMLAAdminSetting(r.clusterProviderGetter, r.privilegedMLAAdminSettingProviderGetter, r.seedsGetter),
		)(mlaadminsetting.UpdateEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		mlaadminsetting.DecodeUpdateReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/mlaadminsetting mlaadminsetting deleteMLAAdminSetting
//
//	Deletes the MLA admin setting that belongs to the cluster.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
func (r Routing) deleteMLAAdminSetting() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.PrivilegedMLAAdminSetting(r.clusterProviderGetter, r.privilegedMLAAdminSettingProviderGetter, r.seedsGetter),
		)(mlaadminsetting.DeleteEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		mlaadminsetting.DecodeDeleteReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/users user listUser
//
//	List users
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []User
//	  401: empty
//	  403: empty
func (r Routing) listUser() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(user.ListEndpoint(r.userInfoGetter, r.userProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/featuregates get status of feature gates
//
//	Status of feature gates
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: FeatureGates
//	  401: errorResponse
//	  403: errorResponse
func (r Routing) getFeatureGates() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(featuregates.GetEndpoint(r.featureGatesProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/gke/clusters project listGKEClusters
//
// Lists GKE clusters.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEClusterList
func (r Routing) listGKEClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEClustersEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.presetProvider, false)),
		externalcluster.DecodeGKEProjectCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/gke/images gke listGKEImages
//
// Lists GKE image types
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEImageList
func (r Routing) listGKEImages() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEImagesEndpoint(r.presetProvider, r.userInfoGetter, false)),
		externalcluster.DecodeGKEVMReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/gke/zones gke listGKEZones
//
// Lists GKE zones
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEZoneList
func (r Routing) listGKEZones() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEZonesEndpoint(r.presetProvider, r.userInfoGetter, false)),
		externalcluster.DecodeGKECommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/gke/vmsizes gke listGKEVMSizes
//
// Lists GKE vmsizes
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPMachineSizeList
func (r Routing) listGKEVMSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEVMSizesEndpoint(r.presetProvider, r.userInfoGetter, false)),
		externalcluster.DecodeGKEVMReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/gke/disktypes gke listGKEDiskTypes
//
// Gets GKE machine disk types.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEDiskTypeList
func (r Routing) listGKEDiskTypes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEDiskTypesEndpoint(r.presetProvider, r.userInfoGetter, false)),
		externalcluster.DecodeGKEVMReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/gke/versions gke listGKEVersions
//
// Lists GKE versions
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []MasterVersion
func (r Routing) listGKEVersions() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEVersionsEndpoint(r.presetProvider, r.userInfoGetter, false)),
		externalcluster.DecodeGKEVersionsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/gke/validatecredentials gke validateGKECredentials
//
// Validates GKE credentials
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
func (r Routing) validateGKECredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEValidateCredentialsEndpoint(r.presetProvider, r.userInfoGetter, false)),
		externalcluster.DecodeGKECommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/eks/validatecredentials eks validateEKSCredentials
//
// Validates EKS credentials
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
func (r Routing) validateEKSCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSValidateCredentialsEndpoint(r.presetProvider, r.userInfoGetter)),
		externalcluster.DecodeEKSTypesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/instancetypes eks listEKSInstanceTypesNoCredentials
//
//	Gets the EKS Instance types for node group based on architecture.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSInstanceTypeList
//	  401: empty
//	  403: empty
func (r Routing) listEKSInstanceTypesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSInstanceTypesWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeEKSNoCredentialSizeReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/subnets eks listEKSSubnetsNoCredentials
//
//	Gets the EKS Subnets for node group.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSSubnetList
//	  401: empty
//	  403: empty
func (r Routing) listEKSSubnetsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSSubnetsWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeEKSSubnetsNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/vpcs eks listEKSVPCsNoCredentials
//
//	Gets the EKS vpc's for node group.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSVPCList
//	  401: empty
//	  403: empty
func (r Routing) listEKSVPCsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSVPCsWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeEKSNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/eks/noderoles eks listEKSNodeRolesNoCredentials
//
//		List EKS Node IAM Roles.
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//	      default: errorResponse
//	      200: EKSNodeRoleList
//	      401: empty
//	      403: empty
func (r Routing) listEKSNodeRolesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSNodeRolesWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeEKSNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/aks/validatecredentials aks validateAKSCredentials
//
// Validates AKS credentials
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
func (r Routing) validateAKSCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.AKSValidateCredentialsEndpoint(r.presetProvider, r.userInfoGetter)),
		externalcluster.DecodeAKSTypesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/aks/vmsizes aks listAKSVMSizes
//
// List AKS available VM sizes in an Azure region.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AKSVMSizeList
func (r Routing) listAKSVMSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListAKSVMSizesEndpoint(r.presetProvider, r.userInfoGetter)),
		externalcluster.DecodeAKSVMSizesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/aks/resourcegroups aks listAKSResourceGroups
//
//	List resource groups in an Azure subscription.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AzureResourceGroupList
func (r Routing) listAKSResourceGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListAKSResourceGroupsEndpoint(r.presetProvider, r.userInfoGetter)),
		externalcluster.DecodeAKSCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/aks/locations aks listAKSLocations
//
// List AKS recommended Locations.
//
//	    Produces:
//	    - application/json
//
//	    Responses:
//		 default: errorResponse
//		 200: AKSLocationList
func (r Routing) listAKSLocations() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListAKSLocationsEndpoint(r.presetProvider, r.userInfoGetter)),
		externalcluster.DecodeAKSCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/aks/modes aks listAKSNodePoolModes
//
//	Gets the AKS node pool modes.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AKSNodePoolModes
//	  401: empty
//	  403: empty
func (r Routing) listAKSNodePoolModes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.AKSNodePoolModesEndpoint()),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/aks/versions aks listAKSVersions
//
// Lists AKS versions
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []MasterVersion
func (r Routing) listAKSVersions() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.AKSVersionsEndpoint(r.kubermaticConfigGetter, r.externalClusterProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/eks/clusters project listEKSClusters
//
// Lists EKS clusters
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSClusterList
func (r Routing) listEKSClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEKSClustersEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.presetProvider)),
		externalcluster.DecodeEKSClusterListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/eks/vpcs eks listEKSVPCS
//
// Lists EKS vpc's
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSVPCList
func (r Routing) listEKSVPCS() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEKSVPCEndpoint(r.userInfoGetter, r.presetProvider)),
		externalcluster.DecodeEKSTypesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/eks/subnets eks listEKSSubnets
//
// Lists EKS subnet list.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSSubnetList
func (r Routing) listEKSSubnets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEKSSubnetsEndpoint(r.userInfoGetter, r.presetProvider)),
		externalcluster.DecodeEKSReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/eks/securitygroups eks listEKSSecurityGroups
//
//	List EKS securitygroup list.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSSecurityGroupList
//	  401: empty
//	  403: empty
func (r Routing) listEKSSecurityGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEKSSecurityGroupsEndpoint(r.userInfoGetter, r.presetProvider)),
		externalcluster.DecodeEKSReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/eks/regions eks listEKSRegions
//
//	List EKS regions.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []EKSRegionList
//	  401: empty
//	  403: empty
func (r Routing) listEKSRegions() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEKSRegionsEndpoint(r.userInfoGetter, r.presetProvider)),
		externalcluster.DecodeEKSCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/eks/clusterroles eks listEKSClusterRoles
//
//	List EKS Cluster Service Roles.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSClusterRoleList
//	  401: empty
//	  403: empty
func (r Routing) listEKSClusterRoles() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListEKSClusterRolesEndpoint(r.userInfoGetter, r.presetProvider)),
		externalcluster.DecodeEKSCommonReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/eks/versions eks listEKSVersions
//
// Lists EKS versions
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []MasterVersion
func (r Routing) listEKSVersions() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSVersionsEndpoint(r.kubermaticConfigGetter, r.externalClusterProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/eks/amitypes eks listEKSAMITypes
//
//	Gets the EKS AMI types for node group.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSAMITypeList
//	  401: empty
//	  403: empty
func (r Routing) listEKSAMITypes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSAMITypesEndpoint()),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/eks/capacitytypes eks listEKSCapacityTypes
//
//	Gets the EKS Capacity types for node group.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: EKSCapacityTypeList
//	  401: empty
//	  403: empty
func (r Routing) listEKSCapacityTypes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.EKSCapacityTypesEndpoint()),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/providers/aks/clusters project listAKSClusters
//
// Lists AKS clusters
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AKSClusterList
func (r Routing) listAKSClusters() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListAKSClustersEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.presetProvider)),
		externalcluster.DecodeAKSClusterListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/kubevirt/vmflavors kubevirt listKubeVirtVMIPresets
//
// Lists available KubeVirt VirtualMachineInstancePreset.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VirtualMachineInstancePresetList
//
// Deprecated: in favor of listKubeVirtInstancetypes.
func (r Routing) listKubeVirtVMIPresets() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.KubeVirtVMIPresetsEndpoint(r.presetProvider, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeKubeVirtGenericReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/kubevirt/instancetypes kubevirt listKubeVirtInstancetypes
//
// Lists available KubeVirt VirtualMachineInstancetype.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VirtualMachineInstancetypeList
func (r Routing) listKubeVirtInstancetypes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.KubeVirtInstancetypesEndpoint(r.presetProvider, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeKubeVirtGenericReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/kubevirt/preferences kubevirt listKubeVirtPreferences
//
// Lists available KubeVirt VirtualMachinePreference.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: VirtualMachinePreferenceList
func (r Routing) listKubeVirtPreferences() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.KubeVirtPreferencesEndpoint(r.presetProvider, r.userInfoGetter, r.settingsProvider)),
		provider.DecodeKubeVirtGenericReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/providers/kubevirt/storageclasses kubevirt listKubevirtStorageClasses
//
// Lists available K8s StorageClasses in the Kubevirt cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: StorageClassList
func (r Routing) listKubevirtStorageClasses() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(provider.KubeVirtStorageClassesEndpoint(r.presetProvider, r.userInfoGetter)),
		provider.DecodeKubeVirtGenericReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// getExternalClusterKubeconfig returns the kubeconfig for the external cluster.
// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/kubeconfig project getExternalClusterKubeconfig
//
//	Gets the kubeconfig for the specified external cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: Kubeconfig
//	  401: empty
//	  403: empty
func (r Routing) getExternalClusterKubeconfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GetKubeconfigEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		cluster.EncodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/seeds/{seed_name}/rulegroups/{rulegroup_id} rulegroup getAdminRuleGroup
//
//	Gets a specified rule group for a given Seed.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) getAdminRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroupadmin.GetEndpoint(r.userInfoGetter)),
		rulegroupadmin.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/seeds/{seed_name}/rulegroups rulegroup listAdminRuleGroups
//
//	Lists rule groups that belong to a given Seed.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) listAdminRuleGroups() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroupadmin.ListEndpoint(r.userInfoGetter)),
		rulegroupadmin.DecodeListReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/seeds/{seed_name}/rulegroups rulegroup createAdminRuleGroup
//
//	Creates a rule group that will belong to the given Seed
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) createAdminRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroupadmin.CreateEndpoint(r.userInfoGetter)),
		rulegroupadmin.DecodeCreateReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/seeds/{seed_name}/rulegroups/{rulegroup_id} rulegroup updateAdminRuleGroup
//
//	Updates the specified rule group for the given Seed.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: RuleGroup
//	  401: empty
//	  403: empty
func (r Routing) updateAdminRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroupadmin.UpdateEndpoint(r.userInfoGetter)),
		rulegroupadmin.DecodeUpdateReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/seeds/{seed_name}/rulegroups/{rulegroup_id} rulegroup deleteAdminRuleGroup
//
//	Deletes the given rule group that belongs to the Seed.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
func (r Routing) deleteAdminRuleGroup() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedRuleGroups(r.clusterProviderGetter, r.ruleGroupProviderGetter, r.seedsGetter),
		)(rulegroupadmin.DeleteEndpoint(r.userInfoGetter)),
		rulegroupadmin.DecodeDeleteReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id} project patchExternalClusterMachineDeployments
//
//	Patches the given cluster using JSON Merge Patch method
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ExternalClusterMachineDeployment
//	  401: empty
//	  403: empty
func (r Routing) patchExternalClusterMachineDeployments() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.PatchMachineDeploymentEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodePatchMachineDeploymentReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id} project getExternalClusterMachineDeployment
//
//	Gets an external cluster machine deployments.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: ExternalClusterMachineDeployment
//	   401: empty
//	   403: empty
func (r Routing) getExternalClusterMachineDeployment() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GetMachineDeploymentEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeGetMachineDeploymentReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/upgrades project getExternalClusterMachineDeploymentUpgrades
//
//	Gets an external cluster machine deployments upgrade versions.
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: []MasterVersion
//	   401: empty
//	   403: empty
func (r Routing) getExternalClusterMachineDeploymentUpgrades() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GetMachineDeploymentUpgradesEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeGetMachineDeploymentReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes project listExternalClusterMachineDeploymentNodes
//
//	Gets an external cluster machine deployment nodes.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ExternalClusterNode
//	  401: empty
//	  403: empty
func (r Routing) listExternalClusterMachineDeploymentNodes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListMachineDeploymentNodesEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeListMachineDeploymentNodesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/metrics project listExternalClusterMachineDeploymentMetrics
//
//	List an external cluster machine deployment metrics.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []NodeMetric
//	  401: empty
//	  403: empty
func (r Routing) listExternalClusterMachineDeploymentMetrics() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListMachineDeploymentMetricsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeGetMachineDeploymentReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}/nodes/events project listExternalClusterMachineDeploymentEvents
//
//	List an external cluster machine deployment events.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []Event
//	  401: empty
//	  403: empty
func (r Routing) listExternalClusterMachineDeploymentEvents() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.ListMachineDeploymentEventsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider)),
		externalcluster.DecodeListMachineDeploymentNodesEvents,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/aks/versions aks listAKSNodeVersionsNoCredentials
//
//	Gets AKS nodepool available versions.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []MasterVersion
//	  401: empty
//	  403: empty
func (r Routing) listAKSNodeVersionsNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.AKSNodeVersionsWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/aks/vmsizes aks listAKSVMSizesNoCredentials
//
//	Gets AKS available VM sizes in an Azure region.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: AKSVMSizeList
//	  401: empty
//	  403: empty
func (r Routing) listAKSVMSizesNoCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.AKSSizesWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeAKSNoCredentialReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/images gke listGKEClusterImages
//
//	Gets GKE cluster images.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEImageList
//	  401: empty
//	  403: empty
func (r Routing) listGKEClusterImages() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEImagesWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/zones gke listGKEClusterZones
//
//	Gets GKE cluster zones.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GKEZoneList
//	  401: empty
//	  403: empty
func (r Routing) listGKEClusterZones() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEZonesWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/sizes gke listGKEClusterSizes
//
//	Gets GKE cluster machine sizes.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPMachineSizeList
//	  401: empty
//	  403: empty
func (r Routing) listGKEClusterSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKESizesWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/kubernetes/clusters/{cluster_id}/providers/gke/disktypes gke listGKEClusterDiskTypes
//
//	Gets GKE cluster machine disk types.
//
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GCPDiskTypeList
//	  401: empty
//	  403: empty
func (r Routing) listGKEClusterDiskTypes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(externalcluster.GKEDiskTypesWithClusterCredentialsEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider, r.externalClusterProvider, r.privilegedExternalClusterProvider, r.settingsProvider)),
		externalcluster.DecodeGetReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/backupdestinations project getBackupDestinationNames
//
//	Gets possible backup destination names for a cluster
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: BackupDestinationNames
//	   401: empty
//	   403: empty
func (r Routing) getBackupDestinationNames() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(backupdestinations.GetEndpoint(r.projectProvider, r.privilegedProjectProvider, r.seedsGetter, r.userInfoGetter)),
		backupdestinations.DecodeGetBackupDestinationNamesReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/cni/{cni_plugin_type}/versions cniversion listVersionsByCNIPlugin
//
// Lists all CNI Plugin versions that are supported for a given CNI plugin type
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: CNIVersions
//	  401: empty
//	  403: empty
func (r Routing) listVersionsByCNIPlugin() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(cniversion.ListVersions()),
		cniversion.DecodeListCNIPluginVersions,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/cniversions project listCNIPluginVersionsForCluster
//
// Lists CNI plugin versions for a given cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: CNIVersions
//	  401: empty
//	  403: empty
func (r Routing) listCNIPluginVersionsForCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cniversion.ListVersionsForCluster(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		cniversion.DecodeListCNIPluginVersionsForClusterReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /providers/{provider_name}/dc/{dc}/networkdefaults networkdefaults getNetworkDefaults
//
//	Retrieves the cluster networking defaults for the given provider and datacenter.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: NetworkDefaults
//	  401: empty
//	  403: empty
func (r Routing) getNetworkDefaults() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(networkdefaults.GetNetworkDefaultsEndpoint(r.seedsGetter, r.userInfoGetter)),
		networkdefaults.DecodeGetNetworkDefaultsReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/kubeconfig/secret createOIDCKubeconfigSecret
//
//	Starts OIDC flow and generates kubeconfig, the generated config
//	contains OIDC provider authentication info. The kubeconfig is stored in the secret.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  201: empty
//	  401: empty
//	  403: empty
func (r Routing) createOIDCKubeconfigSecret(oidcCfg common.OIDCConfiguration) http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.UserInfoUnauthorized(r.userProjectMapper, r.userProvider),
		)(webterminal.CreateOIDCKubeconfigSecretEndpoint(r.projectProvider, r.privilegedProjectProvider, r.oidcIssuerVerifier, oidcCfg)),
		webterminal.DecodeCreateOIDCKubeconfig,
		webterminal.EncodeOIDCKubeconfig,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/quota project getProjectQuota
//
//	Returns Resource Quota for a given project.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ResourceQuota
//	  401: empty
//	  403: empty
func (r Routing) getProjectQuota() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(resourcequota.GetForProjectEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter, r.resourceQuotaProvider)),
		common.DecodeGetProject,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/quotas/{quota_name} resourceQuota admin getResourceQuota
//
//	Gets a specific Resource Quota.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: ResourceQuota
//	  401: empty
//	  403: empty
func (r Routing) getResourceQuota() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(resourcequota.GetResourceQuotaEndpoint(r.userInfoGetter, r.resourceQuotaProvider, r.privilegedProjectProvider)),
		resourcequota.DecodeResourceQuotasReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/quotas resourceQuota admin listResourceQuotas
//
//	Gets a Resource Quota list.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ResourceQuota
//	  401: empty
//	  403: empty
func (r Routing) listResourceQuotas() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(resourcequota.ListResourceQuotasEndpoint(r.userInfoGetter, r.resourceQuotaProvider, r.projectProvider)),
		resourcequota.DecodeListResourceQuotasReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/quotas resourceQuota admin createResourceQuota
//
//	Creates a new Resource Quota.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: empty
//	  401: empty
//	  403: empty
func (r Routing) createResourceQuota() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(resourcequota.CreateResourceQuotaEndpoint(r.userInfoGetter, r.resourceQuotaProvider)),
		resourcequota.DecodeCreateResourceQuotasReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/quotas/{quota_name} resourceQuota admin putResourceQuota
//
//	Updates an existing Resource Quota.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) putResourceQuota() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(resourcequota.PutResourceQuotaEndpoint(r.userInfoGetter, r.resourceQuotaProvider)),
		resourcequota.DecodePutResourceQuotasReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/quotas/{quota_name} resourceQuota admin deleteResourceQuota
//
//	Removes an existing Resource Quota.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteResourceQuota() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(resourcequota.DeleteResourceQuotaEndpoint(r.userInfoGetter, r.resourceQuotaProvider)),
		resourcequota.DecodeResourceQuotasReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route get /api/v2/projects/{project_id}/groupbindings project listGroupProjectBinding
//
//	Lists project's group bindings.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []GroupProjectBinding
//	  401: empty
//	  403: empty
func (r Routing) listGroupProjectBindings() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(groupprojectbinding.ListGroupProjectBindingsEndpoint(
			r.userInfoGetter,
			r.projectProvider,
			r.privilegedProjectProvider,
			r.groupProjectBindingProvider,
		)),
		common.DecodeGetProject,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route get /api/v2/projects/{project_id}/groupbindings/{binding_name} project getGroupProjectBinding
//
//	Get project group binding.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GroupProjectBinding
//	  401: empty
//	  403: empty
func (r Routing) getGroupProjectBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(groupprojectbinding.GetGroupProjectBindingEndpoint(
			r.userInfoGetter,
			r.projectProvider,
			r.privilegedProjectProvider,
			r.groupProjectBindingProvider,
		)),
		groupprojectbinding.DecodeGetGroupProjectBindingReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route post /api/v2/projects/{project_id}/groupbindings project createGroupProjectBinding
//
//	Create project group binding.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: GroupProjectBinding
//	  401: empty
//	  403: empty
func (r Routing) createGroupProjectBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(groupprojectbinding.CreateGroupProjectBindingEndpoint(
			r.userInfoGetter,
			r.projectProvider,
			r.privilegedProjectProvider,
			r.groupProjectBindingProvider,
		)),
		groupprojectbinding.DecodeCreateGroupProjectBindingReq,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route delete /api/v2/projects/{project_id}/groupbindings/{binding_name} project deleteGroupProjectBinding
//
//	Delete project group binding.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteGroupProjectBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(groupprojectbinding.DeleteGroupProjectBindingEndpoint(
			r.userInfoGetter,
			r.projectProvider,
			r.privilegedProjectProvider,
			r.groupProjectBindingProvider,
		)),
		groupprojectbinding.DecodeDeleteGroupProjectBindingReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route patch /api/v2/projects/{project_id}/groupbindings/{binding_name} project patchGroupProjectBinding
//
//	Patch project group binding.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: GroupProjectBinding
//	  401: empty
//	  403: empty
func (r Routing) patchGroupProjectBinding() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(groupprojectbinding.PatchGroupProjectBindingEndpoint(
			r.userInfoGetter,
			r.projectProvider,
			r.privilegedProjectProvider,
			r.groupProjectBindingProvider,
		)),
		groupprojectbinding.DecodePatchGroupProjectBindingReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations applications listApplicationInstallations
//
//	List ApplicationInstallations which belong to the given cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ApplicationInstallationListItem
//	  401: empty
//	  403: empty
func (r Routing) listApplicationInstallations() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(applicationinstallation.ListApplicationInstallations(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		applicationinstallation.DecodeListApplicationInstallations,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations applications createApplicationInstallation
//
//	Creates ApplicationInstallation into the given cluster
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: ApplicationInstallation
//	  401: empty
//	  403: empty
func (r Routing) createApplicationInstallation() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(applicationinstallation.CreateApplicationInstallation(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		applicationinstallation.DecodeCreateApplicationInstallation,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations/{namespace}/{appinstall_name} applications deleteApplicationInstallation
//
//	Deletes the given ApplicationInstallation
//
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: empty
//	   401: empty
//	   403: empty
func (r Routing) deleteApplicationInstallation() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(applicationinstallation.DeleteApplicationInstallation(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		applicationinstallation.DecodeDeleteApplicationInstallation,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations/{namespace}/{appinstall_name} applications getApplicationInstallation
//
//	Gets the given ApplicationInstallation
//
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: ApplicationInstallation
//	   401: empty
//	   403: empty
func (r Routing) getApplicationInstallation() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(applicationinstallation.GetApplicationInstallation(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		applicationinstallation.DecodeGetApplicationInstallation,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v2/projects/{project_id}/clusters/{cluster_id}/applicationinstallations/{namespace}/{appinstall_name} applications updateApplicationInstallation
//
//	Updates the given ApplicationInstallation
//
//
//	 Consumes:
//	 - application/json
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: ApplicationInstallation
//	   401: empty
//	   403: empty
func (r Routing) updateApplicationInstallation() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(applicationinstallation.UpdateApplicationInstallation(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		applicationinstallation.DecodeUpdateApplicationInstallation,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/applicationdefinitions applications listApplicationDefinitions
//
//	List ApplicationDefinitions which are available in the KKP installation
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ApplicationDefinitionListItem
//	  401: empty
//	  403: empty
func (r Routing) listApplicationDefinitions() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(applicationdefinition.ListApplicationDefinitions(r.applicationDefinitionProvider)),
		common.DecodeEmptyReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/applicationdefinitions/{appdef_name} applications getApplicationDefinition
//
//	Gets the given ApplicationDefinition
//
//
//	 Produces:
//	 - application/json
//
//	 Responses:
//	   default: errorResponse
//	   200: ApplicationDefinition
//	   401: empty
//	   403: empty
func (r Routing) getApplicationDefinition() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(applicationdefinition.GetApplicationDefinition(r.applicationDefinitionProvider)),
		applicationdefinition.DecodeGetApplicationDefinition,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/seeds/{seed_name}/ipampools ipampool listIPAMPools
//
//	Lists IPAM pools.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []IPAMPool
//	  401: empty
//	  403: empty
func (r Routing) listIPAMPools() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedIPAMPool(r.privilegedIPAMPoolProviderGetter, r.seedsGetter),
		)(ipampool.ListIPAMPoolsEndpoint(r.userInfoGetter)),
		ipampool.DecodeSeedReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/seeds/{seed_name}/ipampools/{ipampool_name} ipampool getIPAMPool
//
//	Gets a specific IPAM pool.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: IPAMPool
//	  401: empty
//	  403: empty
func (r Routing) getIPAMPool() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedIPAMPool(r.privilegedIPAMPoolProviderGetter, r.seedsGetter),
		)(ipampool.GetIPAMPoolEndpoint(r.userInfoGetter)),
		ipampool.DecodeIPAMPoolReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/seeds/{seed_name}/ipampools ipampool createIPAMPool
//
//	Creates a IPAM pool.
//
//	Consumes:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: empty
//	  401: empty
//	  403: empty
func (r Routing) createIPAMPool() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedIPAMPool(r.privilegedIPAMPoolProviderGetter, r.seedsGetter),
		)(ipampool.CreateIPAMPoolEndpoint(r.userInfoGetter)),
		ipampool.DecodeCreateIPAMPoolReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v2/seeds/{seed_name}/ipampools/{ipampool_name} ipampool patchIPAMPool
//
//	Patches a IPAM pool.
//
//	Consumes:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) patchIPAMPool() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedIPAMPool(r.privilegedIPAMPoolProviderGetter, r.seedsGetter),
		)(ipampool.PatchIPAMPoolEndpoint(r.userInfoGetter)),
		ipampool.DecodePatchIPAMPoolReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v2/seeds/{seed_name}/ipampools/{ipampool_name} ipampool deleteIPAMPool
//
//	Removes an existing IPAM pool.
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteIPAMPool() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedIPAMPool(r.privilegedIPAMPoolProviderGetter, r.seedsGetter),
		)(ipampool.DeleteIPAMPoolEndpoint(r.userInfoGetter)),
		ipampool.DecodeIPAMPoolReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/seeds/{seed_name}/operatingsystemprofiles operatingsystemprofile listOperatingSystemProfiles
//
//	Lists Operating System Profiles.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OperatingSystemProfile
//	  401: empty
//	  403: empty
func (r Routing) listOperatingSystemProfiles() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.PrivilegedOperatingSystemProfile(r.clusterProviderGetter, r.privilegedOperatingSystemProfileProviderGetter, r.seedsGetter),
		)(operatingsystemprofile.ListOperatingSystemProfilesEndpoint(r.userInfoGetter)),
		operatingsystemprofile.DecodeSeedReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /projects/{project_id}/clusters/{cluster_id}/operatingsystemprofiles operatingsystemprofile listOperatingSystemProfilesForCluster
//
//	Lists all available Operating System Profiles for a cluster
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []OperatingSystemProfile
//	  401: empty
//	  403: empty
func (r Routing) listOperatingSystemProfilesForCluster() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.PrivilegedOperatingSystemProfile(r.clusterProviderGetter, r.privilegedOperatingSystemProfileProviderGetter, r.seedsGetter),
		)(operatingsystemprofile.ListOperatingSystemProfilesEndpointForCluster(r.userInfoGetter)),
		operatingsystemprofile.DecodeListOperatingSystemProfiles,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/serviceaccount project listClusterServiceAccount
//
//	List service accounts in cluster.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  200: []ClusterServiceAccount
//	  401: empty
//	  403: empty
func (r Routing) listClusterServiceAccount() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.ListClusterSAEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		cluster.DecodeListClusterServiceAccount,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v2/projects/{project_id}/clusters/{cluster_id}/serviceaccount project createClusterServiceAccount
//
//	Creates a service account in cluster.
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Responses:
//	  default: errorResponse
//	  201: ClusterServiceAccount
//	  401: empty
//	  403: empty
func (r Routing) createClusterServiceAccount() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.CreateClusterSAEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		cluster.DecodeCreateClusterServiceAccount,
		handler.SetStatusCreatedHeader(handler.EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// getClusterServiceAccountKubeconfig returns the kubeconfig for the service account.
// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/serviceaccount/{namespace}/{service_account_id}/kubeconfig project getClusterServiceAccountKubeconfig
//
//	Gets the kubeconfig for the specified service account in cluster.
//
//	Produces:
//	- application/octet-stream
//
//	Responses:
//	  default: errorResponse
//	  200: Kubeconfig
//	  401: empty
//	  403: empty
func (r Routing) getClusterServiceAccountKubeconfig() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.GetClusterSAKubeconigEndpoint(r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)),
		cluster.DecodeClusterSAReq,
		cluster.EncodeKubeconfig,
		r.defaultServerOptions()...,
	)
}

// DeleteClusterServiceAccount delete the service account.
// swagger:route DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/serviceaccount/{namespace}/{service_account_id} project deleteClusterServiceAccount
//
//	Deletes service account in cluster.
//
//	Responses:
//	  default: errorResponse
//	  200: empty
//	  401: empty
//	  403: empty
func (r Routing) deleteClusterServiceAccount() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
		)(cluster.DeleteClusterSAKubeconigEndpoint(r.userInfoGetter, r.projectProvider, r.privilegedProjectProvider)),
		cluster.DecodeClusterSAReq,
		handler.EncodeJSON,
		r.defaultServerOptions()...,
	)
}
