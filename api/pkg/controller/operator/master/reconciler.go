package master

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/master/resources/kubermatic"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// master cluster.
type Reconciler struct {
	ctrlruntimeclient.Client

	log        *zap.SugaredLogger
	recorder   record.EventRecorder
	scheme     *runtime.Scheme
	workerName string
	ctx        context.Context
	versions   common.Versions
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// find the requested configuration
	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := r.Get(r.ctx, request.NamespacedName, config); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not get KubermaticConfiguration %q: %v", request, err)
	}

	identifier, err := cache.MetaNamespaceKeyFunc(config)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to determine string key for KubermaticConfiguration: %v", err)
	}

	logger := r.log.With("config", identifier)

	// create a copy of the configuration with default values applied
	defaulted, err := common.DefaultConfiguration(config, logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to apply defaults: %v", err)
	}

	err = r.reconcile(defaulted, logger)
	if err != nil {
		r.recorder.Event(config, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Kubermatic configuration")

	if err := r.reconcileNamespaces(config, logger); err != nil {
		return err
	}

	if err := r.reconcileServiceAccounts(config, logger); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(config, logger); err != nil {
		return err
	}

	if err := r.reconcileSecrets(config, logger); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(config, logger); err != nil {
		return err
	}

	if err := r.reconcileDeployments(config, logger); err != nil {
		return err
	}

	if err := r.reconcilePodDisruptionBudgets(config, logger); err != nil {
		return err
	}

	if err := r.reconcileServices(config, logger); err != nil {
		return err
	}

	if err := r.reconcileIngresses(config, logger); err != nil {
		return err
	}

	if err := r.reconcileCertificates(config, logger); err != nil {
		return err
	}

	if err := r.reconcileAdmissionWebhooks(config, logger); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) reconcileNamespaces(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Namespaces")

	creators := []reconciling.NamedNamespaceCreatorGetter{
		common.NamespaceCreator(config),
	}

	if err := reconciling.ReconcileNamespaces(r.ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Namespaces: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMaps(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapCreatorGetter{
		kubermatic.UIConfigConfigMapCreator(config),
	}

	if err := reconciling.ReconcileConfigMaps(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Secrets")

	creators := []reconciling.NamedSecretCreatorGetter{
		common.DockercfgSecretCreator(config),
		common.DexCASecretCreator(config),
		common.SeedWebhookServingCASecretCreator(config),
		common.SeedWebhookServingCertSecretCreator(config, r.Client),
		common.MasterFilesSecretCreator(config),
		kubermatic.PresetsSecretCreator(config),
	}

	if err := reconciling.ReconcileSecrets(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileServiceAccounts(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ServiceAccounts")

	creators := []reconciling.NamedServiceAccountCreatorGetter{
		kubermatic.ServiceAccountCreator(config),
	}

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		kubermatic.ClusterRoleBindingCreator(config),
	}

	if err := reconciling.ReconcileClusterRoleBindings(r.ctx, creators, "", r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileDeployments(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Deployments")

	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubermatic.APIDeploymentCreator(config, r.workerName, r.versions),
		kubermatic.UIDeploymentCreator(config, r.versions),
		kubermatic.MasterControllerManagerDeploymentCreator(config, r.workerName, r.versions),
	}

	modifiers := []reconciling.ObjectModifier{
		common.OwnershipModifierFactory(config, r.scheme),
		common.VolumeRevisionLabelsModifierFactory(r.ctx, r.Client),
	}

	if err := reconciling.ReconcileDeployments(r.ctx, creators, config.Namespace, r.Client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcilePodDisruptionBudgets(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling PodDisruptionBudgets")

	creators := []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		kubermatic.APIPDBCreator(config),
		kubermatic.UIPDBCreator(config),
		kubermatic.MasterControllerManagerPDBCreator(config),
	}

	if err := reconciling.ReconcilePodDisruptionBudgets(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileServices(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Services")

	creators := []reconciling.NamedServiceCreatorGetter{
		kubermatic.APIServiceCreator(config),
		kubermatic.UIServiceCreator(config),
		common.SeedAdmissionServiceCreator(config, r.Client),
	}

	if err := reconciling.ReconcileServices(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Services: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileIngresses(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Ingresses")

	creators := []reconciling.NamedIngressCreatorGetter{
		kubermatic.IngressCreator(config),
	}

	if err := reconciling.ReconcileIngresses(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Ingresses: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileCertificates(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Certificates")

	creators := []reconciling.NamedCertificateCreatorGetter{
		kubermatic.CertificateCreator(config),
	}

	if err := reconciling.ReconcileCertificates(r.ctx, creators, config.Namespace, r.Client, common.OwnershipModifierFactory(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Certificates: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileAdmissionWebhooks(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling AdmissionWebhooks")

	creators := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{
		common.SeedAdmissionWebhookCreator(config, r.Client),
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(r.ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile AdmissionWebhooks: %v", err)
	}

	return nil
}
