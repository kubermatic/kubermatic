package master

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/master/resources/kubermatic"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

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

	// silently ignore other worker names
	if config.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return reconcile.Result{}, nil
	}

	identifier, err := cache.MetaNamespaceKeyFunc(config)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to determine string key for KubermaticConfiguration: %v", err)
	}

	logger := r.log.With("config", identifier)

	defaulted, err := r.defaultConfiguration(config, logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to apply defaults: %v", err)
	}

	if defaulted {
		logger.Info("Applied default values to configuration")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, r.reconcile(config, logger)
}

func (r *Reconciler) reconcile(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Kubermatic configuration")

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
		common.SeedWebhookServingCertSecretCreator(config),
	}

	if len(config.Spec.MasterFiles) > 0 {
		creators = append(creators, common.MasterFilesSecretCreator(config))
	}

	if config.Spec.UI.Presets != "" {
		creators = append(creators, kubermatic.PresetsSecretCreator(config))
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
		kubermatic.APIDeploymentCreator(config),
		kubermatic.UIDeploymentCreator(config),
		kubermatic.MasterControllerManagerDeploymentCreator(config),
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
