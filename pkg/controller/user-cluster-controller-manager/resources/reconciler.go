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

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/applications"
	cabundle "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/ca-bundle"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/cloudinitsettings"
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
	mlaloggingagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla/logging-agent"
	mlamonitoringagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla/monitoring-agent"
	nodelocaldns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/node-local-dns"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/openvpn"
	operatingsystemmanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/operating-system-manager"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/prometheus"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/scheduler"
	systembasicuser "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/system-basic-user"
	userauth "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/user-auth"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/usersshkeys"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/crd"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/reconciler/pkg/reconciling"

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

	cluster, err := r.getCluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve cluster: %w", err)
	}

	data := reconcileData{
		caCert:       caCert,
		userSSHKeys:  userSSHKeys,
		ccmMigration: r.ccmMigration || r.ccmMigrationCompleted,
		cluster:      cluster,
	}

	if !cluster.Spec.DisableCSIDriver {
		if r.cloudProvider == kubermaticv1.VSphereCloudProvider ||
			r.cloudProvider == kubermaticv1.VMwareCloudDirectorCloudProvider ||
			(r.cloudProvider == kubermaticv1.NutanixCloudProvider && r.nutanixCSIEnabled) {
			data.csiCloudConfig, err = r.cloudConfig(ctx, resources.CSICloudConfigSecretName)
			if err != nil {
				return fmt.Errorf("failed to get csi config: %w", err)
			}
		} else if r.cloudProvider == kubermaticv1.AzureCloudProvider ||
			r.cloudProvider == kubermaticv1.OpenstackCloudProvider {
			// Azure and Openstack CSI drivers don't have dedicated CSI cloud config.
			data.csiCloudConfig, err = r.cloudConfig(ctx, resources.CloudConfigSecretName)
			if err != nil {
				return fmt.Errorf("failed to get csi config: %w", err)
			}
		}
	}

	err = r.setupNetworkingData(cluster, &data)
	if err != nil {
		return fmt.Errorf("failed to setup cluster networking data: %w", err)
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

	clusterVersion := cluster.Status.Versions.ControlPlane
	if clusterVersion == "" {
		clusterVersion = cluster.Spec.Version
	}

	data.cloudProviderName = cluster.Spec.Cloud.ProviderName
	data.clusterVersion = clusterVersion
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

	if err := r.reconcileCRDs(ctx, data); err != nil {
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
		if err := r.ensureLoggingAgentIsRemoved(ctx); err != nil {
			return err
		}
	} else {
		// remove legacy promtail installation in user cluster
		if err := r.ensureLegacyPromtailIsRemoved(ctx); err != nil {
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

	if cluster.Spec.DisableCSIDriver {
		if err := r.ensureCSIDriverResourcesAreRemoved(ctx); err != nil {
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

	if !data.kubernetesDashboardEnabled {
		if err := r.ensureKubernetesDashboardResourcesAreRemoved(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) ensureAPIServices(ctx context.Context, data reconcileData) error {
	caCert := triple.EncodeCertPEM(data.caCert.Cert)
	creators := []kkpreconciling.NamedAPIServiceReconcilerFactory{
		metricsserver.APIServiceReconciler(caCert),
	}

	if err := kkpreconciling.ReconcileAPIServices(ctx, creators, metav1.NamespaceNone, r); err != nil {
		return fmt.Errorf("failed to reconcile APIServices: %w", err)
	}

	return nil
}

func (r *reconciler) reconcileServiceAccounts(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedServiceAccountReconcilerFactory{
		userauth.ServiceAccountReconciler(),
		usersshkeys.ServiceAccountReconciler(),
		coredns.ServiceAccountReconciler(),
	}

	if r.nodeLocalDNSCache {
		creators = append(creators, nodelocaldns.ServiceAccountReconciler())
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.ServiceAccountReconciler())
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", metav1.NamespaceSystem, err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedServiceAccountReconcilerFactory{
			kubernetesdashboard.ServiceAccountReconciler(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, kubernetesdashboard.Namespace, r); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	cloudInitSAReconciler := []reconciling.NamedServiceAccountReconcilerFactory{
		cloudinitsettings.ServiceAccountReconciler(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, cloudInitSAReconciler, resources.CloudInitSettingsNamespace, r); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter in the namespace %s: %w", resources.CloudInitSettingsNamespace, err)
	}

	// OPA related resources
	if r.opaIntegration {
		creators = []reconciling.NamedServiceAccountReconcilerFactory{
			gatekeeper.ServiceAccountReconciler(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, resources.GatekeeperNamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	if r.isKonnectivityEnabled {
		creators = []reconciling.NamedServiceAccountReconcilerFactory{
			konnectivity.ServiceAccountReconciler(),
			metricsserver.ServiceAccountReconciler(), // required only if metrics-server is running in user cluster
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", metav1.NamespaceSystem, err)
		}
	}

	creators = []reconciling.NamedServiceAccountReconcilerFactory{}
	if r.userClusterMLA.Logging {
		creators = append(creators,
			mlaloggingagent.ServiceAccountReconciler(),
		)
	}
	if r.userClusterMLA.Monitoring {
		creators = append(creators,
			mlamonitoringagent.ServiceAccountReconciler(),
		)
	}

	if len(creators) != 0 {
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, resources.UserClusterMLANamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}
	return nil
}

func (r *reconciler) reconcileRoles(ctx context.Context, data reconcileData) error {
	// kube-system
	creators := []reconciling.NamedRoleReconcilerFactory{
		machinecontroller.KubeSystemRoleReconciler(),
		operatingsystemmanager.KubeSystemRoleReconciler(),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.RoleReconciler())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", metav1.NamespaceSystem, err)
	}

	// kube-public
	creators = []reconciling.NamedRoleReconcilerFactory{
		machinecontroller.ClusterInfoReaderRoleReconciler(),
		machinecontroller.KubePublicRoleReconciler(),
		operatingsystemmanager.KubePublicRoleReconciler(),
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespacePublic, r); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", metav1.NamespacePublic, err)
	}

	// default
	creators = []reconciling.NamedRoleReconcilerFactory{
		machinecontroller.EndpointReaderRoleReconciler(),
		operatingsystemmanager.DefaultRoleReconciler(),
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespaceDefault, r); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", metav1.NamespaceDefault, err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedRoleReconcilerFactory{
			kubernetesdashboard.RoleReconciler(),
		}

		if err := reconciling.ReconcileRoles(ctx, creators, kubernetesdashboard.Namespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	cloudInitRoleReconciler := []reconciling.NamedRoleReconcilerFactory{
		cloudinitsettings.RoleReconciler(),
		operatingsystemmanager.CloudInitSettingsRoleReconciler(),
	}

	if err := reconciling.ReconcileRoles(ctx, cloudInitRoleReconciler, resources.CloudInitSettingsNamespace, r); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter role in the namespace %s: %w", resources.CloudInitSettingsNamespace, err)
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedRoleReconcilerFactory{
			gatekeeper.RoleReconciler(),
		}
		if err := reconciling.ReconcileRoles(ctx, creators, resources.GatekeeperNamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileRoleBindings(ctx context.Context, data reconcileData) error {
	// kube-system
	creators := []reconciling.NamedRoleBindingReconcilerFactory{
		machinecontroller.KubeSystemRoleBindingReconciler(),
		metricsserver.RolebindingAuthReaderReconciler(r.isKonnectivityEnabled),
		scheduler.RoleBindingAuthDelegator(),
		controllermanager.RoleBindingAuthDelegator(),
		operatingsystemmanager.KubeSystemRoleBindingReconciler(),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.RoleBindingReconciler())
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in kube-system Namespace: %w", err)
	}

	// kube-public
	creators = []reconciling.NamedRoleBindingReconcilerFactory{
		machinecontroller.KubePublicRoleBindingReconciler(),
		machinecontroller.ClusterInfoAnonymousRoleBindingReconciler(),
		operatingsystemmanager.KubePublicRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespacePublic, r); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in kube-public Namespace: %w", err)
	}

	// Default
	creators = []reconciling.NamedRoleBindingReconcilerFactory{
		machinecontroller.DefaultRoleBindingReconciler(),
		operatingsystemmanager.DefaultRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespaceDefault, r); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in default Namespace: %w", err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedRoleBindingReconcilerFactory{
			kubernetesdashboard.RoleBindingReconciler(),
		}
		if err := reconciling.ReconcileRoleBindings(ctx, creators, kubernetesdashboard.Namespace, r); err != nil {
			return fmt.Errorf("failed to reconcile RoleBindings in the namespace: %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	cloudInitRoleBindingReconciler := []reconciling.NamedRoleBindingReconcilerFactory{
		cloudinitsettings.RoleBindingReconciler(),
		operatingsystemmanager.CloudInitSettingsRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, cloudInitRoleBindingReconciler, resources.CloudInitSettingsNamespace, r); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter RoleBindings in the namespace: %s: %w", resources.CloudInitSettingsNamespace, err)
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedRoleBindingReconcilerFactory{
			gatekeeper.RoleBindingReconciler(),
		}
		if err := reconciling.ReconcileRoleBindings(ctx, creators, resources.GatekeeperNamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile RoleBindings in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileClusterRoles(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedClusterRoleReconcilerFactory{
		kubestatemetrics.ClusterRoleReconciler(),
		prometheus.ClusterRoleReconciler(),
		machinecontroller.ClusterRoleReconciler(),
		dnatcontroller.ClusterRoleReconciler(),
		metricsserver.ClusterRoleReconciler(),
		coredns.ClusterRoleReconciler(),
		operatingsystemmanager.WebhookClusterRoleReconciler(),
		operatingsystemmanager.ClusterRoleReconciler(),
	}

	if data.kubernetesDashboardEnabled {
		creators = append(creators, kubernetesdashboard.ClusterRoleReconciler())
	}

	if r.opaIntegration {
		creators = append(creators, gatekeeper.ClusterRoleReconciler())
	}

	if r.userClusterMLA.Logging {
		creators = append(creators, mlaloggingagent.ClusterRoleReconciler())
	}
	if r.userClusterMLA.Monitoring {
		creators = append(creators, mlamonitoringagent.ClusterRoleReconciler())
	}

	if err := reconciling.ReconcileClusterRoles(ctx, creators, "", r); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileClusterRoleBindings(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userauth.ClusterRoleBindingReconciler(),
		kubestatemetrics.ClusterRoleBindingReconciler(),
		prometheus.ClusterRoleBindingReconciler(),
		machinecontroller.ClusterRoleBindingReconciler(),
		machinecontroller.NodeBootstrapperClusterRoleBindingReconciler(),
		machinecontroller.NodeSignerClusterRoleBindingReconciler(),
		dnatcontroller.ClusterRoleBindingReconciler(),
		metricsserver.ClusterRoleBindingResourceReaderReconciler(r.isKonnectivityEnabled),
		metricsserver.ClusterRoleBindingAuthDelegatorReconciler(r.isKonnectivityEnabled),
		scheduler.ClusterRoleBindingAuthDelegatorReconciler(),
		controllermanager.ClusterRoleBindingAuthDelegator(),
		systembasicuser.ClusterRoleBinding,
		cloudcontroller.ClusterRoleBindingReconciler(),
		coredns.ClusterRoleBindingReconciler(),
		operatingsystemmanager.ClusterRoleBindingReconciler(),
		operatingsystemmanager.WebhookClusterRoleBindingReconciler(),
	}

	if data.kubernetesDashboardEnabled {
		creators = append(creators, kubernetesdashboard.ClusterRoleBindingReconciler())
	}

	if r.opaIntegration {
		creators = append(creators, gatekeeper.ClusterRoleBindingReconciler())
	}

	if r.userClusterMLA.Logging {
		creators = append(creators, mlaloggingagent.ClusterRoleBindingReconciler())
	}

	if r.userClusterMLA.Monitoring {
		creators = append(creators, mlamonitoringagent.ClusterRoleBindingReconciler())
	}

	if r.isKonnectivityEnabled {
		creators = append(creators, konnectivity.ClusterRoleBindingReconciler())
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, creators, "", r); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileCRDs(ctx context.Context, data reconcileData) error {
	c, err := crd.CRDForObject(&appskubermaticv1.ApplicationInstallation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appskubermaticv1.SchemeGroupVersion.String(),
			Kind:       "ApplicationInstallation",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get ApplicationInstallation CRD: %w", err)
	}

	creators := []kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory{
		machinecontroller.MachineCRDReconciler(),
		machinecontroller.MachineSetCRDReconciler(),
		machinecontroller.MachineDeploymentCRDReconciler(),
		applications.CRDReconciler(c),
		operatingsystemmanager.OperatingSystemConfigCRDReconciler(),
		operatingsystemmanager.OperatingSystemProfileCRDReconciler(),
	}

	if r.opaIntegration {
		gatekeeperCRDs, err := gatekeeper.CRDs()
		if err != nil {
			return fmt.Errorf("failed to load Gatekeeper CRDs: %w", err)
		}

		for i := range gatekeeperCRDs {
			creators = append(creators, gatekeeper.CRDReconciler(gatekeeperCRDs[i]))
		}
	}

	if err := kkpreconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", r); err != nil {
		return fmt.Errorf("failed to reconcile CustomResourceDefinitions: %w", err)
	}

	return nil
}

func (r *reconciler) reconcileMutatingWebhookConfigurations(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedMutatingWebhookConfigurationReconcilerFactory{
		applications.ApplicationInstallationMutatingWebhookConfigurationReconciler(data.caCert.Cert, r.namespace),
		operatingsystemmanager.MutatingwebhookConfigurationReconciler(data.caCert.Cert, r.namespace),
	}

	if data.cloudProviderName != string(kubermaticv1.EdgeCloudProvider) {
		creators = append(creators, machinecontroller.MutatingwebhookConfigurationReconciler(data.caCert.Cert, r.namespace))
	}

	if r.opaIntegration && r.opaEnableMutation {
		creators = append(creators, gatekeeper.MutatingWebhookConfigurationReconciler(r.opaWebhookTimeout))
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, creators, "", r); err != nil {
		return fmt.Errorf("failed to reconcile MutatingWebhookConfigurations: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileValidatingWebhookConfigurations(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedValidatingWebhookConfigurationReconcilerFactory{
		applications.ApplicationInstallationValidatingWebhookConfigurationReconciler(data.caCert.Cert, r.namespace),
		operatingsystemmanager.ValidatingWebhookConfigurationReconciler(data.caCert.Cert, r.namespace),
	}

	if data.cloudProviderName != string(kubermaticv1.EdgeCloudProvider) {
		creators = append(creators, machine.ValidatingWebhookConfigurationReconciler(data.caCert.Cert, r.namespace))
	}

	if r.opaIntegration {
		creators = append(creators, gatekeeper.ValidatingWebhookConfigurationReconciler(r.opaWebhookTimeout))
	}

	if data.ccmMigration && data.csiCloudConfig != nil {
		creators = append(creators, csimigration.ValidatingwebhookConfigurationReconciler(data.caCert.Cert, metav1.NamespaceSystem, resources.VsphereCSIMigrationWebhookConfigurationWebhookName))
	}

	if !data.cluster.Spec.DisableCSIDriver {
		if r.cloudProvider == kubermaticv1.VSphereCloudProvider || r.cloudProvider == kubermaticv1.NutanixCloudProvider || r.cloudProvider == kubermaticv1.OpenstackCloudProvider ||
			r.cloudProvider == kubermaticv1.DigitaloceanCloudProvider {
			creators = append(creators, csisnapshotter.ValidatingSnapshotWebhookConfigurationReconciler(data.caCert.Cert, metav1.NamespaceSystem, resources.CSISnapshotValidationWebhookConfigurationName))
		}
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, creators, "", r); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingWebhookConfigurations: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileServices(ctx context.Context, data reconcileData) error {
	creatorsKubeSystem := []reconciling.NamedServiceReconcilerFactory{
		coredns.ServiceReconciler(r.dnsClusterIP),
	}
	if r.isKonnectivityEnabled {
		// metrics-server running in user cluster - ClusterIP service
		creatorsKubeSystem = append(creatorsKubeSystem, metricsserver.ServiceReconciler(data.ipFamily))
	} else {
		// metrics-server running in seed cluster - ExternalName service
		creatorsKubeSystem = append(creatorsKubeSystem, metricsserver.ExternalNameServiceReconciler(r.namespace))
	}

	if err := reconciling.ReconcileServices(ctx, creatorsKubeSystem, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile Services in kube-system namespace: %w", err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators := []reconciling.NamedServiceReconcilerFactory{
			kubernetesdashboard.ServiceReconciler(data.ipFamily),
		}
		if err := reconciling.ReconcileServices(ctx, creators, kubernetesdashboard.Namespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Services in namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	// OPA related resources
	if r.opaIntegration {
		creators := []reconciling.NamedServiceReconcilerFactory{
			gatekeeper.ServiceReconciler(),
		}
		if err := reconciling.ReconcileServices(ctx, creators, resources.GatekeeperNamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Services in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileEndpoints(ctx context.Context, data reconcileData) error {
	if !data.reconcileK8sSvcEndpoints {
		return nil
	}
	epReconcilers := []reconciling.NamedEndpointsReconcilerFactory{
		kubernetesresources.EndpointsReconciler(data.k8sServiceEndpointAddress, data.k8sServiceEndpointPort),
	}
	if err := reconciling.ReconcileEndpoints(ctx, epReconcilers, metav1.NamespaceDefault, r); err != nil {
		return fmt.Errorf("failed to reconcile Endpoints: %w", err)
	}
	epSliceReconcilers := []reconciling.NamedEndpointSliceReconcilerFactory{
		kubernetesresources.EndpointSliceReconciler(data.k8sServiceEndpointAddress, data.k8sServiceEndpointPort),
	}
	if err := reconciling.ReconcileEndpointSlices(ctx, epSliceReconcilers, metav1.NamespaceDefault, r); err != nil {
		return fmt.Errorf("failed to reconcile EndpointSlices: %w", err)
	}
	return nil
}

func (r *reconciler) reconcileConfigMaps(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedConfigMapReconcilerFactory{
		machinecontroller.ClusterInfoConfigMapReconciler(r.clusterURL.String(), data.caCert.Cert),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespacePublic, r); err != nil {
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
		creators = []reconciling.NamedConfigMapReconcilerFactory{
			cabundle.ConfigMapReconciler(r.caBundle),
			envoyagent.ConfigMapReconciler(envoyConfig),
		}
		if !r.isKonnectivityEnabled {
			creators = append(creators, openvpn.ClientConfigConfigMapReconciler(r.tunnelingAgentIP.String(), r.openvpnServerPort))
		}
	} else {
		creators = []reconciling.NamedConfigMapReconcilerFactory{
			cabundle.ConfigMapReconciler(r.caBundle),
		}
		if !r.isKonnectivityEnabled {
			creators = append(creators, openvpn.ClientConfigConfigMapReconciler(r.clusterURL.Hostname(), r.openvpnServerPort))
		}
	}

	creators = append(creators, coredns.ConfigMapReconciler())

	if r.nodeLocalDNSCache {
		creators = append(creators, nodelocaldns.ConfigMapReconciler(r.dnsClusterIP))
	}

	if data.csiCloudConfig != nil {
		if r.cloudProvider == kubermaticv1.VMwareCloudDirectorCloudProvider {
			creators = append(creators, cloudcontroller.VMwareCloudDirectorCSIConfig(data.csiCloudConfig))
		}
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-system namespace: %w", err)
	}

	if r.userClusterMLA.Monitoring {
		customScrapeConfigs, err := r.getUserClusterMonitoringAgentCustomScrapeConfigs(ctx)
		if err != nil {
			return fmt.Errorf("failed to get user cluster prometheus custom scrape configs: %w", err)
		}
		creators = []reconciling.NamedConfigMapReconcilerFactory{
			mlamonitoringagent.ConfigMapReconciler(mlamonitoringagent.Config{
				MLAGatewayURL:       r.userClusterMLA.MLAGatewayURL + "/api/v1/push",
				TLSCertFile:         fmt.Sprintf("%s/%s", resources.MLAMonitoringAgentClientCertMountPath, resources.MLAMonitoringAgentClientCertSecretKey),
				TLSKeyFile:          fmt.Sprintf("%s/%s", resources.MLAMonitoringAgentClientCertMountPath, resources.MLAMonitoringAgentClientKeySecretKey),
				TLSCACertFile:       fmt.Sprintf("%s/%s", resources.MLAMonitoringAgentClientCertMountPath, resources.MLAGatewayCACertKey),
				CustomScrapeConfigs: customScrapeConfigs,
				HAClusterIdentifier: r.clusterName,
			}),
		}
		if err := reconciling.ReconcileConfigMaps(ctx, creators, resources.UserClusterMLANamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile ConfigMap in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}
	return nil
}

func (r *reconciler) reconcileSecrets(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedSecretReconcilerFactory{}
	if !r.isKonnectivityEnabled {
		creators = append(creators, openvpn.ClientCertificate(data.openVPNCACert))
	} else {
		// required only if metrics-server is running in user cluster
		creators = append(creators, metricsserver.TLSServingCertSecretReconciler(
			func() (*triple.KeyPair, error) {
				return data.caCert, nil
			}),
		)
	}

	if !data.cluster.Spec.DisableCSIDriver && data.csiCloudConfig != nil {
		if r.cloudProvider == kubermaticv1.AzureCloudProvider || r.cloudProvider == kubermaticv1.OpenstackCloudProvider || r.cloudProvider == kubermaticv1.VSphereCloudProvider {
			creators = append(creators, cloudcontroller.CloudConfig(data.csiCloudConfig, resources.CSICloudConfigSecretName))
		}

		if r.cloudProvider == kubermaticv1.VSphereCloudProvider {
			creators = append(creators, csisnapshotter.TLSServingCertificateReconciler(resources.CSISnapshotValidationWebhookName, data.caCert))
			if data.ccmMigration {
				creators = append(creators, csimigration.TLSServingCertificateReconciler(data.caCert))
			}
		}

		if r.cloudProvider == kubermaticv1.NutanixCloudProvider {
			creators = append(creators, cloudcontroller.NutanixCSIConfig(data.csiCloudConfig),
				csisnapshotter.TLSServingCertificateReconciler(resources.CSISnapshotValidationWebhookName, data.caCert))
		}
	}

	if !data.cluster.Spec.DisableCSIDriver {
		if r.cloudProvider == kubermaticv1.OpenstackCloudProvider || r.cloudProvider == kubermaticv1.DigitaloceanCloudProvider {
			creators = append(creators, csisnapshotter.TLSServingCertificateReconciler(resources.CSISnapshotValidationWebhookName, data.caCert))
		}
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.SecretReconciler(data.userSSHKeys))
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in kube-system Namespace: %w", err)
	}

	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators = []reconciling.NamedSecretReconcilerFactory{
			kubernetesdashboard.KeyHolderSecretReconciler(),
			kubernetesdashboard.CsrfTokenSecretReconciler(),
		}

		if err := reconciling.ReconcileSecrets(ctx, creators, kubernetesdashboard.Namespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedSecretReconcilerFactory{
			gatekeeper.SecretReconciler(),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.GatekeeperNamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	if r.userClusterMLA.Monitoring {
		creators = []reconciling.NamedSecretReconcilerFactory{
			mlamonitoringagent.ClientCertificateReconciler(data.mlaGatewayCACert),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.UserClusterMLANamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}
	if r.userClusterMLA.Logging {
		creators = []reconciling.NamedSecretReconcilerFactory{
			mlaloggingagent.SecretReconciler(mlaloggingagent.Config{
				MLAGatewayURL: r.userClusterMLA.MLAGatewayURL + "/loki/api/v1/push",
				TLSCertFile:   fmt.Sprintf("%s/%s", resources.MLALoggingAgentClientCertMountPath, resources.MLALoggingAgentClientCertSecretKey),
				TLSKeyFile:    fmt.Sprintf("%s/%s", resources.MLALoggingAgentClientCertMountPath, resources.MLALoggingAgentClientKeySecretKey),
				TLSCACertFile: fmt.Sprintf("%s/%s", resources.MLALoggingAgentClientCertMountPath, resources.MLAGatewayCACertKey),
			}),
			mlaloggingagent.ClientCertificateReconciler(data.mlaGatewayCACert),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.UserClusterMLANamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}

	creators = []reconciling.NamedSecretReconcilerFactory{
		cloudinitsettings.SecretReconciler(),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, resources.CloudInitSettingsNamespace, r); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", resources.CloudInitSettingsNamespace, err)
	}

	return nil
}

// psaPrivilegedLabeler returns a namespace reconciler that applies PSA privileged labels.
func psaPrivilegedLabeler(namespace string) reconciling.NamedNamespaceReconcilerFactory {
	return func() (string, reconciling.NamespaceReconciler) {
		return namespace, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			for k, v := range resources.PSALabelsPrivileged() {
				ns.Labels[k] = v
			}
			return ns, nil
		}
	}
}

// psaBaselineLabeler returns a namespace reconciler that applies PSA baseline labels.
func psaBaselineLabeler(namespace string) reconciling.NamedNamespaceReconcilerFactory {
	return func() (string, reconciling.NamespaceReconciler) {
		return namespace, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			for k, v := range resources.PSALabelsBaseline() {
				ns.Labels[k] = v
			}
			return ns, nil
		}
	}
}

func (r *reconciler) reconcileDaemonSet(ctx context.Context, data reconcileData) error {
	var dsReconcilers []reconciling.NamedDaemonSetReconcilerFactory

	if r.nodeLocalDNSCache {
		dsReconcilers = append(dsReconcilers, nodelocaldns.DaemonSetReconciler(r.imageRewriter))
	}

	if r.userSSHKeyAgent {
		dsReconcilers = append(dsReconcilers, usersshkeys.DaemonSetReconciler(r.versions, r.imageRewriter))
	}

	if len(r.tunnelingAgentIP) > 0 {
		configHash, err := r.getEnvoyAgentConfigHash(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve envoy-agent config hash: %w", err)
		}
		dsReconcilers = append(dsReconcilers, envoyagent.DaemonSetReconciler(r.tunnelingAgentIP, r.versions, configHash, r.imageRewriter))
	}

	if err := reconciling.ReconcileDaemonSets(ctx, dsReconcilers, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile the DaemonSet: %w", err)
	}

	if r.userClusterMLA.Logging {
		dsReconcilers = []reconciling.NamedDaemonSetReconcilerFactory{
			mlaloggingagent.DaemonSetReconciler(data.loggingRequirements, r.imageRewriter),
		}
		if err := reconciling.ReconcileDaemonSets(ctx, dsReconcilers, resources.UserClusterMLANamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile the DaemonSet: %w", err)
		}
	}
	return nil
}

func (r *reconciler) reconcileNamespaces(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedNamespaceReconcilerFactory{
		cloudinitsettings.NamespaceReconciler,
		// PSA labels for system namespaces
		psaPrivilegedLabeler(metav1.NamespaceSystem),
		psaPrivilegedLabeler(metav1.NamespacePublic),
		psaPrivilegedLabeler(resources.NamespaceNodeLease),
		psaBaselineLabeler(metav1.NamespaceDefault),
	}
	if data.kubernetesDashboardEnabled {
		creators = append(creators, kubernetesdashboard.NamespaceReconciler)
	}

	if r.opaIntegration {
		creators = append(creators, gatekeeper.NamespaceReconciler)
		creators = append(creators, gatekeeper.KubeSystemLabeler)
	}
	if r.userClusterMLA.Logging || r.userClusterMLA.Monitoring {
		creators = append(creators, mla.NamespaceReconciler)
	}

	if err := reconciling.ReconcileNamespaces(ctx, creators, "", r); err != nil {
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
	err := r.reconcileDefaultServiceAccount(ctx, metav1.NamespaceSystem)
	if err != nil {
		return fmt.Errorf("failed to update default service account: %w", err)
	}

	return nil
}

func (r *reconciler) reconcileDeployments(ctx context.Context, data reconcileData) error {
	// Kubernetes Dashboard and related resources
	if data.kubernetesDashboardEnabled {
		creators := []reconciling.NamedDeploymentReconcilerFactory{
			kubernetesdashboard.DeploymentReconciler(r.imageRewriter),
		}
		if err := reconciling.ReconcileDeployments(ctx, creators, kubernetesdashboard.Namespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", kubernetesdashboard.Namespace, err)
		}
	}

	kubeSystemReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		coredns.DeploymentReconciler(r.clusterSemVer, data.cluster, r.imageRewriter),
	}

	if err := reconciling.ReconcileDeployments(ctx, kubeSystemReconcilers, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", metav1.NamespaceSystem, err)
	}

	// OPA related resources
	if r.opaIntegration {
		creators := []reconciling.NamedDeploymentReconcilerFactory{
			gatekeeper.ControllerDeploymentReconciler(r.opaEnableMutation, r.imageRewriter, data.gatekeeperCtrlRequirements),
			gatekeeper.AuditDeploymentReconciler(r.imageRewriter, data.gatekeeperAuditRequirements),
		}

		if err := reconciling.ReconcileDeployments(ctx, creators, resources.GatekeeperNamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}

	if r.userClusterMLA.Monitoring {
		creators := []reconciling.NamedDeploymentReconcilerFactory{
			mlamonitoringagent.DeploymentReconciler(data.monitoringRequirements, data.monitoringReplicas, r.imageRewriter),
		}
		if err := reconciling.ReconcileDeployments(ctx, creators, resources.UserClusterMLANamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", resources.UserClusterMLANamespace, err)
		}
	}

	if r.isKonnectivityEnabled {
		konnectivityResources := resources.GetOverrides(data.cluster.Spec.ComponentsOverride)

		creators := []reconciling.NamedDeploymentReconcilerFactory{
			konnectivity.DeploymentReconciler(
				data.clusterVersion, data.cluster,
				r.konnectivityServerHost, r.konnectivityServerPort, r.konnectivityKeepaliveTime,
				r.imageRewriter, konnectivityResources,
			),
			metricsserver.DeploymentReconciler(r.imageRewriter), // deploy metrics-server in user cluster
		}
		if err := reconciling.ReconcileDeployments(ctx, creators, metav1.NamespaceSystem, r); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", metav1.NamespaceSystem, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileNetworkPolicies(ctx context.Context, data reconcileData) error {
	namedNetworkPolicyReconcilerFactories := []reconciling.NamedNetworkPolicyReconcilerFactory{
		kubesystem.DefaultNetworkPolicyReconciler(),
		coredns.KubeDNSNetworkPolicyReconciler(data.k8sServiceEndpointAddress, int(data.k8sServiceEndpointPort), data.k8sServiceAPIIP.String()),
	}

	if r.userSSHKeyAgent {
		namedNetworkPolicyReconcilerFactories = append(namedNetworkPolicyReconcilerFactories,
			usersshkeys.NetworkPolicyReconciler(data.k8sServiceEndpointAddress, int(data.k8sServiceEndpointPort), data.k8sServiceAPIIP.String()))
	}

	if r.isKonnectivityEnabled {
		namedNetworkPolicyReconcilerFactories = append(namedNetworkPolicyReconcilerFactories, metricsserver.NetworkPolicyReconciler(), konnectivity.NetworkPolicyReconciler())
	}

	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyReconcilerFactories, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to ensure Network Policies: %w", err)
	}

	return nil
}

func (r *reconciler) reconcilePodDisruptionBudgets(ctx context.Context) error {
	creators := []reconciling.NamedPodDisruptionBudgetReconcilerFactory{
		coredns.PodDisruptionBudgetReconciler(),
	}
	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedPodDisruptionBudgetReconcilerFactory{
			gatekeeper.PodDisruptionBudgetReconciler(),
		}
		if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, resources.GatekeeperNamespace, r); err != nil {
			return fmt.Errorf("failed to reconcile PodDisruptionBudgets in namespace %s: %w", resources.GatekeeperNamespace, err)
		}
	}
	if r.isKonnectivityEnabled {
		creators = append(creators,
			konnectivity.PodDisruptionBudgetReconciler(),
			metricsserver.PodDisruptionBudgetReconciler(),
		)
	}
	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %w", err)
	}
	return nil
}

type reconcileData struct {
	caCert            *triple.KeyPair
	openVPNCACert     *resources.ECDSAKeyPair
	mlaGatewayCACert  *resources.ECDSAKeyPair
	userSSHKeys       map[string][]byte
	cloudProviderName string
	clusterVersion    semver.Semver
	cluster           *kubermaticv1.Cluster
	// csiCloudConfig is currently used only by vSphere, VMware Cloud Director and Nutanix,
	// who need it to properly configure the external CSI driver; however this can be nil if the
	// CSI driver has been explicitly disabled
	csiCloudConfig              []byte
	ccmMigration                bool
	monitoringRequirements      *corev1.ResourceRequirements
	loggingRequirements         *corev1.ResourceRequirements
	gatekeeperCtrlRequirements  *corev1.ResourceRequirements
	gatekeeperAuditRequirements *corev1.ResourceRequirements
	monitoringReplicas          *int32
	ipFamily                    kubermaticv1.IPFamily
	k8sServiceAPIIP             *net.IP
	k8sServiceEndpointAddress   string
	k8sServiceEndpointPort      int32
	reconcileK8sSvcEndpoints    bool
	kubernetesDashboardEnabled  bool
}

func (r *reconciler) ensureOPAIntegrationIsRemoved(ctx context.Context) error {
	resources, err := gatekeeper.GetResourcesToRemoveOnDelete()
	if err != nil {
		return err
	}

	for _, resource := range resources {
		err := r.Delete(ctx, resource)
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
	if err := r.Delete(ctx, &admissionregistrationv1.MutatingWebhookConfiguration{
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

	return util.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
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
		r,
		types.NamespacedName{Namespace: resources.GatekeeperNamespace, Name: resources.GatekeeperControllerDeploymentName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, kubermaticv1.HealthStatusDown,
			fmt.Errorf("failed to get dep health %s: %w", resources.GatekeeperControllerDeploymentName, err)
	}

	auditHealth, err = resources.HealthyDeployment(ctx,
		r,
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
		r,
		types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLAMonitoringAgentDeploymentName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown,
			fmt.Errorf("failed to get dep health %s: %w", resources.MLAMonitoringAgentDeploymentName, err)
	}

	return health, nil
}

func (r *reconciler) getMLALoggingHealth(ctx context.Context) (kubermaticv1.HealthStatus, error) {
	loggingHealth, err := resources.HealthyDaemonSet(ctx,
		r,
		types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLALoggingAgentDaemonSetName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to get ds health %s: %w", resources.MLALoggingAgentDaemonSetName, err)
	}
	return loggingHealth, nil
}

func (r *reconciler) ensureLoggingAgentIsRemoved(ctx context.Context) error {
	for _, resource := range mlaloggingagent.ResourcesOnDeletion() {
		err := r.Delete(ctx, resource)
		if errC := r.cleanUpMLAHealthStatus(ctx, true, false, err); errC != nil {
			return fmt.Errorf("failed to update mla logging health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure logging agent is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureLegacyPromtailIsRemoved(ctx context.Context) error {
	for _, resource := range mlaloggingagent.LegacyResourcesOnDeletion() {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure promtail is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureUserClusterMonitoringAgentIsRemoved(ctx context.Context) error {
	for _, resource := range mlamonitoringagent.ResourcesOnDeletion() {
		err := r.Delete(ctx, resource)
		if errC := r.cleanUpMLAHealthStatus(ctx, false, true, err); errC != nil {
			return fmt.Errorf("failed to update mla monitoring health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure user cluster monitoring agent is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureLegacyPrometheusIsRemoved(ctx context.Context) error {
	for _, resource := range mlamonitoringagent.LegacyResourcesOnDeletion() {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure user cluster monitoring agent is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureMLAIsRemoved(ctx context.Context) error {
	for _, resource := range mla.ResourcesOnDeletion() {
		err := r.Delete(ctx, resource)
		if errC := r.cleanUpMLAHealthStatus(ctx, true, true, err); errC != nil {
			return fmt.Errorf("failed to update mla health status in cluster: %w", errC)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure mla is removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureCSIDriverResourcesAreRemoved(ctx context.Context) error {
	for _, resource := range cloudcontroller.ResourcesForDeletion(metav1.NamespaceSystem) {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure CSI driver resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureOpenVPNSetupIsRemoved(ctx context.Context) error {
	for _, resource := range openvpn.ResourcesForDeletion() {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OpenVPN resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureKonnectivitySetupIsRemoved(ctx context.Context) error {
	for _, resource := range konnectivity.ResourcesForDeletion() {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure Konnectivity resources are removed/not present: %w", err)
		}
	}
	for _, resource := range metricsserver.UserClusterResourcesForDeletion() {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure metrics-server resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *reconciler) ensureKubernetesDashboardResourcesAreRemoved(ctx context.Context) error {
	for _, resource := range kubernetesdashboard.ResourcesForDeletion() {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure Kubernetes Dashboard resources are removed/not present: %w", err)
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
	err := r.Get(ctx, types.NamespacedName{Name: resources.EnvoyAgentConfigMapName, Namespace: metav1.NamespaceSystem}, &cm)
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
	return util.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
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
	return util.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
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
