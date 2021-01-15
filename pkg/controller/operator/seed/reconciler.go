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

package seed

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common/vpa"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/gatekeeper"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	versions       kubermaticversion.Versions
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

	// to allow a step-by-step migration of seed clusters, it's possible to
	// disable the operator's reconciling logic for seeds
	if _, ok := seed.Annotations[common.SkipReconcilingAnnotation]; ok {
		log.Info("seed is marked as paused, skipping reconciliation")
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

	// create a copy of the configuration with default values applied
	defaulted, err := common.DefaultConfiguration(&config, log)
	if err != nil {
		return fmt.Errorf("failed to apply defaults to KubermaticConfiguration: %v", err)
	}

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

	// Seed CR inside the seed cluster was deleted
	if seedCopy.DeletionTimestamp != nil {
		return r.cleanupDeletedSeed(defaulted, seedCopy, seedClient, log)
	}

	// make sure to use the seedCopy so the owner ref has the correct UID
	if err := r.reconcileResources(defaulted, seedCopy, seedClient, log); err != nil {
		r.masterRecorder.Event(&config, corev1.EventTypeWarning, "SeedReconcilingError", fmt.Sprintf("%s: %v", seedName, err))
		r.masterRecorder.Event(seed, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		seedRecorder.Event(seedCopy, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return err
	}

	return nil
}

func (r *Reconciler) cleanupDeletedSeed(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	if !kubernetes.HasAnyFinalizer(seed, common.CleanupFinalizer) {
		return nil
	}

	log.Debug("Seed was deleted, cleaning up cluster-wide resources")

	if err := common.CleanupClusterResource(client, &rbacv1.ClusterRoleBinding{}, kubermatic.ClusterRoleBindingName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %v", err)
	}

	if err := common.CleanupClusterResource(client, &rbacv1.ClusterRoleBinding{}, nodeportproxy.ClusterRoleBindingName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %v", err)
	}

	if err := common.CleanupClusterResource(client, &rbacv1.ClusterRole{}, nodeportproxy.ClusterRoleName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRole: %v", err)
	}

	if err := common.CleanupClusterResource(client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, common.SeedAdmissionWebhookName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ValidatingWebhookConfiguration: %v", err)
	}

	oldSeed := seed.DeepCopy()
	kubernetes.RemoveFinalizer(seed, common.CleanupFinalizer)

	if err := client.Patch(r.ctx, seed, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
		return fmt.Errorf("failed to remove finalizer from Seed: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileResources(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	oldSeed := seed.DeepCopy()
	kubernetes.AddFinalizer(seed, common.CleanupFinalizer)
	if err := client.Patch(r.ctx, seed, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
		return fmt.Errorf("failed to add finalizer to Seed: %v", err)
	}

	seed, err := common.DefaultSeed(seed, log)
	if err != nil {
		return fmt.Errorf("failed to apply default values to Seed:  %v", err)
	}

	if err := r.reconcileNamespaces(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileServiceAccounts(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileRoles(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileRoleBindings(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileClusterRoles(cfg, seed, client, log); err != nil {
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

	if err := r.reconcilePodDisruptionBudgets(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileServices(cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileAdmissionWebhooks(cfg, seed, client, log); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) reconcileNamespaces(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Namespaces")

	creators := []reconciling.NamedNamespaceCreatorGetter{
		common.NamespaceCreator(cfg),
	}

	if err := reconciling.ReconcileNamespaces(r.ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile Namespaces: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileServiceAccounts(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Kubermatic ServiceAccounts")

	creators := []reconciling.NamedServiceAccountCreatorGetter{
		kubermatic.ServiceAccountCreator(cfg, seed),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ServiceAccountCreator(cfg))
	}

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, r.namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic ServiceAccounts: %v", err)
	}

	if cfg.Spec.FeatureGates.Has(features.VerticalPodAutoscaler) {
		creators := []reconciling.NamedServiceAccountCreatorGetter{
			vpa.RecommenderServiceAccountCreator(),
			vpa.UpdaterServiceAccountCreator(),
			vpa.AdmissionControllerServiceAccountCreator(),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile VPA ServiceAccounts: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileRoles(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Roles")

	if seed.Spec.NodeportProxy.Disable {
		return nil
	}

	creators := []reconciling.NamedRoleCreatorGetter{
		nodeportproxy.RoleCreator(),
	}

	if err := reconciling.ReconcileRoles(r.ctx, creators, r.namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileRoleBindings(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling RoleBindings")

	if seed.Spec.NodeportProxy.Disable {
		return nil
	}

	creators := []reconciling.NamedRoleBindingCreatorGetter{
		nodeportproxy.RoleBindingCreator(cfg),
	}

	if err := reconciling.ReconcileRoleBindings(r.ctx, creators, r.namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoles(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoles")

	creators := []reconciling.NamedClusterRoleCreatorGetter{
		gatekeeper.ClusterRoleCreator(),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ClusterRoleCreator(cfg))
	}

	if cfg.Spec.FeatureGates.Has(features.VerticalPodAutoscaler) {
		creators = append(creators, vpa.ClusterRoleCreators()...)
	}

	if err := reconciling.ReconcileClusterRoles(r.ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		kubermatic.ClusterRoleBindingCreator(cfg, seed),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ClusterRoleBindingCreator(cfg))
	}

	if cfg.Spec.FeatureGates.Has(features.VerticalPodAutoscaler) {
		creators = append(creators, vpa.ClusterRoleBindingCreators()...)
	}

	if err := reconciling.ReconcileClusterRoleBindings(r.ctx, creators, "", client); err != nil {
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
		common.ExtraFilesSecretCreator(cfg),
		common.SeedWebhookServingCASecretCreator(cfg),
		common.SeedWebhookServingCertSecretCreator(cfg, client),
	}

	if cfg.Spec.ImagePullSecret != "" {
		creators = append(creators, common.DockercfgSecretCreator(cfg))
	}

	if cfg.Spec.Auth.CABundle != "" {
		creators = append(creators, common.DexCASecretCreator(cfg))
	}

	if err := reconciling.ReconcileSecrets(r.ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Secrets: %v", err)
	}

	if cfg.Spec.FeatureGates.Has(features.VerticalPodAutoscaler) {
		creators := []reconciling.NamedSecretCreatorGetter{
			vpa.AdmissionControllerServingCertCreator(),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileSecrets(r.ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile VPA Secrets: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileDeployments(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Deployments")

	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubermatic.SeedControllerManagerDeploymentCreator(r.workerName, r.versions, cfg, seed),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(
			creators,
			nodeportproxy.EnvoyDeploymentCreator(cfg, seed, r.versions),
			nodeportproxy.UpdaterDeploymentCreator(cfg, seed, r.versions),
		)
	}

	volumeLabelModifier := common.VolumeRevisionLabelsModifierFactory(r.ctx, client)
	modifiers := []reconciling.ObjectModifier{
		common.OwnershipModifierFactory(seed, r.scheme),
		volumeLabelModifier,
	}
	// add the image pull secret wrapper only when an image pull secret is
	// provided
	if cfg.Spec.ImagePullSecret != "" {
		modifiers = append(modifiers, reconciling.ImagePullSecretsWrapper(common.DockercfgSecretName))
	}

	if err := reconciling.ReconcileDeployments(r.ctx, creators, r.namespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Deployments: %v", err)
	}

	if cfg.Spec.FeatureGates.Has(features.VerticalPodAutoscaler) {
		creators = []reconciling.NamedDeploymentCreatorGetter{
			vpa.RecommenderDeploymentCreator(cfg, r.versions),
			vpa.UpdaterDeploymentCreator(cfg, r.versions),
			vpa.AdmissionControllerDeploymentCreator(cfg, r.versions),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileDeployments(r.ctx, creators, metav1.NamespaceSystem, client, volumeLabelModifier); err != nil {
			return fmt.Errorf("failed to reconcile VPA Deployments: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcilePodDisruptionBudgets(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling PodDisruptionBudgets")

	creators := []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		kubermatic.SeedControllerManagerPDBCreator(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.EnvoyPDBCreator())
	}

	if err := reconciling.ReconcilePodDisruptionBudgets(r.ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %v", err)
	}

	return nil
}

func (r *Reconciler) reconcileServices(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Services")

	creators := []reconciling.NamedServiceCreatorGetter{
		common.SeedAdmissionServiceCreator(cfg, client),
	}

	if err := reconciling.ReconcileServices(r.ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Services: %v", err)
	}

	// The nodeport-proxy LoadBalancer is not given an owner reference, so in case someone accidentally deletes
	// the Seed resource, the current LoadBalancer IP is not lost. To be truly destructive, users would need to
	// remove the entire Kubermatic namespace.
	if !seed.Spec.NodeportProxy.Disable {
		creators = []reconciling.NamedServiceCreatorGetter{
			nodeportproxy.ServiceCreator(seed),
		}

		if err := reconciling.ReconcileServices(r.ctx, creators, cfg.Namespace, client); err != nil {
			return fmt.Errorf("failed to reconcile nodeport-proxy Services: %v", err)
		}
	}

	if cfg.Spec.FeatureGates.Has(features.VerticalPodAutoscaler) {
		creators := []reconciling.NamedServiceCreatorGetter{
			vpa.AdmissionControllerServiceCreator(),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileServices(r.ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile VPA Services: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileAdmissionWebhooks(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling AdmissionWebhooks")

	creators := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{
		common.SeedAdmissionWebhookCreator(cfg, client),
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(r.ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile AdmissionWebhooks: %v", err)
	}

	return nil
}
