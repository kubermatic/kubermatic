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
	"fmt"
	"net"
	"strings"

	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/cloudcontroller"
	cabundle "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/ca-bundle"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/cloudinitsettings"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/clusterautoscaler"
	controllermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/controller-manager"
	coredns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/core-dns"
	csimigration "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/csi-migration"
	dnatcontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/dnat-controller"
	envoyagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/envoy-agent"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/gatekeeper"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/konnectivity"
	kubestatemetrics "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kube-state-metrics"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubernetes-dashboard"
	machinecontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine-controller"
	metricsserver "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/metrics-server"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla"
	userclusterprometheus "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla/prometheus"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/mla/promtail"
	nodelocaldns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/node-local-dns"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/openvpn"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/prometheus"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/scheduler"
	systembasicuser "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/system-basic-user"
	userauth "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/user-auth"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/usersshkeys"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) reconcile(ctx context.Context) error {
	caCert, err := r.caCert(ctx)
	if err != nil {
		return fmt.Errorf("failed to get caCert: %v", err)
	}
	openVPNCACert, err := r.openVPNCA(ctx)
	if err != nil {
		return fmt.Errorf("failed to get openVPN CA cert: %v", err)
	}
	userSSHKeys, err := r.userSSHKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get userSSHKeys: %v", err)
	}
	cloudConfig, err := r.cloudConfig(ctx, resources.CloudConfigConfigMapName)
	if err != nil {
		return fmt.Errorf("failed to get cloudConfig: %v", err)
	}
	var CSICloudConfig []byte
	if r.cloudProvider == kubermaticv1.ProviderVSphere {
		CSICloudConfig, err = r.cloudConfig(ctx, resources.CSICloudConfigConfigMapName)
		if err != nil {
			return fmt.Errorf("failed to get cloudConfig: %v", err)
		}
	}

	data := reconcileData{
		caCert:         caCert,
		openVPNCACert:  openVPNCACert,
		userSSHKeys:    userSSHKeys,
		cloudConfig:    cloudConfig,
		csiCloudConfig: CSICloudConfig,
		ccmMigration:   r.ccmMigration || r.ccmMigrationCompleted,
	}

	if r.userClusterMLA.Monitoring || r.userClusterMLA.Logging {
		data.mlaGatewayCACert, err = r.mlaGatewayCA(ctx)
		if err != nil {
			return fmt.Errorf("failed to get MLA Gateway CA cert: %v", err)
		}
		data.monitoringRequirements, data.loggingRequirements, err = r.mlaResourceRequirements(ctx)
		if err != nil {
			return fmt.Errorf("failed to get MLA resource requirements: %w", err)
		}
	}

	// Must be first because of openshift
	if err := r.ensureAPIServices(ctx, data); err != nil {
		return err
	}

	// We need to reconcile namespaces and services next to make sure
	// the openshift apiservices become available ASAP
	if err := r.reconcileNamespaces(ctx); err != nil {
		return err
	}

	if err := r.reconcileServiceAccounts(ctx); err != nil {
		return err
	}

	if err := r.reconcilePodDisruptionBudgets(ctx); err != nil {
		return err
	}

	if err := r.reconcileDeployments(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileServices(ctx); err != nil {
		return err
	}

	if err := r.reconcileClusterRoles(ctx); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(ctx); err != nil {
		return err
	}

	if err := r.reconcileRoles(ctx); err != nil {
		return err
	}

	if err := r.reconcileRoleBindings(ctx); err != nil {
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

	// Try to delete OPA integration deployment if its present
	if !r.opaIntegration {
		if err := r.ensureOPAIntegrationIsRemoved(ctx); err != nil {
			return err
		}
	} else {
		if err := r.healthCheck(ctx); err != nil {
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
		if err := r.ensureUserClusterPrometheusIsRemoved(ctx); err != nil {
			return err
		}
	}
	if !r.userClusterMLA.Logging && !r.userClusterMLA.Monitoring {
		if err := r.ensureMLAIsRemoved(ctx); err != nil {
			return err
		}
	}

	if r.isKonnectivityEnabled {
		if err := r.reconcileKonnectivityDeployments(ctx); err != nil {
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
		return fmt.Errorf("failed to reconcile APIServices: %v", err)
	}

	return nil
}

func (r *reconciler) reconcileServiceAccounts(ctx context.Context) error {
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
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", metav1.NamespaceSystem, err)
	}

	// Kubernetes Dashboard and related resources
	creators = []reconciling.NamedServiceAccountCreatorGetter{
		kubernetesdashboard.ServiceAccountCreator(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", kubernetesdashboard.Namespace, err)
	}

	cloudInitSACreator := []reconciling.NamedServiceAccountCreatorGetter{
		cloudinitsettings.ServiceAccountCreator(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, cloudInitSACreator, resources.CloudInitSettingsNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter in the namespace %s: %v", resources.CloudInitSettingsNamespace, err)
	}

	// OPA related resources
	if r.opaIntegration {
		creators = []reconciling.NamedServiceAccountCreatorGetter{
			gatekeeper.ServiceAccountCreator(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", resources.GatekeeperNamespace, err)
		}
	}

	if r.isKonnectivityEnabled {
		creators = []reconciling.NamedServiceAccountCreatorGetter{
			konnectivity.ServiceAccountCreator(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", metav1.NamespaceSystem, err)
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
			userclusterprometheus.ServiceAccountCreator(),
		)
	}

	if len(creators) != 0 {
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", resources.UserClusterMLANamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileRoles(ctx context.Context) error {
	// kube-system
	creators := []reconciling.NamedRoleCreatorGetter{
		machinecontroller.KubeSystemRoleCreator(),
		clusterautoscaler.KubeSystemRoleCreator(),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.RoleCreator())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", metav1.NamespaceSystem, err)
	}

	// kube-public
	creators = []reconciling.NamedRoleCreatorGetter{
		machinecontroller.ClusterInfoReaderRoleCreator(),
		machinecontroller.KubePublicRoleCreator(),
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespacePublic, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", metav1.NamespacePublic, err)
	}

	// default
	creators = []reconciling.NamedRoleCreatorGetter{
		machinecontroller.EndpointReaderRoleCreator(),
		clusterautoscaler.DefaultRoleCreator(),
	}

	if err := reconciling.ReconcileRoles(ctx, creators, metav1.NamespaceDefault, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", metav1.NamespaceDefault, err)
	}

	// Kubernetes Dashboard and related resources
	creators = []reconciling.NamedRoleCreatorGetter{
		kubernetesdashboard.RoleCreator(),
	}
	if err := reconciling.ReconcileRoles(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", kubernetesdashboard.Namespace, err)
	}

	cloudInitRoleCreator := []reconciling.NamedRoleCreatorGetter{
		cloudinitsettings.RoleCreator(),
	}
	if err := reconciling.ReconcileRoles(ctx, cloudInitRoleCreator, resources.CloudInitSettingsNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter role in the namespace %s: %v", resources.CloudInitSettingsNamespace, err)
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedRoleCreatorGetter{
			gatekeeper.RoleCreator(),
		}
		if err := reconciling.ReconcileRoles(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileRoleBindings(ctx context.Context) error {
	// kube-system
	creators := []reconciling.NamedRoleBindingCreatorGetter{
		machinecontroller.KubeSystemRoleBindingCreator(),
		metricsserver.RolebindingAuthReaderCreator(),
		scheduler.RoleBindingAuthDelegator(),
		controllermanager.RoleBindingAuthDelegator(),
		clusterautoscaler.KubeSystemRoleBindingCreator(),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.RoleBindingCreator())
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in kube-system Namespace: %v", err)
	}

	// kube-public
	creators = []reconciling.NamedRoleBindingCreatorGetter{
		machinecontroller.KubePublicRoleBindingCreator(),
		machinecontroller.ClusterInfoAnonymousRoleBindingCreator(),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespacePublic, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in kube-public Namespace: %v", err)
	}

	// Default
	creators = []reconciling.NamedRoleBindingCreatorGetter{
		machinecontroller.DefaultRoleBindingCreator(),
		clusterautoscaler.DefaultRoleBindingCreator(),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, creators, metav1.NamespaceDefault, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in default Namespace: %v", err)
	}

	// Kubernetes Dashboard and related resources
	creators = []reconciling.NamedRoleBindingCreatorGetter{
		kubernetesdashboard.RoleBindingCreator(),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in the namespace: %s: %v", kubernetesdashboard.Namespace, err)
	}

	cloudInitRoleBindingCreator := []reconciling.NamedRoleBindingCreatorGetter{
		cloudinitsettings.RoleBindingCreator(),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, cloudInitRoleBindingCreator, resources.CloudInitSettingsNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile cloud-init-getter RoleBindings in the namespace: %s: %v", resources.CloudInitSettingsNamespace, err)
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedRoleBindingCreatorGetter{
			gatekeeper.RoleBindingCreator(),
		}
		if err := reconciling.ReconcileRoleBindings(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile RoleBindings in namespace %s: %v", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileClusterRoles(ctx context.Context) error {
	creators := []reconciling.NamedClusterRoleCreatorGetter{
		kubestatemetrics.ClusterRoleCreator(),
		prometheus.ClusterRoleCreator(),
		machinecontroller.ClusterRoleCreator(),
		dnatcontroller.ClusterRoleCreator(),
		metricsserver.ClusterRoleCreator(),
		clusterautoscaler.ClusterRoleCreator(),
		kubernetesdashboard.ClusterRoleCreator(),
		coredns.ClusterRoleCreator(),
	}
	if r.opaIntegration {
		creators = append(creators, gatekeeper.ClusterRoleCreator())
	}

	if r.userClusterMLA.Logging {
		creators = append(creators, promtail.ClusterRoleCreator())
	}
	if r.userClusterMLA.Monitoring {
		creators = append(creators, userclusterprometheus.ClusterRoleCreator())
	}

	if err := reconciling.ReconcileClusterRoles(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileClusterRoleBindings(ctx context.Context) error {
	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		userauth.ClusterRoleBindingCreator(),
		kubestatemetrics.ClusterRoleBindingCreator(),
		prometheus.ClusterRoleBindingCreator(),
		machinecontroller.ClusterRoleBindingCreator(),
		machinecontroller.NodeBootstrapperClusterRoleBindingCreator(),
		machinecontroller.NodeSignerClusterRoleBindingCreator(),
		dnatcontroller.ClusterRoleBindingCreator(),
		metricsserver.ClusterRoleBindingResourceReaderCreator(),
		metricsserver.ClusterRoleBindingAuthDelegatorCreator(),
		scheduler.ClusterRoleBindingAuthDelegatorCreator(),
		controllermanager.ClusterRoleBindingAuthDelegator(),
		clusterautoscaler.ClusterRoleBindingCreator(),
		systembasicuser.ClusterRoleBinding,
		cloudcontroller.ClusterRoleBindingCreator(),
		kubernetesdashboard.ClusterRoleBindingCreator(),
		coredns.ClusterRoleBindingCreator(),
	}
	if r.opaIntegration {
		creators = append(creators, gatekeeper.ClusterRoleBindingCreator())
	}

	if r.userClusterMLA.Logging {
		creators = append(creators, promtail.ClusterRoleBindingCreator())
	}

	if r.userClusterMLA.Monitoring {
		creators = append(creators, userclusterprometheus.ClusterRoleBindingCreator())
	}

	if r.isKonnectivityEnabled {
		creators = append(creators, konnectivity.ClusterRoleBindingCreator())
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileCRDs(ctx context.Context) error {
	creators := []reconciling.NamedCustomResourceDefinitionCreatorGetter{
		machinecontroller.MachineCRDCreator(),
		machinecontroller.MachineSetCRDCreator(),
		machinecontroller.MachineDeploymentCRDCreator(),
		machinecontroller.ClusterCRDCreator(),
	}

	if r.opaIntegration {
		creators = append(creators,
			gatekeeper.ConfigCRDCreator(),
			gatekeeper.ConstraintTemplateCRDCreator(),
			gatekeeper.ConstraintPodStatusCRDCreator(),
			gatekeeper.ConstraintTemplatePodStatusCRDCreator(),
			gatekeeper.MutatorPodStatusCRDCreator(),
			gatekeeper.AssignCRDCreator(),
			gatekeeper.AssignMetadataCRDCreator())
	}

	if err := reconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile CustomResourceDefinitions: %v", err)
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
	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile MutatingWebhookConfigurations: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileValidatingWebhookConfigurations(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{}
	if r.opaIntegration {
		creators = append(creators, gatekeeper.ValidatingWebhookConfigurationCreator(r.opaWebhookTimeout))
	}

	if data.ccmMigration && data.csiCloudConfig != nil {
		creators = append(creators, csimigration.ValidatingwebhookConfigurationCreator(data.caCert.Cert, metav1.NamespaceSystem, resources.VsphereCSIMigrationWebhookConfigurationWebhookName))
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingWebhookConfigurations: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileServices(ctx context.Context) error {
	creatorsKubeSystem := []reconciling.NamedServiceCreatorGetter{
		metricsserver.ExternalNameServiceCreator(r.namespace),
		coredns.ServiceCreator(r.dnsClusterIP),
	}

	if err := reconciling.ReconcileServices(ctx, creatorsKubeSystem, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services in kube-system namespace: %v", err)
	}

	// Kubernetes Dashboard and related resources
	creators := []reconciling.NamedServiceCreatorGetter{
		kubernetesdashboard.ServiceCreator(),
	}
	if err := reconciling.ReconcileServices(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services in namespace %s: %v", kubernetesdashboard.Namespace, err)
	}

	// OPA related resources
	if r.opaIntegration {
		creators := []reconciling.NamedServiceCreatorGetter{
			gatekeeper.ServiceCreator(),
		}
		if err := reconciling.ReconcileServices(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Services in namespace %s: %v", resources.GatekeeperNamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileConfigMaps(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		machinecontroller.ClusterInfoConfigMapCreator(r.clusterURL.String(), data.caCert.Cert),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespacePublic, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-public namespace: %v", err)
	}

	if len(r.tunnelingAgentIP) > 0 {
		creators = []reconciling.NamedConfigMapCreatorGetter{
			openvpn.ClientConfigConfigMapCreator(r.tunnelingAgentIP.String(), r.openvpnServerPort),
			cabundle.ConfigMapCreator(r.caBundle),
			envoyagent.ConfigMapCreator(envoyagent.Config{
				AdminPort: 9902,
				ProxyHost: r.clusterURL.Hostname(),
				ProxyPort: 8088,
				Listeners: []envoyagent.Listener{
					{
						BindAddress: r.tunnelingAgentIP.String(),
						BindPort:    r.openvpnServerPort,
						Authority:   net.JoinHostPort(fmt.Sprintf("openvpn-server.%s.svc.cluster.local", r.namespace), "1194"),
					},
					{
						BindAddress: r.tunnelingAgentIP.String(),
						BindPort:    r.kasSecurePort,
						Authority:   net.JoinHostPort(fmt.Sprintf("apiserver-external.%s.svc.cluster.local", r.namespace), "443"),
					},
				},
			}),
		}
	} else {
		creators = []reconciling.NamedConfigMapCreatorGetter{
			openvpn.ClientConfigConfigMapCreator(r.clusterURL.Hostname(), r.openvpnServerPort),
			cabundle.ConfigMapCreator(r.caBundle),
		}
	}

	creators = append(creators, coredns.ConfigMapCreator())

	if r.nodeLocalDNSCache {
		creators = append(creators, nodelocaldns.ConfigMapCreator(r.dnsClusterIP))
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-system namespace: %v", err)
	}

	if r.userClusterMLA.Monitoring {
		customScrapeConfigs, err := r.getUserClusterPrometheusCustomScrapeConfigs(ctx)
		if err != nil {
			return fmt.Errorf("failed to get user cluster prometheus custom scrape configs: %w", err)
		}
		creators = []reconciling.NamedConfigMapCreatorGetter{
			userclusterprometheus.ConfigMapCreator(userclusterprometheus.Config{
				MLAGatewayURL:       r.userClusterMLA.MLAGatewayURL + "/api/v1/push",
				TLSCertFile:         fmt.Sprintf("%s/%s", resources.UserClusterPrometheusClientCertMountPath, resources.UserClusterPrometheusClientCertSecretKey),
				TLSKeyFile:          fmt.Sprintf("%s/%s", resources.UserClusterPrometheusClientCertMountPath, resources.UserClusterPrometheusClientKeySecretKey),
				TLSCACertFile:       fmt.Sprintf("%s/%s", resources.UserClusterPrometheusClientCertMountPath, resources.MLAGatewayCACertKey),
				CustomScrapeConfigs: customScrapeConfigs,
			}),
		}
		if err := reconciling.ReconcileConfigMaps(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ConfigMap in namespace %s: %v", resources.UserClusterMLANamespace, err)
		}
	}
	return nil
}

func (r *reconciler) reconcileSecrets(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedSecretCreatorGetter{
		openvpn.ClientCertificate(data.openVPNCACert),
		cloudcontroller.CloudConfig(data.cloudConfig, resources.CloudConfigSecretName),
	}

	if data.csiCloudConfig != nil {
		creators = append(creators, cloudcontroller.CloudConfig(data.csiCloudConfig, resources.CSICloudConfigSecretName))
		if data.ccmMigration {
			creators = append(creators, csimigration.TLSServingCertificateCreator(data.caCert))
		}
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.SecretCreator(data.userSSHKeys))
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in kube-system Namespace: %v", err)
	}

	// Kubernetes Dashboard and related resources
	creators = []reconciling.NamedSecretCreatorGetter{
		kubernetesdashboard.KeyHolderSecretCreator(),
		kubernetesdashboard.CsrfTokenSecretCreator(),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in namespace %s: %v", kubernetesdashboard.Namespace, err)
	}

	// OPA relate resources
	if r.opaIntegration {
		creators = []reconciling.NamedSecretCreatorGetter{
			gatekeeper.SecretCreator(),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %v", resources.GatekeeperNamespace, err)
		}
	}

	if r.userClusterMLA.Monitoring {
		creators = []reconciling.NamedSecretCreatorGetter{
			userclusterprometheus.ClientCertificateCreator(data.mlaGatewayCACert),
		}
		if err := reconciling.ReconcileSecrets(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %v", resources.UserClusterMLANamespace, err)
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
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %v", resources.UserClusterMLANamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileDaemonSet(ctx context.Context, data reconcileData) error {
	var dsCreators []reconciling.NamedDaemonSetCreatorGetter

	if r.nodeLocalDNSCache {
		dsCreators = append(dsCreators, nodelocaldns.DaemonSetCreator(r.registryWithOverwrite))
	}

	if r.userSSHKeyAgent {
		dsCreators = append(dsCreators, usersshkeys.DaemonSetCreator(r.versions))
	}

	if len(r.tunnelingAgentIP) > 0 {
		dsCreators = append(dsCreators, envoyagent.DaemonSetCreator(r.tunnelingAgentIP, r.versions, r.registryWithOverwrite))
	}

	if err := reconciling.ReconcileDaemonSets(ctx, dsCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the DaemonSet: %v", err)
	}

	if r.userClusterMLA.Logging {
		dsCreators = []reconciling.NamedDaemonSetCreatorGetter{
			promtail.DaemonSetCreator(data.loggingRequirements, r.registryWithOverwrite),
		}
		if err := reconciling.ReconcileDaemonSets(ctx, dsCreators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile the DaemonSet: %v", err)
		}
	}
	return nil
}

func (r *reconciler) reconcileNamespaces(ctx context.Context) error {

	creators := []reconciling.NamedNamespaceCreatorGetter{
		kubernetesdashboard.NamespaceCreator,
		cloudinitsettings.NamespaceCreator,
	}
	if r.opaIntegration {
		creators = append(creators, gatekeeper.NamespaceCreator)
		creators = append(creators, gatekeeper.KubeSystemLabeler)
	}
	if r.userClusterMLA.Logging || r.userClusterMLA.Monitoring {
		creators = append(creators, mla.NamespaceCreator)
	}
	if err := reconciling.ReconcileNamespaces(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile namespaces: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileDeployments(ctx context.Context, data reconcileData) error {
	// Kubernetes Dashboard and related resources
	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubernetesdashboard.DeploymentCreator(r.registryWithOverwrite),
	}
	if err := reconciling.ReconcileDeployments(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in namespace %s: %v", kubernetesdashboard.Namespace, err)
	}

	kubeSystemCreators := []reconciling.NamedDeploymentCreatorGetter{
		coredns.DeploymentCreator(r.clusterSemVer, r.registryWithOverwrite),
	}

	if err := reconciling.ReconcileDeployments(ctx, kubeSystemCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in namespace %s: %v", metav1.NamespaceSystem, err)
	}

	// OPA related resources
	if r.opaIntegration {
		creators := []reconciling.NamedDeploymentCreatorGetter{
			gatekeeper.ControllerDeploymentCreator(r.opaEnableMutation, r.registryWithOverwrite),
			gatekeeper.AuditDeploymentCreator(r.registryWithOverwrite),
		}

		if err := reconciling.ReconcileDeployments(ctx, creators, resources.GatekeeperNamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %v", resources.GatekeeperNamespace, err)
		}
	}

	if r.userClusterMLA.Monitoring {
		creators := []reconciling.NamedDeploymentCreatorGetter{
			userclusterprometheus.DeploymentCreator(data.monitoringRequirements, r.registryWithOverwrite),
		}
		if err := reconciling.ReconcileDeployments(ctx, creators, resources.UserClusterMLANamespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Deployments in namespace %s: %v", resources.UserClusterMLANamespace, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileKonnectivityDeployments(ctx context.Context) error {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		konnectivity.DeploymentCreator(r.clusterURL.Hostname(), r.registryWithOverwrite),
	}
	if err := reconciling.ReconcileDeployments(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in namespace %s: %v", metav1.NamespaceSystem, err)
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
			return fmt.Errorf("failed to reconcile PodDisruptionBudgets in namespace %s: %v", resources.GatekeeperNamespace, err)
		}
	}
	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %v", err)
	}
	return nil
}

type reconcileData struct {
	caCert           *triple.KeyPair
	openVPNCACert    *resources.ECDSAKeyPair
	mlaGatewayCACert *resources.ECDSAKeyPair
	userSSHKeys      map[string][]byte
	cloudConfig      []byte
	// csiCloudConfig is currently used only by vSphere, whose needs it to properly configure the external CSI driver
	csiCloudConfig         []byte
	ccmMigration           bool
	monitoringRequirements *corev1.ResourceRequirements
	loggingRequirements    *corev1.ResourceRequirements
}

func (r *reconciler) ensureOPAIntegrationIsRemoved(ctx context.Context) error {
	for _, resource := range gatekeeper.GetResourcesToRemoveOnDelete() {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OPA integration is removed/not present: %v", err)
		}
	}

	return nil
}

func (r *reconciler) ensureOPAExperimentalMutationWebhookIsRemoved(ctx context.Context) error {
	if err := r.Client.Delete(ctx, &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperMutatingWebhookConfigurationName,
		}}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to remove Mutation Webhook: %v", err)
	}
	return nil
}

func (r *reconciler) healthCheck(ctx context.Context) error {
	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx,
		types.NamespacedName{Namespace: r.namespace, Name: strings.TrimPrefix(r.namespace, "cluster-")}, cluster); err != nil {
		return fmt.Errorf("failed getting cluster for cluster health check: %v", err)
	}
	oldCluster := cluster.DeepCopy()

	ctrlHealth, auditHealth, err := r.getGatekeeperHealth(ctx)
	if err != nil {
		return err
	}

	cluster.Status.ExtendedHealth.GatekeeperController = ctrlHealth
	cluster.Status.ExtendedHealth.GatekeeperAudit = auditHealth

	if oldCluster.Status.ExtendedHealth != cluster.Status.ExtendedHealth {
		if err := r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("error patching cluster health status: %v", err)
		}
	}
	return nil
}

func (r *reconciler) getGatekeeperHealth(ctx context.Context) (
	ctlrHealth kubermaticv1.HealthStatus, auditHealth kubermaticv1.HealthStatus, err error) {

	ctlrHealth, err = resources.HealthyDeployment(ctx,
		r.Client,
		types.NamespacedName{Namespace: resources.GatekeeperNamespace, Name: resources.GatekeeperControllerDeploymentName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, kubermaticv1.HealthStatusDown,
			fmt.Errorf("failed to get dep health %q: %v", resources.GatekeeperControllerDeploymentName, err)
	}

	auditHealth, err = resources.HealthyDeployment(ctx,
		r.Client,
		types.NamespacedName{Namespace: resources.GatekeeperNamespace, Name: resources.GatekeeperAuditDeploymentName},
		1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, kubermaticv1.HealthStatusDown,
			fmt.Errorf("failed to get dep health %q: %v", resources.GatekeeperAuditDeploymentName, err)
	}
	return ctlrHealth, auditHealth, nil
}

func (r *reconciler) ensurePromtailIsRemoved(ctx context.Context) error {
	for _, resource := range promtail.ResourcesOnDeletion() {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure promtail is removed/not present: %v", err)
		}
	}
	return nil
}

func (r *reconciler) ensureUserClusterPrometheusIsRemoved(ctx context.Context) error {
	for _, resource := range userclusterprometheus.ResourcesOnDeletion() {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure user cluster prometheus is removed/not present: %v", err)
		}
	}
	return nil
}

func (r *reconciler) ensureMLAIsRemoved(ctx context.Context) error {
	for _, resource := range mla.ResourcesOnDeletion() {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure mla is removed/not present: %v", err)
		}
	}
	return nil
}

func (r *reconciler) getUserClusterPrometheusCustomScrapeConfigs(ctx context.Context) (string, error) {
	if r.userClusterMLA.PrometheusScrapeConfigPrefix == "" {
		return "", nil
	}
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, ctrlruntimeclient.InNamespace(resources.UserClusterMLANamespace)); err != nil {
		return "", fmt.Errorf("failed to list the configmap: %w", err)
	}
	customScrapeConfigs := ""
	for _, configMap := range configMapList.Items {
		if !strings.HasPrefix(configMap.GetName(), r.userClusterMLA.PrometheusScrapeConfigPrefix) {
			continue
		}
		for _, v := range configMap.Data {
			customScrapeConfigs += strings.TrimSpace(v) + "\n"
		}
	}
	return customScrapeConfigs, nil
}
