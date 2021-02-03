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

	openshiftresources "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/openshift/resources"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/clusterautoscaler"
	controllermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/controller-manager"
	coredns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/core-dns"
	dnatcontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/dnat-controller"
	envoyagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/envoy-agent"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/gatekeeper"
	kubestatemetrics "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kube-state-metrics"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubernetes-dashboard"
	machinecontroller "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/machine-controller"
	metricsserver "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/metrics-server"
	nodelocaldns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/node-local-dns"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/openshift"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/openvpn"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/prometheus"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/scheduler"
	systembasicuser "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/system-basic-user"
	userauth "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/user-auth"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/usersshkeys"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	cloudConfig, err := r.cloudConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cloudConfig: %v", err)
	}
	data := reconcileData{
		caCert:        caCert,
		openVPNCACert: openVPNCACert,
		userSSHKeys:   userSSHKeys,
		cloudConfig:   cloudConfig,
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

	if err := r.reconcileServiceAcconts(ctx); err != nil {
		return err
	}

	if err := r.reconcileUnstructured(ctx); err != nil {
		return err
	}

	if err := r.reconcilePodDisruptionBudgets(ctx); err != nil {
		return err
	}

	if err := r.reconcileDeployments(ctx); err != nil {
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

	if err := r.reconcileMutatingWebhookConfigurations(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileSecrets(ctx, data); err != nil {
		return err
	}

	if err := r.reconcileDaemonSet(ctx); err != nil {
		return err
	}

	if err := r.reconcileValidatingWebhookConfigurations(ctx, data); err != nil {
		return err
	}

	// Try to delete OPA integration deployment if its present
	if !r.opaIntegration {
		if err := r.ensureOPAIntegrationIsRemoved(ctx); err != nil {
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

	if r.openshift {
		openshiftAPIServiceCreators, err := openshift.GetAPIServicesForOpenshiftVersion(r.version, caCert)
		if err != nil {
			return fmt.Errorf("failed to get openshift apiservice creators: %v", err)
		}
		creators = append(creators, openshiftAPIServiceCreators...)
	}

	if err := reconciling.ReconcileAPIServices(ctx, creators, metav1.NamespaceNone, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile APIServices: %v", err)
	}

	return nil
}

func (r *reconciler) reconcileServiceAcconts(ctx context.Context) error {
	creators := []reconciling.NamedServiceAccountCreatorGetter{
		userauth.ServiceAccountCreator(),
		usersshkeys.ServiceAccountCreator(),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.ServiceAccountCreator())
	}

	if r.openshift {
		creators = append(creators, openshift.TokenOwnerServiceAccount)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", metav1.NamespaceSystem, err)
	}

	if !r.openshift {
		// Kubernetes Dashboard and related resources
		creators = []reconciling.NamedServiceAccountCreatorGetter{
			kubernetesdashboard.ServiceAccountCreator(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", kubernetesdashboard.Namespace, err)
		}
		creators = []reconciling.NamedServiceAccountCreatorGetter{
			coredns.ServiceAccountCreator(),
			nodelocaldns.ServiceAccountCreator(),
		}
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", metav1.NamespaceSystem, err)
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

	if !r.openshift {
		// Kubernetes Dashboard and related resources
		creators = []reconciling.NamedRoleCreatorGetter{
			kubernetesdashboard.RoleCreator(),
		}
		if err := reconciling.ReconcileRoles(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", kubernetesdashboard.Namespace, err)
		}
	}

	if r.openshift {
		namespacedName, roleCreator := openshiftresources.MachineControllerRole()
		if err := reconciling.EnsureNamedObject(ctx,
			namespacedName,
			reconciling.RoleObjectWrapper(roleCreator),
			r.Client,
			&rbacv1.Role{},
			false); err != nil {
			return fmt.Errorf("failed to reconcile Role %q: %v", namespacedName.String(), err)
		}

		// openshift-kube-scheduler
		creators := []reconciling.NamedRoleCreatorGetter{openshift.KubeSchedulerRoleCreatorGetter}
		if err := reconciling.ReconcileRoles(ctx, creators, "openshift-kube-scheduler", r.Client); err != nil {
			return err
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

	if !r.openshift {
		// Kubernetes Dashboard and related resources
		creators = []reconciling.NamedRoleBindingCreatorGetter{
			kubernetesdashboard.RoleBindingCreator(),
		}
		if err := reconciling.ReconcileRoleBindings(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile RoleBindings in the namespace: %s: %v", kubernetesdashboard.Namespace, err)
		}
	}

	if r.openshift {
		namespacedName, roleBindingCreator := openshiftresources.MachineControllerRoleBinding()
		if err := reconciling.EnsureNamedObject(ctx,
			namespacedName,
			reconciling.RoleBindingObjectWrapper(roleBindingCreator),
			r.Client,
			&rbacv1.RoleBinding{},
			false); err != nil {
			return fmt.Errorf("failed to reconcile RoleBinding %q: %v", namespacedName.String(), err)
		}

		// openshift-kube-scheduler
		creators := []reconciling.NamedRoleBindingCreatorGetter{openshift.KubeSchedulerRoleBindingCreatorGetter}
		if err := reconciling.ReconcileRoleBindings(ctx, creators, "openshift-kube-scheduler", r.Client); err != nil {
			return err
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
	}

	if !r.openshift {
		creators = append(creators,
			[]reconciling.NamedClusterRoleCreatorGetter{
				kubernetesdashboard.ClusterRoleCreator(),
				coredns.ClusterRoleCreator(),
			}...)
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
	}

	if r.openshift {
		creators = append(creators, openshift.TokenOwnerServiceAccountClusterRoleBinding)
	} else {
		creators = append(creators,
			[]reconciling.NamedClusterRoleBindingCreatorGetter{
				kubernetesdashboard.ClusterRoleBindingCreator(),
				coredns.ClusterRoleBindingCreator(),
			}...)
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
		creators = append(creators, gatekeeper.ConfigCRDCreator())
		creators = append(creators, gatekeeper.ConstraintTemplateCRDCreator())
	}

	if err := reconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile CustomResourceDefinitions: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileMutatingWebhookConfigurations(
	ctx context.Context,
	data reconcileData,
) error {
	creators := []reconciling.NamedMutatingWebhookConfigurationCreatorGetter{
		machinecontroller.MutatingwebhookConfigurationCreator(data.caCert.Cert, r.namespace),
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile MutatingWebhookConfigurations: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileValidatingWebhookConfigurations(
	ctx context.Context,
	data reconcileData,
) error {
	creators := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{}
	if r.opaIntegration {
		creators = append(creators, gatekeeper.ValidatingWebhookConfigurationCreator(data.caCert.Cert, r.namespace))
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingWebhookConfigurations: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileServices(ctx context.Context) error {
	creatorsKubeSystem := []reconciling.NamedServiceCreatorGetter{
		metricsserver.ExternalNameServiceCreator(r.namespace),
	}

	if err := reconciling.ReconcileServices(ctx, creatorsKubeSystem, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services in kube-system namespace: %v", err)
	}

	if !r.openshift {
		// Kubernetes Dashboard and related resources
		creators := []reconciling.NamedServiceCreatorGetter{
			kubernetesdashboard.ServiceCreator(),
		}
		if err := reconciling.ReconcileServices(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Services in namespace %s: %v", kubernetesdashboard.Namespace, err)
		}
		creatorsKubeSystem = []reconciling.NamedServiceCreatorGetter{
			coredns.ServiceCreator(r.dnsClusterIP),
		}
		if err := reconciling.ReconcileServices(ctx, creatorsKubeSystem, metav1.NamespaceSystem, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Services in kube-system namespace: %v", err)
		}
	}

	if r.openshift {
		if err := reconciling.ReconcileServices(ctx, []reconciling.NamedServiceCreatorGetter{openshift.APIServicecreatorGetterFactory(r.namespace)}, "openshift-apiserver", r.Client); err != nil {
			return fmt.Errorf("failed to reconcile services in the openshift-apiserver namespace: %v", err)
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
						Authority:   net.JoinHostPort(fmt.Sprintf("apiserver-external.%s.svc.cluster.local", r.namespace), "6443"),
					},
				},
			}),
		}
	} else {
		creators = []reconciling.NamedConfigMapCreatorGetter{
			openvpn.ClientConfigConfigMapCreator(r.clusterURL.Hostname(), r.openvpnServerPort),
		}
	}

	if r.openshift {
		creators = append(creators, openshift.ControlplaneConfigCreator(r.platform))
	} else {
		creators = append(creators,
			coredns.ConfigMapCreator(),
			nodelocaldns.ConfigMapCreator(r.dnsClusterIP),
		)
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-system namespace: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileSecrets(ctx context.Context, data reconcileData) error {
	creators := []reconciling.NamedSecretCreatorGetter{
		openvpn.ClientCertificate(data.openVPNCACert),
		cloudcontroller.CloudConfig(data.cloudConfig),
	}

	if r.userSSHKeyAgent {
		creators = append(creators, usersshkeys.SecretCreator(data.userSSHKeys))
	}

	if r.openshift {
		creators = append(creators, openshift.OAuthBootstrapPasswordCreatorGetter(r.seedClient, r.namespace))
		if r.cloudCredentialSecretTemplate != nil {
			creators = append(creators, openshift.CloudCredentialSecretCreator(*r.cloudCredentialSecretTemplate))
		}
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in kube-system Namespace: %v", err)
	}

	if !r.openshift {
		// Kubernetes Dashboard and related resources
		creators = []reconciling.NamedSecretCreatorGetter{
			kubernetesdashboard.KeyHolderSecretCreator(),
			kubernetesdashboard.CsrfTokenSecretCreator(),
		}

		if err := reconciling.ReconcileSecrets(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
			return fmt.Errorf("failed to reconcile Secrets in namespace %s: %v", kubernetesdashboard.Namespace, err)
		}
	}

	if r.openshift {
		creators = []reconciling.NamedSecretCreatorGetter{openshift.RegistryServingCert(data.caCert)}
		if err := reconciling.ReconcileSecrets(ctx, creators, openshiftresources.RegistryNamespaceName, r.Client); err != nil {
			return fmt.Errorf("failed to create secrets in %q namespace: %v", openshiftresources.RegistryNamespaceName, err)
		}
	}

	return nil
}

func (r *reconciler) reconcileDaemonSet(ctx context.Context) error {
	var dsCreators []reconciling.NamedDaemonSetCreatorGetter

	if r.userSSHKeyAgent {
		dsCreators = append(dsCreators, usersshkeys.DaemonSetCreator(r.versions))
	}

	if !r.openshift {
		dsCreators = append(dsCreators, nodelocaldns.DaemonSetCreator())
	}

	if len(r.tunnelingAgentIP) > 0 {
		dsCreators = append(dsCreators, envoyagent.DaemonSetCreator(r.tunnelingAgentIP, r.versions))
	}

	if err := reconciling.ReconcileDaemonSets(ctx, dsCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the DaemonSet: %v", err)
	}

	return nil
}

func (r *reconciler) reconcileNamespaces(ctx context.Context) error {

	if !r.openshift {
		creators := []reconciling.NamedNamespaceCreatorGetter{
			kubernetesdashboard.NamespaceCreator,
		}
		if r.opaIntegration {
			creators = append(creators, gatekeeper.NamespaceCreator)
		}
		if err := reconciling.ReconcileNamespaces(ctx, creators, "", r.Client); err != nil {
			return fmt.Errorf("failed to reconcile namespaces: %v", err)
		}
		return nil
	}

	creators := []reconciling.NamedNamespaceCreatorGetter{
		openshift.APIServerNSCreatorGetter,
		openshift.ControllerManagerNSCreatorGetter,
		openshift.KubeSchedulerNSCreatorGetter,
		openshift.NetworkOperatorNSGetter,
		openshift.RegistryNSGetter,
		openshift.CloudCredentialOperatorNSGetter,
	}
	if err := reconciling.ReconcileNamespaces(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile namespaces: %v", err)
	}

	return nil
}

func (r *reconciler) reconcileUnstructured(ctx context.Context) error {
	if !r.openshift {
		return nil
	}

	// On the very first reconciliation we don't have a cache yet
	if r.cache == nil {
		return nil
	}

	creators := []reconciling.NamedUnstructuredCreatorGetter{
		openshift.InfrastructureCreatorGetter(r.platform),
		openshift.ClusterVersionCreatorGetter(r.namespace),
		openshift.ConsoleOAuthClientCreator(r.openshiftConsoleCallbackURI),
	}
	r.log.Debug("Reconciling unstructured")
	// The delegatingReader from the `mgr` always redirects request for unstructured.Unstructured
	// to the API even though the cache-backed reader is perfectly capable of creating watches
	// for unstructured.Unstructured: https://github.com/kubernetes-sigs/controller-runtime/issues/615
	// Since using the API is very expensive as we get triggered by almost anything, we construct our
	// own client that uses the cache as reader.
	client := ctrlruntimeclientClient{
		Reader:       r.cache,
		Writer:       r.Client,
		StatusClient: r.Client,
	}
	if err := reconciling.ReconcileUnstructureds(ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile unstructureds: %v", err)
	}
	r.log.Debug("Finished reconciling unstructured")

	return nil
}

type ctrlruntimeclientClient struct {
	ctrlruntimeclient.Reader
	ctrlruntimeclient.Writer
	ctrlruntimeclient.StatusClient
}

func (r *reconciler) reconcileDeployments(ctx context.Context) error {
	if r.openshift {
		return nil
	}
	// Kubernetes Dashboard and related resources
	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubernetesdashboard.DeploymentCreator(),
	}
	if err := reconciling.ReconcileDeployments(ctx, creators, kubernetesdashboard.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in namespace %s: %v", kubernetesdashboard.Namespace, err)
	}

	kubeSystemCreators := []reconciling.NamedDeploymentCreatorGetter{
		coredns.DeploymentCreator(r.clusterSemVer),
	}

	if err := reconciling.ReconcileDeployments(ctx, kubeSystemCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in namespace %s: %v", metav1.NamespaceSystem, err)
	}

	return nil
}

func (r *reconciler) reconcilePodDisruptionBudgets(ctx context.Context) error {
	creators := []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		coredns.PodDisruptionBudgetCreator(),
	}
	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %v", err)
	}
	return nil
}

type reconcileData struct {
	caCert        *triple.KeyPair
	openVPNCACert *resources.ECDSAKeyPair
	userSSHKeys   map[string][]byte
	cloudConfig   []byte
}

func (r *reconciler) ensureOPAIntegrationIsRemoved(ctx context.Context) error {
	for _, resource := range gatekeeper.GetResourcesToRemoveOnDelete() {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OPA integration is removed/not present: %v", err)
		}
	}

	return nil
}
