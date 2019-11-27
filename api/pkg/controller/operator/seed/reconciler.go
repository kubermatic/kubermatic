package seed

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/seed/resources/kubermatic"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// seed cluster.
type Reconciler struct {
	ctrlruntimeclient.Client

	ctx            context.Context
	log            *zap.SugaredLogger
	scheme         *runtime.Scheme
	namespace      string
	masterClient   ctrlruntimeclient.Client
	masterRecorder record.EventRecorder
	seedClients    map[string]ctrlruntimeclient.Client
	seedRecorders  map[string]record.EventRecorder
	seedsGetter    provider.SeedsGetter
	workerName     string
	versions       common.Versions
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

	// find requested seed
	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %v", err)
	}

	seed, exists := seeds[seedName]
	if !exists {
		log.Debug("ignoring request for non-existing seed")
		return nil
	}

	// get pre-constructed seed client
	seedClient, exists := r.seedClients[seedName]
	if !exists {
		log.Debug("ignoring request for existing but uninitialized seed; the controller will be reloaded once the kubeconfig is available")
		return nil
	}

	// get pre-constructed seed client
	seedRecorder := r.seedRecorders[seedName]

	// find the owning KubermaticConfiguration
	configList := &operatorv1alpha1.KubermaticConfigurationList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace: r.namespace,
	}

	if err := r.masterClient.List(r.ctx, configList, listOpts); err != nil {
		return fmt.Errorf("failed to find KubermaticConfigurations: %v", err)
	}

	if len(configList.Items) == 0 {
		log.Debug("ignoring request for namespace without KubermaticConfiguration")
		return nil
	}

	if len(configList.Items) > 1 {
		log.Infow("there are multiple KubermaticConfiguration objects, cannot reconcile", "namespace", r.namespace)
		return nil
	}

	config := configList.Items[0]

	// As the Seed CR is the owner for all resources managed by this controller,
	// we wait for the seed-sync controller to do its job and mirror the Seed CR
	// into the seed cluster.
	seedCopy := &kubermaticv1.Seed{}
	name := types.NamespacedName{
		Name:      seedName,
		Namespace: r.namespace,
	}

	if err := seedClient.Get(r.ctx, name, seedCopy); err != nil {
		if kerrors.IsNotFound(err) {
			err = errors.New("seed cluster has not yet been provisioned and contains no Seed CR yet")

			r.masterRecorder.Event(&config, corev1.EventTypeWarning, "SeedReconcilingSkipped", fmt.Sprintf("%s: %v", seedName, err))
			r.masterRecorder.Event(seed, corev1.EventTypeWarning, "ReconcilingSkipped", err.Error())
			seedRecorder.Event(seedCopy, corev1.EventTypeWarning, "ReconcilingSkipped", err.Error())
			return err
		}

		return fmt.Errorf("failed to get Seed in seed cluster: %v", err)
	}

	// make sure to use the seedCopy so the owner ref has the correct UID
	if err := r.reconcileResources(&config, seedCopy, seedClient, log); err != nil {
		r.masterRecorder.Event(&config, corev1.EventTypeWarning, "SeedReconcilingError", fmt.Sprintf("%s: %v", seedName, err))
		r.masterRecorder.Event(seed, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		seedRecorder.Event(seedCopy, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return err
	}

	return nil
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

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, r.namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		kubermatic.ClusterRoleBindingCreator(cfg, seed),
	}

	if err := reconciling.ReconcileClusterRoleBindings(r.ctx, creators, "", client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMaps(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapCreatorGetter{
		kubermatic.BackupContainersConfigMapCreator(cfg),
	}

	if creator := kubermatic.ClusterNamespacePrometheusScrapingConfigsConfigMapCreator(cfg); creator != nil {
		creators = append(creators, creator)
	}

	if creator := kubermatic.ClusterNamespacePrometheusRulesConfigMapCreator(cfg); creator != nil {
		creators = append(creators, creator)
	}

	if err := reconciling.ReconcileConfigMaps(r.ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Secrets")

	creators := []reconciling.NamedSecretCreatorGetter{
		common.DockercfgSecretCreator(cfg),
		common.DexCASecretCreator(cfg),
		common.SeedWebhookServingCASecretCreator(cfg),
		common.SeedWebhookServingCertSecretCreator(cfg, client),
	}

	if len(cfg.Spec.MasterFiles) > 0 {
		creators = append(creators, common.MasterFilesSecretCreator(cfg))
	}

	if err := reconciling.ReconcileSecrets(r.ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileDeployments(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Deployments")

	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubermatic.SeedControllerManagerDeploymentCreator(r.workerName, r.versions, cfg, seed),
	}

	modifiers := []reconciling.ObjectModifier{
		common.OwnershipModifierFactory(seed, r.scheme),
		common.VolumeRevisionLabelsModifierFactory(r.ctx, client),
	}

	if err := reconciling.ReconcileDeployments(r.ctx, creators, r.namespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	return nil
}
