package usercluster

import (
	"context"
	"fmt"

	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/clusterautoscaler"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/controller-manager"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/dnat-controller"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/kube-state-metrics"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/machine-controller"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/openshift"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/prometheus"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/scheduler"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/user-auth"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) reconcile(ctx context.Context) error {
	// Must be first because of openshift
	if err := r.ensureAPIServices(ctx); err != nil {
		return err
	}

	// We need to reconcile namespaces and services next to make sure
	// the openshift apiservices become available ASAP
	if err := r.reconcileNamespaces(ctx); err != nil {
		return err
	}

	if err := r.reconcileUnstructured(ctx); err != nil {
		return err
	}

	if err := r.reconcileServices(ctx); err != nil {
		return err
	}

	if err := r.reconcileServiceAcconts(ctx); err != nil {
		return err
	}

	if err := r.reconcileRoles(ctx); err != nil {
		return err
	}

	if err := r.reconcileRoleBindings(ctx); err != nil {
		return err
	}

	if err := r.reconcileClusterRoles(ctx); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(ctx); err != nil {
		return err
	}

	if err := r.reconcileCRDs(ctx); err != nil {
		return err
	}

	if err := r.reconcileMutatingWebhookConfigurations(ctx); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(ctx); err != nil {
		return err
	}

	if err := r.reconcileSecrets(ctx); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) ensureAPIServices(ctx context.Context) error {
	creators := []reconciling.NamedAPIServiceCreatorGetter{}
	caCert := certutil.EncodeCertPEM(r.caCert)
	if r.openshift {
		openshiftAPIServiceCreators, err := openshift.GetAPIServicesForOpenshiftVersion(r.version, caCert)
		if err != nil {
			return fmt.Errorf("failed to get openshift apiservice creators: %v", err)
		}
		creators = append(creators, openshiftAPIServiceCreators...)
	} else {
		creators = append(creators, metricsserver.APIServiceCreator(caCert))
	}

	if err := reconciling.ReconcileAPIServices(ctx, creators, metav1.NamespaceNone, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile APIServices: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileServiceAcconts(ctx context.Context) error {
	creators := []reconciling.NamedServiceAccountCreatorGetter{
		userauth.ServiceAccountCreator(),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", metav1.NamespaceSystem, err)
	}
	return nil
}

func (r *reconciler) reconcileRoles(ctx context.Context) error {
	// kube-system
	creators := []reconciling.NamedRoleCreatorGetter{
		machinecontroller.KubeSystemRoleCreator(),
		clusterautoscaler.KubeSystemRoleCreator(),
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

	if err := reconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile CustomResourceDefinitions: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileMutatingWebhookConfigurations(ctx context.Context) error {
	creators := []reconciling.NamedMutatingWebhookConfigurationCreatorGetter{
		machinecontroller.MutatingwebhookConfigurationCreator(r.caCert, r.namespace),
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile MutatingWebhookConfigurations: %v", err)
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
	if r.openshift {
		if err := reconciling.ReconcileServices(ctx, []reconciling.NamedServiceCreatorGetter{openshift.APIServicecreatorGetterFactory(r.namespace)}, "openshift-apiserver", r.Client); err != nil {
			return fmt.Errorf("failed to reconcile services in the openshift-apiserver namespace: %v", err)
		}
	}
	return nil
}

func (r *reconciler) reconcileConfigMaps(ctx context.Context) error {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		machinecontroller.ClusterInfoConfigMapCreator(r.clusterURL.String(), r.caCert),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespacePublic, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-public namespace: %v", err)
	}

	creators = []reconciling.NamedConfigMapCreatorGetter{
		openvpn.ClientConfigConfigMapCreator(r.clusterURL.Hostname(), r.openvpnServerPort),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-system namespace: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileSecrets(ctx context.Context) error {
	creators := []reconciling.NamedSecretCreatorGetter{
		openvpn.ClientCertificate(r.openVPNCA),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in kue-system Namespace: %v", err)
	}

	return nil
}

func (r *reconciler) reconcileNamespaces(ctx context.Context) error {
	if !r.openshift {
		return nil
	}
	creators := []reconciling.NamedNamespaceCreatorGetter{openshift.APIServerNSCreatorGetter,
		openshift.ControllerManagerNSCreatorGetter,
		openshift.KubeSchedulerNSCreatorGetter,
		openshift.NetworkOperatorNSGetter,
		openshift.OpenshiftCloudCredentialOperatorNSGetter,
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

	creators := []reconciling.NamedUnstructuredCreatorGetter{
		openshift.InfrastructureCreatorGetter(r.platform),
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
