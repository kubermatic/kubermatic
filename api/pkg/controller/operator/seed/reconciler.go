package seed

import (
	"context"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/seed/resources/kubermatic"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// seed cluster.
type Reconciler struct {
	ctrlruntimeclient.Client

	ctx                context.Context
	log                *zap.SugaredLogger
	namespace          string
	masterClient       ctrlruntimeclient.Client
	seedsGetter        provider.SeedsGetter
	seedClients        map[string]ctrlruntimeclient.Client
	masterRecorder     record.EventRecorder
	workerName         string
	workerNameSelector labels.Selector
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request.NamespacedName)

	err := r.reconcile(log, request.Name)
	if err != nil {
		log.Errorw("failed to reconcile", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(log *zap.SugaredLogger, seedName string) error {
	log.Debug("reconciling")

	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %v", err)
	}

	seed, exists := seeds[seedName]
	if !exists {
		log.Debug("ignoring request for non-existing seed")
		return nil
	}

	if seed.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return nil
	}

	client, exists := r.seedClients[seedName]
	if !exists {
		log.Debug("ignoring request for existing but uninitialized seed; the controller will be restarted once the kubeconfig is available")
		return nil
	}

	configList := &operatorv1alpha1.KubermaticConfigurationList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     r.namespace,
		LabelSelector: r.workerNameSelector,
	}

	if err := r.masterClient.List(r.ctx, configList, listOpts); err != nil {
		return fmt.Errorf("failed to find KubermaticConfigurations: %v", err)
	}

	if len(configList.Items) != 1 {
		log.Debug("ignoring request for namespace without KubermaticConfiguration")
		return nil
	}

	config := configList.Items[0]

	return r.reconcileResources(&config, seed, client, log)
}

func (r *Reconciler) reconcileResources(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	if err := r.reconcileServiceAccounts(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileSecrets(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileDeployments(cfg, seed, client, log); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) reconcileServiceAccounts(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ServiceAccounts")

	creators := []reconciling.NamedServiceAccountCreatorGetter{
		kubermatic.ServiceAccountCreator(cfg, seed),
	}

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, r.namespace, client, common.OwnershipModifierFactory(cfg)); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		kubermatic.ClusterRoleBindingCreator(cfg, seed),
	}

	if err := reconciling.ReconcileClusterRoleBindings(r.ctx, creators, "", client, common.OwnershipModifierFactory(cfg)); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMaps(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapCreatorGetter{
		kubermatic.BackupContainersConfigMapCreator(cfg),
	}

	if err := reconciling.ReconcileConfigMaps(r.ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(cfg)); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Secrets")

	creators := []reconciling.NamedSecretCreatorGetter{
		common.DockercfgSecretCreator(cfg),
		common.DexCASecretCreator(cfg),
	}

	if len(cfg.Spec.MasterFiles) > 0 {
		creators = append(creators, common.MasterFilesSecretCreator(cfg))
	}

	if err := reconciling.ReconcileSecrets(r.ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(cfg)); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileDeployments(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Deployments")

	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubermatic.SeedControllerManagerDeploymentCreator(r.workerName, cfg, seed),
	}

	modifiers := []reconciling.ObjectModifier{
		common.OwnershipModifierFactory(cfg),
		common.VolumeRevisionLabelsModifierFactory(r.ctx, client),
	}

	if err := reconciling.ReconcileDeployments(r.ctx, creators, r.namespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	return nil
}
