package usercluster

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/controller-manager"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/dnat-controller"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/kube-state-metrics"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/machine-controller"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/prometheus"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/scheduler"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster/resources/user-auth"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile creates, updates, or deletes Kubernetes resources to match the desired state.
func (r *reconciler) reconcile(ctx context.Context) error {
	if err := r.ensureAPIServices(ctx); err != nil {
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

	if err := r.reconcileServices(ctx); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(ctx); err != nil {
		return err
	}

	return nil
}

// GetAPIServiceCreators returns a list of APIServiceCreator
func (r *reconciler) GetAPIServiceCreators() []resources.APIServiceCreator {
	var creators []resources.APIServiceCreator
	if !r.openshift {
		creators = append(creators, metricsserver.APIService)
	}
	return creators
}

func (r *reconciler) ensureAPIServices(ctx context.Context) error {
	creators := r.GetAPIServiceCreators()

	for _, create := range creators {
		apiService, err := create(nil)
		if err != nil {
			return fmt.Errorf("failed to build APIService: %v", err)
		}

		existing := &apiregistrationv1beta1.APIService{}
		err = r.Get(ctx, controllerclient.ObjectKey{Namespace: apiService.Namespace, Name: apiService.Name}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.Create(ctx, apiService); err != nil {
					return fmt.Errorf("failed to create APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
				}
				glog.V(2).Infof("Created a new APIService %s in namespace %s", apiService.Name, apiService.Namespace)
				return nil
			}
			return err
		}

		apiService, err = create(existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build APIService : %v", err)
		}
		if equality.Semantic.DeepEqual(apiService, existing) {
			continue
		}

		if err := r.Update(ctx, apiService); err != nil {
			return fmt.Errorf("failed to update APIService %s in namespace %s: %v", apiService.Name, apiService.Namespace, err)
		}
		glog.V(4).Infof("Updated the APIService %s in namespace %s", apiService.Name, apiService.Namespace)
	}

	return nil
}

func (r *reconciler) reconcileServiceAcconts(ctx context.Context) error {
	creators := []resources.NamedServiceAccountCreatorGetter{
		userauth.ServiceAccountCreator(),
	}

	if err := resources.ReconcileServiceAccounts(creators, metav1.NamespaceSystem, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", metav1.NamespaceSystem, err)
	}
	return nil
}

func (r *reconciler) reconcileRoles(ctx context.Context) error {
	// kube-system
	creators := []resources.NamedRoleCreatorGetter{
		machinecontroller.KubeSystemRoleCreator(),
	}

	if err := resources.ReconcileRoles(creators, metav1.NamespaceSystem, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", metav1.NamespaceSystem, err)
	}

	// kube-public
	creators = []resources.NamedRoleCreatorGetter{
		machinecontroller.ClusterInfoReaderRoleCreator(),
		machinecontroller.KubePublicRoleCreator(),
	}

	if err := resources.ReconcileRoles(creators, metav1.NamespacePublic, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", metav1.NamespacePublic, err)
	}

	// default
	creators = []resources.NamedRoleCreatorGetter{
		machinecontroller.EndpointReaderRoleCreator(),
	}

	if err := resources.ReconcileRoles(creators, metav1.NamespaceDefault, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", metav1.NamespaceDefault, err)
	}

	return nil
}

func (r *reconciler) reconcileRoleBindings(ctx context.Context) error {
	// kube-system
	creators := []resources.NamedRoleBindingCreatorGetter{
		machinecontroller.KubeSystemRoleBindingCreator(),
		metricsserver.RolebindingAuthReaderCreator(),
		scheduler.RoleBindingAuthDelegator(),
		controllermanager.RoleBindingAuthDelegator(),
	}
	if err := resources.ReconcileRoleBindings(creators, metav1.NamespaceSystem, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in kube-system Namespace: %v", err)
	}

	// kube-public
	creators = []resources.NamedRoleBindingCreatorGetter{
		machinecontroller.KubePublicRoleBindingCreator(),
		machinecontroller.ClusterInfoAnonymousRoleBindingCreator(),
	}
	if err := resources.ReconcileRoleBindings(creators, metav1.NamespacePublic, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in kube-public Namespace: %v", err)
	}

	// Default
	creators = []resources.NamedRoleBindingCreatorGetter{
		machinecontroller.DefaultRoleBindingCreator(),
	}
	if err := resources.ReconcileRoleBindings(creators, metav1.NamespaceDefault, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Roles in default Namespace: %v", err)
	}

	return nil
}

func (r *reconciler) reconcileClusterRoles(ctx context.Context) error {
	creators := []resources.NamedClusterRoleCreatorGetter{
		kubestatemetrics.ClusterRoleCreator(),
		prometheus.ClusterRoleCreator(),
		machinecontroller.ClusterRoleCreator(),
		dnatcontroller.ClusterRoleCreator(),
		metricsserver.ClusterRoleCreator(),
	}

	if err := resources.ReconcileClusterRoles(creators, "", r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileClusterRoleBindings(ctx context.Context) error {
	creators := []resources.NamedClusterRoleBindingCreatorGetter{
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
	}

	if err := resources.ReconcileClusterRoleBindings(creators, "", r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileCRDs(ctx context.Context) error {
	creators := []resources.NamedCustomResourceDefinitionCreatorGetter{
		machinecontroller.MachineCRDCreator(),
		machinecontroller.MachineSetCRDCreator(),
		machinecontroller.MachineDeploymentCRDCreator(),
		machinecontroller.ClusterCRDCreator(),
	}

	if err := resources.ReconcileCustomResourceDefinitions(creators, "", r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile CustomResourceDefinitions: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileMutatingWebhookConfigurations(ctx context.Context) error {
	creators := []resources.NamedMutatingWebhookConfigurationCreatorGetter{
		machinecontroller.MutatingwebhookConfigurationCreator(r.caCert, r.namespace),
	}

	if err := resources.ReconcileMutatingWebhookConfigurations(creators, "", r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile MutatingWebhookConfigurations: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileServices(ctx context.Context) error {
	creators := []resources.NamedServiceCreatorGetter{
		metricsserver.ExternalNameServiceCreator(r.namespace),
	}

	if err := resources.ReconcileServices(creators, metav1.NamespaceSystem, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile Services in kube-system namespace: %v", err)
	}
	return nil
}

func (r *reconciler) reconcileConfigMaps(ctx context.Context) error {
	creators := []resources.NamedConfigMapCreatorGetter{
		machinecontroller.ClusterInfoConfigMapCreator(r.clusterURL.String(), r.caCert),
	}

	if err := resources.ReconcileConfigMaps(creators, metav1.NamespacePublic, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-public namespace: %v", err)
	}

	creators = []resources.NamedConfigMapCreatorGetter{
		openvpn.ClientConfigConfigMapCreator(r.clusterURL.Hostname(), r.openvpnServerPort),
	}

	if err := resources.ReconcileConfigMaps(creators, metav1.NamespaceSystem, r.Client, r.cache); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in kube-system namespace: %v", err)
	}
	return nil
}
