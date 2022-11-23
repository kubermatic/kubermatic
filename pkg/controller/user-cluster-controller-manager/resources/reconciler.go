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

package resources

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"strings"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/applications"
	cabundle "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/ca-bundle"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/cloudinitsettings"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/clusterautoscaler"
	controllermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/controller-manager"
	coredns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/core-dns"
	csimigration "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/csi-migration"
	csisnapshotter "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/csi-snapshotter"
	dnatcontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/dnat-controller"
	envoyagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/envoy-agent"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/gatekeeper"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/konnectivity"
	kubestatemetrics "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kube-state-metrics"
	kubernetesresources "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubernetes"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubernetes-dashboard"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubesystem"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine"
	machinecontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine-controller"
	metricsserver "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/metrics-server"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla"
	userclustermonitoringagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla/monitoring-agent"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla/promtail"
	nodelocaldns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/node-local-dns"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/openvpn"
	operatingsystemmanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/operating-system-manager"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/prometheus"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/scheduler"
	systembasicuser "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/system-basic-user"
	userauth "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/user-auth"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/usersshkeys"
	"k8c.io/kubermatic/v2/pkg/crd"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) reconcile(ctx context.Context) error {
	caCert, err := r.caCert(ctx)
	if err != nil {
		return fmt.Errorf("failed to get caCert: %w", err)
	}
	userSSHKeys, err := r.userSSHKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get userSSHKeys: %w", err)
	}
	cloudConfig, err := r.cloudConfig(ctx, resources.CloudConfigSeedSecretName)
	if err != nil {
		return fmt.Errorf("failed to get cloudConfig: %w", err)
	}

	data := reconcileData{
		caCert:       caCert,
		userSSHKeys:  userSSHKeys,
		cloudConfig:  cloudConfig,
		ccmMigration: r.ccmMigration || r.ccmMigrationCompleted,
	}

	if r.cloudProvider == kubermaticv1.VSphereCloudProvider || r.cloudProvider == kubermaticv1.VMwareCloudDirectorCloudProvider || (r.cloudProvider == kubermaticv1.NutanixCloudProvider && r.nutanixCSIEnabled) {
		data.csiCloudConfig, err = r.cloudConfig(ctx, resources.CSICloudConfigSecretName)
		if err != nil {
			return fmt.Errorf("failed to get csi config: %w", err)
		}
	}

	data.clusterAddress, data.ipFamily, data.k8sServiceApiIP, data.reconcileK8sSvcEndpoints, data.coreDNSReplicas, err = r.networkingData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster address: %w", err)
	}

	if !r.isKonnectivityEnabled {
		data.openVPNCACert, err = r.openVPNCA(ctx)
		if err != nil {
			return fmt.Errorf("failed to get openVPN CA cert: %w", err)
		}
	}

	if r.userClusterMLA.Monitoring || r.userClusterMLA.Logging {
		data.mlaGatewayCACert, err = r.mlaGatewayCA(ctx)
		if err != nil {
			return fmt.Errorf("failed to get MLA Gateway CA cert: %w", err)
		}
		data.monitoringRequirements, data.loggingRequirements, data.monitoringReplicas, err = r.mlaReconcileData(ctx)
		if err != nil {
			return fmt.Errorf("failed to get MLA resource requirements: %w", err)
		}
	}

	if r.opaIntegration {
		data.gatekeeperCtrlRequirements, data.gatekeeperAuditRequirements, err = r.opaReconcileData(ctx)
		if err != nil {
			return fmt.Errorf("failed to get OPA resource requirements: %w", err)
		}
	}

	cluster, err := r.getCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve cluster: %w", err)
	}

	data.operatingSystemManagerEnabled = cluster.Spec.IsOperatingSystemManagerEnabled()
	data.kubernetesDashboardEnabled = cluster.Spec.IsKubernetesDashboardEnabled()

	// Must be first because of openshift
	if err := r.ensureAPIServices(ctx, data); err != nil {
		return err
	}

	// We need to reconcile namespaces and services next to make sure
	// the openshift apiservices become available ASAP
	if err := r.reconcileNamespaces(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileServiceAccounts(ctx, data); err != nil {
		return err
	}

	if err := r.reconcilePodDisruptionBudgets(ctx); err != nil {
		return err
	}

	if err := r.reconcileDeployments(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileServices(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileEndpoints(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileClusterRoles(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileRoles(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileRoleBindings(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileCRDs(ctx); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileSecrets(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileDaemonSet(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileValidatingWebhookConfigurations(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileMutatingWebhookConfigurations(ctx, data); err != nil {
		return err
	}

	if r.networkPolices {
		if err := r.reconcileNetworkPolicies(ctx, data); err != nil {
			return err
		}
	}

	// Try to delete OPA integration deployment if its present
	if !r.opaIntegration {
		if err := r.ensureOPAIntegrationIsRemoved(ctx); err != nil {
			return err
		}
	}

	if r.opaIntegration && !r.opaEnableMutation {
		if err := r.ensureOPAExperimentalMutationWebhookIsRemoved(ctx); err != nil {
			return err
		}
	}

	if !r.userClusterMLA.Logging {
		if err := r.ensurePromtailIsRemoved(ctx); err != nil {
			return err
		}
	}
	if !r.userClusterMLA.Monitoring {
		if err := r.ensureUserClusterMonitoringAgentIsRemoved(ctx); err != nil {
			return err
		}
	} else {
		// remove legacy prometheus installation in user cluster
		if err := r.ensureLegacyPrometheusIsRemoved(ctx); err != nil {
			return err
		}
	}

	if r.opaIntegration || r.userClusterMLA.Logging || r.userClusterMLA.Monitoring {
		if err := r.healthCheck(ctx); err != nil {
			return err
		}
	}

	if !r.userClusterMLA.Logging && !r.userClusterMLA.Monitoring {
		if err := r.ensureMLAIsRemoved(ctx); err != nil {
			return err
		}
	}

	// This code supports switching between OpenVPN and Konnectivity setup (in both directions).
	// It can be removed one release after deprecating OpenVPN.
	if r.isKonnectivityEnabled {
		if err := r.ensureOpenVPNSetupIsRemoved(ctx); err != nil {
			return err
		}
	} else {
		if err := r.ensureKonnectivitySetupIsRemoved(ctx); err != nil {
			return err
		}
	}

	if !data.operatingSystemManagerEnabled {
		if err := r.ensureOSMResourcesAreRemoved(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) ensureAPIServices(ctx context.Context, data reconcileData) error {
	caCert := triple.EncodeCertPEM(data.caCert.Cert)
	creators := []reconciling.NamedAPIServiceCreatorGetter{
		metricsserver.APIServiceCreator(caCert),
	}

	if err := reconciling.ReconcileAPIServices(ctx, creators, metav1.NamespaceNone, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile APIServices: %w", err)
	}

	return nil
}

func (r *reconciler) reconcileServiceAccounts(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedServiceAccountCreatorGetter{
		userauth.ServiceAccountCreator(),
		usersshkeys.ServiceAccountCreator(),
		coredns.ServiceAccountCreator(),
	}

	if r.nodeLocalDNSCache {
		creators = append(creators, nodelocaldns.ServiceAccountCreator())
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.ServiceAccountCreator())
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", metav1.NamespaceSystem, err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedServiceAccountCreatorGetter{
			kubernetesdashboard.ServiceAccountCreator(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	cloudInitSACreator := []reconciling.NamedServiceAccountCreatorGetter{
		cloudinitsettings.ServiceAccountCreator(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, cloudInitSACreator, resources.CloudInitSettingsNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter in the namespace %s: %w", resources.CloudInitSettingsNamespace, err)
	}

	// OPA related resources
	if r.opaIntegration {
		creators = []reconciling.NamedServiceAccountCreatorGetter{
			gatekeeper.ServiceAccountCreator(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	if r.isKonnectivityEnabled {
		creators = []reconciling.NamedServiceAccountCreatorGetter{
			konnectivity.ServiceAccountCreator(),
			metricsserver.ServiceAccountCreator(), // required only if metrics-server is running in user cluster
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", metav1.NamespaceSystem, err)
		}
	}

	creators = []reconciling.NamedServiceAccountCreatorGetter{}
	if r.userClusterMLA.Logging {
		creators = append(creators,
			promtail.ServiceAccountCreator(),
		)
	}
	if r.userClusterMLA.Monitoring {
		creators = append(creators,
			userclustermonitoringagent.ServiceAccountCreator(),
		)
	}

	if len(creators) != 0 {
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileRoles(ctx context.Context, data reconcileData) error {
	// kube-system
	creators := []reconciling.NamedRoleCreatorGetter{
		machinecontroller.KubeSystemRoleCreator(),
		clusterautoscaler.KubeSystemRoleCreator(),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.RoleCreator())
	}

	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.KubeSystemRoleCreator())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", metav1.NamespaceSystem, err)
	}

	// kube-public
	creators = []reconciling.NamedRoleCreatorGetter{
		machinecontroller.ClusterInfoReaderRoleCreator(),
		machinecontroller.KubePublicRoleCreator(),
	}

	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.KubePublicRoleCreator())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespacePublic, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", metav1.NamespacePublic, err)
	}

	// default
	creators = []reconciling.NamedRoleCreatorGetter{
		machinecontroller.EndpointReaderRoleCreator(),
		clusterautoscaler.DefaultRoleCreator(),
	}

	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.DefaultRoleCreator())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespaceDefault, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", metav1.NamespaceDefault, err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedRoleCreatorGetter{
			kubernetesdashboard.RoleCreator(),
		}

		if err := reconciling.ReconcileRoles(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	cloudInitRoleCreator := []reconciling.NamedRoleCreatorGetter{
		cloudinitsettings.RoleCreator(),
	}

	if data.operatingSystemManagerEnabled {
		cloudInitRoleCreator = append(cloudInitRoleCreator, operatingsystemmanager.CloudInitSettingsRoleCreator())
	}

	if err := reconciling.ReconcileRoles(ctx, cloudInitRoleCreator, resources.CloudInitSettingsNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter role in the namespace %s: %w", resources.CloudInitSettingsNamespace, err)
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedRoleCreatorGetter{
			gatekeeper.RoleCreator(),
		}
		if err := reconciling.ReconcileRoles(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileRoleBindings(ctx context.Context, data reconcileData) error {
	// kube-system
	creators := []reconciling.NamedRoleBindingCreatorGetter{
		machinecontroller.KubeSystemRoleBindingCreator(),
		metricsserver.RolebindingAuthReaderCreator(r.isKonnectivityEnabled),
		scheduler.RoleBindingAuthDelegator(),
		controllermanager.RoleBindingAuthDelegator(),
		clusterautoscaler.KubeSystemRoleBindingCreator(),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.RoleBindingCreator())
	}

	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.KubeSystemRoleBindingCreator())
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in kube-system Namespace: %w", err)
	}

	// kube-public
	creators = []reconciling.NamedRoleBindingCreatorGetter{
		machinecontroller.KubePublicRoleBindingCreator(),
		machinecontroller.ClusterInfoAnonymousRoleBindingCreator(),
	}
	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.KubePublicRoleBindingCreator())
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespacePublic, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in kube-public Namespace: %w", err)
	}

	// Default
	creators = []reconciling.NamedRoleBindingCreatorGetter{
		machinecontroller.DefaultRoleBindingCreator(),
		clusterautoscaler.DefaultRoleBindingCreator(),
	}
	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.DefaultRoleBindingCreator())
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespaceDefault, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in default Namespace: %w", err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedRoleBindingCreatorGetter{
			kubernetesdashboard.RoleBindingCreator(),
		}
		if err := reconciling.ReconcileRoleBindings(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile RoleBindings in the namespace: %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	cloudInitRoleBindingCreator := []reconciling.NamedRoleBindingCreatorGetter{
		cloudinitsettings.RoleBindingCreator(),
	}

	if data.operatingSystemManagerEnabled {
		cloudInitRoleBindingCreator = append(cloudInitRoleBindingCreator, operatingsystemmanager.CloudInitSettingsRoleBindingCreator())
	}

	if err := reconciling.ReconcileRoleBindings(ctx, cloudInitRoleBindingCreator, resources.CloudInitSettingsNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter RoleBindings in the namespace: %s: %w", resources.CloudInitSettingsNamespace, err)
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedRoleBindingCreatorGetter{
			gatekeeper.RoleBindingCreator(),
		}
		if err := reconciling.ReconcileRoleBindings(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile RoleBindings in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileClusterRoles(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedClusterRoleCreatorGetter{
		kubestatemetrics.ClusterRoleCreator(),
		prometheus.ClusterRoleCreator(),
		machinecontroller.ClusterRoleCreator(),
		dnatcontroller.ClusterRoleCreator(),
		metricsserver.ClusterRoleCreator(),
		clusterautoscaler.ClusterRoleCreator(),
		coredns.ClusterRoleCreator(),
	}

	if data.kubernetesDashboardEnabled {
		creators = append(creators, kubernetesdashboard.ClusterRoleCreator())
	}

	if r.opaIntegration {
		creators = append(creators, gatekeeper.ClusterRoleCreator())
	}

	if r.userClusterMLA.Logging {
		creators = append(creators, promtail.ClusterRoleCreator())
	}
	if r.userClusterMLA.Monitoring {
		creators = append(creators, userclustermonitoringagent.ClusterRoleCreator())
	}

	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.MachineDeploymentsClusterRoleCreator())
		creators = append(creators, operatingsystemmanager.WebhookClusterRoleCreator())
	}

	if err := reconciling.ReconcileClusterRoles(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileClusterRoleBindings(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		userauth.ClusterRoleBindingCreator(),
		kubestatemetrics.ClusterRoleBindingCreator(),
		prometheus.ClusterRoleBindingCreator(),
		machinecontroller.ClusterRoleBindingCreator(),
		machinecontroller.NodeBootstrapperClusterRoleBindingCreator(),
		machinecontroller.NodeSignerClusterRoleBindingCreator(),
		dnatcontroller.ClusterRoleBindingCreator(),
		metricsserver.ClusterRoleBindingResourceReaderCreator(r.isKonnectivityEnabled),
		metricsserver.ClusterRoleBindingAuthDelegatorCreator(r.isKonnectivityEnabled),
		scheduler.ClusterRoleBindingAuthDelegatorCreator(),
		controllermanager.ClusterRoleBindingAuthDelegator(),
		clusterautoscaler.ClusterRoleBindingCreator(),
		systembasicuser.ClusterRoleBinding,
		cloudcontroller.ClusterRoleBindingCreator(),
		coredns.ClusterRoleBindingCreator(),
	}

	if data.kubernetesDashboardEnabled {
		creators = append(creators, kubernetesdashboard.ClusterRoleBindingCreator())
	}

	if r.opaIntegration {
		creators = append(creators, gatekeeper.ClusterRoleBindingCreator())
	}

	if r.userClusterMLA.Logging {
		creators = append(creators, promtail.ClusterRoleBindingCreator())
	}

	if r.userClusterMLA.Monitoring {
		creators = append(creators, userclustermonitoringagent.ClusterRoleBindingCreator())
	}

	if r.isKonnectivityEnabled {
		creators = append(creators, konnectivity.ClusterRoleBindingCreator())
	}

	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.MachineDeploymentsClusterRoleBindingCreator())
		creators = append(creators, operatingsystemmanager.WebhookClusterRoleBindingCreator())
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileCRDs(ctx context.Context) error {
	c, err := crd.CRDForObject(&appskubermaticv1.ApplicationInstallation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
			Kind:       "ApplicationInstallation",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get ApplicationInstallation CRD: %w", err)
	}

	creators := []reconciling.NamedCustomResourceDefinitionCreatorGetter{
		machinecontroller.MachineCRDCreator(),
		machinecontroller.MachineSetCRDCreator(),
		machinecontroller.MachineDeploymentCRDCreator(),
		applications.CRDCreator(c),
	}

	if r.opaIntegration {
		creators = append(creators,
			gatekeeper.ConfigCRDCreator(),
			gatekeeper.ConstraintTemplateCRDCreator(),
			gatekeeper.ConstraintPodStatusCRDCreator(),
			gatekeeper.ConstraintTemplatePodStatusCRDCreator(),
			gatekeeper.MutatorPodStatusCRDCreator(),
			gatekeeper.AssignCRDCreator(),
			gatekeeper.AssignMetadataCRDCreator(),
			gatekeeper.ModifySetCRDCreator(),
			gatekeeper.ProviderCRDCreator())
	}

	if err := reconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile CustomResourceDefinitions: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileMutatingWebhookConfigurations(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedMutatingWebhookConfigurationCreatorGetter{
		machinecontroller.MutatingwebhookConfigurationCreator(data.caCert.Cert, r.namespace),
	}
	if r.opaIntegration && r.opaEnableMutation {
		creators = append(creators, gatekeeper.MutatingWebhookConfigurationCreator(r.opaWebhookTimeout))
	}
	if data.operatingSystemManagerEnabled {
		creators = append(creators, operatingsystemmanager.MutatingwebhookConfigurationCreator(data.caCert.Cert, r.namespace))
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile MutatingWebhookConfigurations: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileValidatingWebhookConfigurations(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{
		applications.ApplicationInstallationValidatingWebhookConfigurationCreator(data.caCert.Cert, r.namespace),
		machine.ValidatingWebhookConfigurationCreator(data.caCert.Cert, r.namespace),
	}
	if r.opaIntegration {
		creators = append(creators, gatekeeper.ValidatingWebhookConfigurationCreator(r.opaWebhookTimeout))
	}

	if data.ccmMigration && data.csiCloudConfig != nil {
		creators = append(creators, csimigration.ValidatingwebhookConfigurationCreator(data.caCert.Cert, metav1.NamespaceSystem, resources.VsphereCSIMigrationWebhookConfigurationWebhookName))
	}

	if r.cloudProvider == kubermaticv1.VSphereCloudProvider || r.cloudProvider == kubermaticv1.NutanixCloudProvider || r.cloudProvider == kubermaticv1.OpenstackCloudProvider ||
		r.cloudProvider == kubermaticv1.DigitaloceanCloudProvider {
		creators = append(creators, csisnapshotter.ValidatingSnapshotWebhookConfigurationCreator(data.caCert.Cert, metav1.NamespaceSystem, resources.CSISnapshotValidationWebhookConfigurationName))
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingWebhookConfigurations: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileServices(ctx context.Context, data reconcileData) error {
	creatorsKubeSystem := []reconciling.NamedServiceCreatorGetter{
		coredns.ServiceCreator(r.dnsClusterIP),
	}
	if r.isKonnectivityEnabled {
		// metrics-server running in user cluster - ClusterIP service
		creatorsKubeSystem = append(creatorsKubeSystem, metricsserver.ServiceCreator(data.ipFamily))
	} else {
		// metrics-server running in seed cluster - ExternalName service
		creatorsKubeSystem = append(creatorsKubeSystem, metricsserver.ExternalNameServiceCreator(r.namespace))
	}

	if err := reconciling.ReconcileServices(ctx, creatorsKubeSystem, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services in kube-system namespace: %w", err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators := []reconciling.NamedServiceCreatorGetter{
			kubernetesdashboard.ServiceCreator(data.ipFamily),
		}
		if err := reconciling.ReconcileServices(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Services in namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	// OPA related resources
	if r.opaIntegration {
		creators := []reconciling.NamedServiceCreatorGetter{
			gatekeeper.ServiceCreator(),
		}
		if err := reconciling.ReconcileServices(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Services in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileEndpoints(ctx context.Context, data reconcileData) error {
	if !data.reconcileK8sSvcEndpoints {
		return nil
	}
	epCreators := []reconciling.NamedEndpointsCreatorGetter{
		kubernetesresources.EndpointsCreator(data.clusterAddress),
	}
	if err := reconciling.ReconcileEndpoints(ctx, epCreators, metav1.NamespaceDefault, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Endpoints: %w", err)
	}
	epSliceCreators := []reconciling.NamedEndpointSliceCreatorGetter{
		kubernetesresources.EndpointSliceCreator(data.clusterAddress),
	}
	if err := reconciling.ReconcileEndpointSlices(ctx, epSliceCreators, metav1.NamespaceDefault, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile EndpointSlices: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileConfigMaps(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		machinecontroller.ClusterInfoConfigMapCreator(r.clusterURL.String(), data.caCert.Cert),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespacePublic, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-public namespace: %w", err)
	}

	if len(r.tunnelingAgentIP) > 0 {
		envoyConfig := envoyagent.Config{
			AdminPort: 9902,
			ProxyHost: r.clusterURL.Hostname(),
			ProxyPort: 8088,
			Listeners: []envoyagent.Listener{
				{
					BindAddress: r.tunnelingAgentIP.String(),
					BindPort:    r.kasSecurePort,
					Authority:   net.JoinHostPort(fmt.Sprintf("apiserver-external.%s.svc.cluster.local", r.namespace), "443"),
				},
			},
		}
		if !r.isKonnectivityEnabled {
			// add OpenVPN server port listener if Konnectivity is NOT enabled
			envoyConfig.Listeners = append(envoyConfig.Listeners,
				envoyagent.Listener{
					BindAddress: r.tunnelingAgentIP.String(),
					BindPort:    r.openvpnServerPort,
					Authority:   net.JoinHostPort(fmt.Sprintf("openvpn-server.%s.svc.cluster.local", r.namespace), "1194"),
				})
		}
		creators = []reconciling.NamedConfigMapCreatorGetter{
			cabundle.ConfigMapCreator(r.caBundle),
			envoyagent.ConfigMapCreator(envoyConfig),
		}
		if !r.isKonnectivityEnabled {
			creators = append(creators, openvpn.ClientConfigConfigMapCreator(r.tunnelingAgentIP.String(), r.openvpnServerPort))
		}
	} else {
		creators = []reconciling.NamedConfigMapCreatorGetter{
			cabundle.ConfigMapCreator(r.caBundle),
		}
		if !r.isKonnectivityEnabled {
			creators = append(creators, openvpn.ClientConfigConfigMapCreator(r.clusterURL.Hostname(), r.openvpnServerPort))
		}
	}

	creators = append(creators, coredns.ConfigMapCreator())

	if r.nodeLocalDNSCache {
		creators = append(creators, nodelocaldns.ConfigMapCreator(r.dnsClusterIP))
	}

	if data.csiCloudConfig != nil {
		if r.cloudProvider == kubermaticv1.VMwareCloudDirectorCloudProvider {
			creators = append(creators, cloudcontroller.VMwareCloudDirectorCSIConfig(data.csiCloudConfig))
		}
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-system namespace: %w", err)
	}

	if r.userClusterMLA.Monitoring {
		customScrapeConfigs, err := r.getUserClusterMonitoringAgentCustomScrapeConfigs(ctx)
		if err != nil {
			return fmt.Errorf("failed to get user cluster prometheus custom scrape configs: %w", err)
		}
		creators = []reconciling.NamedConfigMapCreatorGetter{
			userclustermonitoringagent.ConfigMapCreator(userclustermonitoringagent.Config{
				MLAGatewayURL:       r.userClusterMLA.MLAGatewayURL + "/api/v1/push",
				TLSCertFile:         fmt.Sprintf("%s/%s", resources.UserClusterMonitoringAgentClientCertMountPath, resources.UserClusterMonitoringAgentClientCertSecretKey),
				TLSKeyFile:          fmt.Sprintf("%s/%s", resources.UserClusterMonitoringAgentClientCertMountPath, resources.UserClusterMonitoringAgentClientKeySecretKey),
				TLSCACertFile:       fmt.Sprintf("%s/%s", resources.UserClusterMonitoringAgentClientCertMountPath, resources.MLAGatewayCACertKey),
				CustomScrapeConfigs: customScrapeConfigs,
				HAClusterIdentifier: r.clusterName,
			}),
		}
		if err := reconciling.ReconcileConfigMaps(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ConfigMap in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}
	return nil
}

func (r *reconciler) reconcileSecrets(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedSecretCreatorGetter{
		cloudcontroller.CloudConfig(data.cloudConfig, resources.CloudConfigSecretName),
	}
	if !r.isKonnectivityEnabled {
		creators = append(creators, openvpn.ClientCertificate(data.openVPNCACert))
	} else {
		// required only if metrics-server is running in user cluster
		creators = append(creators, metricsserver.TLSServingCertSecretCreator(
			func() (*triple.KeyPair, error) {
				return data.caCert, nil
			}),
		)
	}

	if data.csiCloudConfig != nil {
		if r.cloudProvider == kubermaticv1.VSphereCloudProvider {
			creators = append(creators, cloudcontroller.CloudConfig(data.csiCloudConfig, resources.CSICloudConfigSecretName),
				csisnapshotter.TLSServingCertificateCreator(resources.CSISnapshotValidationWebhookName, data.caCert))
			if data.ccmMigration {
				creators = append(creators, csimigration.TLSServingCertificateCreator(data.caCert))
			}
		}

		if r.cloudProvider == kubermaticv1.NutanixCloudProvider {
			creators = append(creators, cloudcontroller.NutanixCSIConfig(data.csiCloudConfig),
				csisnapshotter.TLSServingCertificateCreator(resources.CSISnapshotValidationWebhookName, data.caCert))
		}
	}

	if r.cloudProvider == kubermaticv1.OpenstackCloudProvider || r.cloudProvider == kubermaticv1.DigitaloceanCloudProvider {
		creators = append(creators, csisnapshotter.TLSServingCertificateCreator(resources.CSISnapshotValidationWebhookName, data.caCert))
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.SecretCreator(data.userSSHKeys))
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in kube-system Namespace: %w", err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedSecretCreatorGetter{
			kubernetesdashboard.KeyHolderSecretCreator(),
			kubernetesdashboard.CsrfTokenSecretCreator(),
		}

		if err := reconciling.ReconcileSecrets(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedSecretCreatorGetter{
			gatekeeper.SecretCreator(),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	if r.userClusterMLA.Monitoring {
		creators = []reconciling.NamedSecretCreatorGetter{
			userclustermonitoringagent.ClientCertificateCreator(data.mlaGatewayCACert),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}
	if r.userClusterMLA.Logging {
		creators = []reconciling.NamedSecretCreatorGetter{
			promtail.SecretCreator(promtail.Config{
				MLAGatewayURL: r.userClusterMLA.MLAGatewayURL + "/loki/api/v1/push",
				TLSCertFile:   fmt.Sprintf("%s/%s", resources.PromtailClientCertMountPath, resources.PromtailClientCertSecretKey),
				TLSKeyFile:    fmt.Sprintf("%s/%s", resources.PromtailClientCertMountPath, resources.PromtailClientKeySecretKey),
				TLSCACertFile: fmt.Sprintf("%s/%s", resources.PromtailClientCertMountPath, resources.MLAGatewayCACertKey),
			}),
			promtail.ClientCertificateCreator(data.mlaGatewayCACert),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}

	// Operating System Manager
	if data.operatingSystemManagerEnabled {
		creators = []reconciling.NamedSecretCreatorGetter{
			cloudinitsettings.SecretCreator(),
		}

		if err := reconciling.ReconcileSecrets(ctx, creators, resources.CloudInitSettingsNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.CloudInitSettingsNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileDaemonSet(ctx context.Context, data reconcileData) error {
	var dsCreators []reconciling.NamedDaemonSetCreatorGetter

	if r.nodeLocalDNSCache {
		dsCreators = append(dsCreators, nodelocaldns.DaemonSetCreator(r.imageRewriter))
	}

	if r.userSSHKeyAgent {
		dsCreators = append(dsCreators, usersshkeys.DaemonSetCreator(r.versions, r.imageRewriter))
	}

	if len(r.tunnelingAgentIP) > 0 {
		configHash, err := r.getEnvoyAgentConfigHash(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve envoy-agent config hash: %w", err)
		}
		dsCreators = append(dsCreators, envoyagent.DaemonSetCreator(r.tunnelingAgentIP, r.versions, configHash, r.imageRewriter))
	}

	if err := reconciling.ReconcileDaemonSets(ctx, dsCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the DaemonSet: %w", err)
	}

	if r.userClusterMLA.Logging {
		dsCreators = []reconciling.NamedDaemonSetCreatorGetter{
			promtail.DaemonSetCreator(data.loggingRequirements, r.imageRewriter),
		}
		if err := reconciling.ReconcileDaemonSets(ctx, dsCreators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile the DaemonSet: %w", err)
		}
	}
	return nil
}

func (r *reconciler) reconcileNamespaces(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedNamespaceCreatorGetter{
		cloudinitsettings.NamespaceCreator,
	}
	if data.kubernetesDashboardEnabled {
		creators = append(creators, kubernetesdashboard.NamespaceCreator)
	}

	if r.opaIntegration {
		creators = append(creators, gatekeeper.NamespaceCreator)
		creators = append(creators, gatekeeper.KubeSystemLabeler)
	}
	if r.userClusterMLA.Logging || r.userClusterMLA.Monitoring {
		creators = append(creators, mla.NamespaceCreator)
	}

	if err := reconciling.ReconcileNamespaces(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile namespaces: %w", err)
	}

	// update default serviceAccount for each created namespace
	for _, creator := range creators {
		namespace, _ := creator()
		err := r.reconcileDefaultServiceAccount(ctx, namespace)

		if err != nil {
			return fmt.Errorf("failed to update default service account: %w", err)
		}
	}

	// finally, ensure kube-system default service account is updated as well
	err := r.reconcileDefaultServiceAccount(ctx, resources.KubeSystemNamespaceName)
	if err != nil {
		return fmt.Errorf("failed to update default service account: %w", err)
	}

	return nil
}

func (r *reconciler) reconcileDeployments(ctx context.Context, data reconcileData) error {
	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators := []reconciling.NamedDeploymentCreatorGetter{
			kubernetesdashboard.DeploymentCreator(r.imageRewriter),
		}
		if err := reconciling.ReconcileDeployments(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	kubeSystemCreators := []reconciling.NamedDeploymentCreatorGetter{
		coredns.DeploymentCreator(r.clusterSemVer, data.coreDNSReplicas, r.imageRewriter),
	}

	if err := reconciling.ReconcileDeployments(ctx, kubeSystemCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", metav1.NamespaceSystem, err)
	}

	// OPA related resources
	if r.opaIntegration {
		creators := []reconciling.NamedDeploymentCreatorGetter{
			gatekeeper.ControllerDeploymentCreator(r.opaEnableMutation, r.imageRewriter, data.gatekeeperCtrlRequirements),
			gatekeeper.AuditDeploymentCreator(r.imageRewriter, data.gatekeeperAuditRequirements),
		}

		if err := reconciling.ReconcileDeployments(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	if r.userClusterMLA.Monitoring {
		creators := []reconciling.NamedDeploymentCreatorGetter{
			userclustermonitoringagent.DeploymentCreator(data.monitoringRequirements, data.monitoringReplicas, r.imageRewriter),
		}
		if err := reconciling.ReconcileDeployments(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}

	if r.isKonnectivityEnabled {
		creators := []reconciling.NamedDeploymentCreatorGetter{
			konnectivity.DeploymentCreator(r.konnectivityServerHost, r.konnectivityServerPort, r.imageRewriter),
			metricsserver.DeploymentCreator(r.imageRewriter), // deploy metrics-server in user cluster
		}
		if err := reconciling.ReconcileDeployments(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", metav1.NamespaceSystem, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileNetworkPolicies(ctx context.Context, data reconcileData) error {
	namedNetworkPolicyCreatorGetters := []reconciling.NamedNetworkPolicyCreatorGetter{
		kubesystem.DefaultNetworkPolicyCreator(),
		coredns.KubeDNSNetworkPolicyCreator(data.clusterAddress.IP, int(data.clusterAddress.Port), data.k8sServiceApiIP.String()),
	}

	if r.userSSHKeyAgent {
		namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters,
			usersshkeys.NetworkPolicyCreator(data.clusterAddress.IP, int(data.clusterAddress.Port), data.k8sServiceApiIP.String()))
	}

	if r.isKonnectivityEnabled {
		namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters, metricsserver.NetworkPolicyCreator(), konnectivity.NetworkPolicyCreator())
	}

	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyCreatorGetters, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to ensure Network Policies: %w", err)
	}

	return nil
}

func (r *reconciler) reconcilePodDisruptionBudgets(ctx context.Context) error {
	creators := []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		coredns.PodDisruptionBudgetCreator(),
	}
	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedPodDisruptionBudgetCreatorGetter{
			gatekeeper.PodDisruptionBudgetCreator(),
		}
		if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile PodDisruptionBudgets in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}
	if r.isKonnectivityEnabled {
		creators = append(creators,
			konnectivity.PodDisruptionBudgetCreator(),
			metricsserver.PodDisruptionBudgetCreator(),
		)
	}
	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %w", err)
	}
	return nil
}

type reconcileData struct {
	caCert           *triple.KeyPair
	openVPNCACert    *resources.ECDSAKeyPair
	mlaGatewayCACert *resources.ECDSAKeyPair
	userSSHKeys      map[string][]byte
	cloudConfig      []byte
	// csiCloudConfig is currently used only by vSphere, VMware Cloud Director, and Nutanix, whose needs it to properly configure the external CSI driver
	csiCloudConfig                []byte
	ccmMigration                  bool
	monitoringRequirements        *corev1.ResourceRequirements
	loggingRequirements           *corev1.ResourceRequirements
	gatekeeperCtrlRequirements    *corev1.ResourceRequirements
	gatekeeperAuditRequirements   *corev1.ResourceRequirements
	monitoringReplicas            *int32
	clusterAddress                *kubermaticv1.ClusterAddress
	ipFamily                      kubermaticv1.IPFamily
	k8sServiceApiIP               *net.IP
	reconcileK8sSvcEndpoints      bool
	kubernetesDashboardEnabled    bool
	operatingSystemManagerEnabled bool
	coreDNSReplicas               *int32
}

func (r *reconciler) ensureOPAIntegrationIsRemoved(ctx context.Context) error {
	for _, resource := range gatekeeper.GetResourcesToRemoveOnDelete() {
		err := r.Client.Delete(ctx, resource)
		if errC := r.cleanUpOPAHealthStatus(ctx, err); errC != nil {
			return fmt.Errorf("failed to update OPA health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OPA integration is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureOPAExperimentalMutationWebhookIsRemoved(ctx context.Context) error {
	if err := r.Client.Delete(ctx, &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperMutatingWebhookConfigurationName,
		}}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to remove Mutation Webhook: %w", err)
	}
	return nil
}

func (r *reconciler) getCluster(ctx context.Context) (*kubermaticv1.Cluster, error) {
	cluster, err := kubernetes.ClusterFromNamespace(ctx, r.seedClient, r.namespace)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, fmt.Errorf("no cluster exists for namespace %q", r.namespace)
	}

	return cluster, nil
}

func (r *reconciler) healthCheck(ctx context.Context) error {
	cluster, err := r.getCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed getting cluster for cluster health check: %w", err)
	}

	var (
		ctrlGatekeeperHealth  kubermaticv1.HealthStatus
		auditGatekeeperHealth kubermaticv1.HealthStatus
		monitoringHealth      kubermaticv1.HealthStatus
		loggingHealth         kubermaticv1.HealthStatus
	)

	if r.opaIntegration {
		ctrlGatekeeperHealth, auditGatekeeperHealth, err = r.getGatekeeperHealth(ctx)
		if err != nil {
			return err
		}
	}

	if r.userClusterMLA.Monitoring {
		monitoringHealth, err = r.getMLAMonitoringHealth(ctx)
		if err != nil {
			return err
		}
	}

	if r.userClusterMLA.Logging {
		loggingHealth, err = r.getMLALoggingHealth(ctx)
		if err != nil {
			return err
		}
	}

	return helper.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		if r.opaIntegration {
			c.Status.ExtendedHealth.GatekeeperController = &ctrlGatekeeperHealth
			c.Status.ExtendedHealth.GatekeeperAudit = &auditGatekeeperHealth
		}

		if r.userClusterMLA.Monitoring {
			c.Status.ExtendedHealth.Monitoring = &monitoringHealth
		}

		if r.userClusterMLA.Logging {
			c.Status.ExtendedHealth.Logging = &loggingHealth
		}
	})
}

func (r *reconciler) getGatekeeperHealth(ctx context.Context) (ctlrHealth kubermaticv1.HealthStatus, auditHealth kubermaticv1.HealthStatus, err error) {
	ctlrHealth, err = resources.HealthyDeployment(ctx,
		r.Client,
		types.NamespacedName{Namespace: resources.GatekeeperNamespace, Name: resources.GatekeeperControllerDeploymentName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, kubermaticv1.HealthStatusDown,
			fmt.Errorf("failed to get dep health %s: %w", resources.GatekeeperControllerDeploymentName, err)
	}

	auditHealth, err = resources.HealthyDeployment(ctx,
		r.Client,
		types.NamespacedName{Namespace: resources.GatekeeperNamespace, Name: resources.GatekeeperAuditDeploymentName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, kubermaticv1.HealthStatusDown,
			fmt.Errorf("failed to get dep health %s: %w", resources.GatekeeperAuditDeploymentName, err)
	}
	return ctlrHealth, auditHealth, nil
}

func (r *reconciler) getMLAMonitoringHealth(ctx context.Context) (health kubermaticv1.HealthStatus, err error) {
	health, err = resources.HealthyDeployment(ctx,
		r.Client,
		types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.UserClusterMonitoringAgentDeploymentName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown,
			fmt.Errorf("failed to get dep health %s: %w", resources.UserClusterMonitoringAgentDeploymentName, err)
	}

	return health, nil
}

func (r *reconciler) getMLALoggingHealth(ctx context.Context) (kubermaticv1.HealthStatus, error) {
	loggingHealth, err := resources.HealthyDaemonSet(ctx,
		r.Client,
		types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.PromtailDaemonSetName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to get ds health %s: %w", resources.PromtailDaemonSetName, err)
	}
	return loggingHealth, nil
}

func (r *reconciler) ensurePromtailIsRemoved(ctx context.Context) error {
	for _, resource := range promtail.ResourcesOnDeletion() {
		err := r.Client.Delete(ctx, resource)
		if errC := r.cleanUpMLAHealthStatus(ctx, true, false, err); errC != nil {
			return fmt.Errorf("failed to update mla logging health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure promtail is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureUserClusterMonitoringAgentIsRemoved(ctx context.Context) error {
	for _, resource := range userclustermonitoringagent.ResourcesOnDeletion() {
		err := r.Client.Delete(ctx, resource)
		if errC := r.cleanUpMLAHealthStatus(ctx, false, true, err); errC != nil {
			return fmt.Errorf("failed to update mla monitoring health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure user cluster monitring agent is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureLegacyPrometheusIsRemoved(ctx context.Context) error {
	for _, resource := range userclustermonitoringagent.LegacyResourcesOnDeletion() {
		err := r.Client.Delete(ctx, resource)
		if errC := r.cleanUpMLAHealthStatus(ctx, false, true, err); errC != nil {
			return fmt.Errorf("failed to update mla monitoring health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure user cluster monitring agent is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureMLAIsRemoved(ctx context.Context) error {
	for _, resource := range mla.ResourcesOnDeletion() {
		err := r.Client.Delete(ctx, resource)
		if errC := r.cleanUpMLAHealthStatus(ctx, true, true, err); errC != nil {
			return fmt.Errorf("failed to update mla health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure mla is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureOpenVPNSetupIsRemoved(ctx context.Context) error {
	for _, resource := range openvpn.ResourcesForDeletion() {
		err := r.Client.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OpenVPN resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureKonnectivitySetupIsRemoved(ctx context.Context) error {
	for _, resource := range konnectivity.ResourcesForDeletion() {
		err := r.Client.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure Konnectivity resources are removed/not present: %w", err)
		}
	}
	for _, resource := range metricsserver.UserClusterResourcesForDeletion() {
		err := r.Client.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure metrics-server resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureOSMResourcesAreRemoved(ctx context.Context) error {
	for _, resource := range operatingsystemmanager.ResourcesForDeletion() {
		err := r.Client.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OSM resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) getUserClusterMonitoringAgentCustomScrapeConfigs(ctx context.Context) (string, error) {
	if r.userClusterMLA.MonitoringAgentScrapeConfigPrefix == "" {
		return "", nil
	}
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, ctrlruntimeclient.InNamespace(resources.UserClusterMLANamespace)); err != nil {
		return "", fmt.Errorf("failed to list the configmap: %w", err)
	}
	customScrapeConfigs := ""
	for _, configMap := range configMapList.Items {
		if !strings.HasPrefix(configMap.GetName(), r.userClusterMLA.MonitoringAgentScrapeConfigPrefix) {
			continue
		}
		for _, v := range configMap.Data {
			customScrapeConfigs += strings.TrimSpace(v) + "\n"
		}
	}
	return customScrapeConfigs, nil
}

func (r *reconciler) getEnvoyAgentConfigHash(ctx context.Context) (string, error) {
	cm := corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: resources.EnvoyAgentConfigMapName, Namespace: metav1.NamespaceSystem}, &cm)
	if err != nil {
		return "", fmt.Errorf("failed to get envoy-agent configmap: %w", err)
	}
	configHash := sha1.New()
	configHash.Write([]byte(cm.Data[resources.EnvoyAgentConfigFileName]))
	return fmt.Sprintf("%x", configHash.Sum(nil)), nil
}

func (r *reconciler) cleanUpOPAHealthStatus(ctx context.Context, errC error) error {
	cluster, err := r.getCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed getting cluster for cluster health check: %w", err)
	}

	down := kubermaticv1.HealthStatusDown

	// Ensure that health status in Cluster CR is removed
	return helper.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.GatekeeperAudit = nil
		c.Status.ExtendedHealth.GatekeeperController = nil
		if errC != nil && !apierrors.IsNotFound(errC) {
			c.Status.ExtendedHealth.GatekeeperAudit = &down
			c.Status.ExtendedHealth.GatekeeperController = &down
		}
	})
}

func (r *reconciler) cleanUpMLAHealthStatus(ctx context.Context, logging, monitoring bool, errC error) error {
	cluster, err := r.getCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed getting cluster for cluster health check: %w", err)
	}

	down := kubermaticv1.HealthStatusDown

	// Ensure that health status in Cluster CR is removed
	return helper.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		if !r.userClusterMLA.Logging && logging {
			c.Status.ExtendedHealth.Logging = nil
			if errC != nil && !apierrors.IsNotFound(errC) {
				c.Status.ExtendedHealth.Logging = &down
			}
		}

		if !r.userClusterMLA.Monitoring && monitoring {
			c.Status.ExtendedHealth.Monitoring = nil
			if errC != nil && !apierrors.IsNotFound(errC) {
				c.Status.ExtendedHealth.Monitoring = &down
			}
		}
	})
}
