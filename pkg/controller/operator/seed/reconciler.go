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
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common/vpa"
	kubermaticseed "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/metering"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/crd"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// seed cluster.
type Reconciler struct {
	log                    *zap.SugaredLogger
	scheme                 *runtime.Scheme
	namespace              string
	masterClient           ctrlruntimeclient.Client
	masterRecorder         record.EventRecorder
	configGetter           provider.KubermaticConfigurationGetter
	seedClients            map[string]ctrlruntimeclient.Client
	seedRecorders          map[string]record.EventRecorder
	initializedSeedsGetter provider.SeedsGetter
	workerName             string
	versions               kubermaticversion.Versions
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("seed", request.Name)
	log.Debug("Reconciling")

	err := r.reconcile(ctx, log, request.Name)
	if err != nil {
		log.Errorw("failed to reconcile", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, seedName string) error {
	// find requested seed
	seeds, err := r.initializedSeedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %w", err)
	}

	seed, exists := seeds[seedName]
	if !exists {
		log.Debug("ignoring request for non-existing / uninitialized seed")
		return nil
	}

	if seed.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debugf("seed does not have matching %s label", kubermaticv1.WorkerNameLabelKey)
		return nil
	}

	// to allow a step-by-step migration of seed clusters, it's possible to
	// disable the operator's reconciling logic for seeds
	if _, ok := seed.Annotations[common.SkipReconcilingAnnotation]; ok {
		log.Debug("seed is marked as paused, skipping reconciliation")
		return nil
	}

	// get pre-constructed seed client
	seedClient, exists := r.seedClients[seed.Name]
	if !exists {
		log.Debug("ignoring request for existing but uninitialized seed; the controller will be reloaded once the kubeconfig is available")
		return nil
	}

	// get pre-constructed seed client
	seedRecorder := r.seedRecorders[seed.Name]

	// find the owning KubermaticConfiguration
	config, err := r.configGetter(ctx)
	if err != nil || config == nil {
		return err
	}

	// As the Seed CR is the owner for all resources managed by this controller,
	// we need the copy of the Seed resource from the master cluster on the seed cluster.
	seedCopy := &kubermaticv1.Seed{}
	name := types.NamespacedName{
		Name:      seed.Name,
		Namespace: r.namespace,
	}

	if err := seedClient.Get(ctx, name, seedCopy); err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf("cannot find copy of Seed resource on seed cluster: %w", err)

			r.masterRecorder.Event(config, corev1.EventTypeWarning, "SeedReconcilingSkipped", fmt.Sprintf("%s: %v", seed.Name, err))
			r.masterRecorder.Event(seed, corev1.EventTypeWarning, "ReconcilingSkipped", err.Error())

			if err := r.setSeedCondition(ctx, seed, corev1.ConditionFalse, "ReconcilingSkipped", err.Error()); err != nil {
				log.Errorw("Failed to update seed status", zap.Error(err))
			}

			return err
		}

		return fmt.Errorf("failed to get Seed in seed cluster: %w", err)
	}

	// Seed CR inside the seed cluster was deleted
	if seedCopy.DeletionTimestamp != nil {
		return r.cleanupDeletedSeed(ctx, config, seedCopy, seedClient, log)
	}

	// make sure to use the seedCopy so the owner ref has the correct UID
	if err := r.reconcileResources(ctx, config, seedCopy, seedClient, log); err != nil {
		r.masterRecorder.Event(config, corev1.EventTypeWarning, "SeedReconcilingError", fmt.Sprintf("%s: %v", seed.Name, err))
		r.masterRecorder.Event(seed, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		seedRecorder.Event(seedCopy, corev1.EventTypeWarning, "ReconcilingError", err.Error())

		if err := r.setSeedCondition(ctx, seed, corev1.ConditionFalse, "ReconcilingError", err.Error()); err != nil {
			log.Errorw("Failed to update seed status", zap.Error(err))
		}

		return err
	}

	if err := r.setSeedCondition(ctx, seed, corev1.ConditionTrue, "ReconcilingSuccess", ""); err != nil {
		log.Errorw("Failed to update seed status", zap.Error(err))
	}

	return nil
}

func (r *Reconciler) cleanupDeletedSeed(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	if !kubernetes.HasAnyFinalizer(seed, common.CleanupFinalizer) {
		return nil
	}

	// Note that this function does not remove CRDs, not because we're lazy, but because
	// it's safer to keep them around and old resources do not hurt cluster performance.

	log.Debug("Seed was deleted, cleaning up cluster-wide resources")

	if err := common.CleanupClusterResource(ctx, client, &rbacv1.ClusterRoleBinding{}, kubermaticseed.ClusterRoleBindingName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &rbacv1.ClusterRoleBinding{}, nodeportproxy.ClusterRoleBindingName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &rbacv1.ClusterRoleBinding{}, common.WebhookClusterRoleBindingName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &rbacv1.ClusterRole{}, nodeportproxy.ClusterRoleName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRole: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &rbacv1.ClusterRole{}, common.WebhookClusterRoleName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRole: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, common.SeedAdmissionWebhookName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up Seed ValidatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, common.KubermaticConfigurationAdmissionWebhookName(cfg)); err != nil {
		return fmt.Errorf("failed to clean up KubermaticConfiguration ValidatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, common.ApplicationDefinitionAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up ApplicationDefinition ValidatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, kubermaticseed.ClusterAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up Cluster ValidatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.MutatingWebhookConfiguration{}, kubermaticseed.ClusterAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up Cluster MutatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.MutatingWebhookConfiguration{}, kubermaticseed.AddonAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up Cluster MutatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.MutatingWebhookConfiguration{}, kubermaticseed.MLAAdminSettingAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up Cluster MutatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, kubermaticseed.OSCAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up OSC ValidatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, kubermaticseed.OSPAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up OSP ValidatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, kubermaticseed.IPAMPoolAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up IPAMPool ValidatingWebhookConfiguration: %w", err)
	}

	// On shared master+seed clusters, the kubermatic-webhook currently has the -seed-name
	// flag set; now that the seed (maybe the shared seed, maybe another) is gone, we must
	// trigger a reconciliation once to get rid of the flag. If the deleted Seed is just
	// a standalone cluster, no problem, the webhook Deployment will simply be deleted when
	// the kubermatic namespace is deleted. On shared seeds though the master-operator will
	// continue to reconcile the Deployment, but would itself not remove the -seed-name flag.
	creators := []reconciling.NamedDeploymentCreatorGetter{
		common.WebhookDeploymentCreator(cfg, r.versions, nil, true), // true is the important thing here
	}

	modifiers := []reconciling.ObjectModifier{
		common.OwnershipModifierFactory(seed, r.scheme),
		common.VolumeRevisionLabelsModifierFactory(ctx, client),
	}
	// add the image pull secret wrapper only when an image pull secret is provided
	if cfg.Spec.ImagePullSecret != "" {
		modifiers = append(modifiers, reconciling.ImagePullSecretsWrapper(common.DockercfgSecretName))
	}

	if err := reconciling.ReconcileDeployments(ctx, creators, r.namespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile webhook Deployment: %w", err)
	}

	return kubernetes.TryRemoveFinalizer(ctx, client, seed, common.CleanupFinalizer)
}

func (r *Reconciler) setSeedCondition(ctx context.Context, seed *kubermaticv1.Seed, status corev1.ConditionStatus, reason string, message string) error {
	return kubermaticv1helper.UpdateSeedStatus(ctx, r.masterClient, seed, func(s *kubermaticv1.Seed) {
		kubermaticv1helper.SetSeedCondition(s, kubermaticv1.SeedConditionResourcesReconciled, status, reason, message)
	})
}

func (r *Reconciler) reconcileResources(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	if err := kubernetes.TryAddFinalizer(ctx, client, seed, common.CleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	// apply the default values from the config to the current Seed
	seed, err := defaulting.DefaultSeed(seed, cfg, log)
	if err != nil {
		return fmt.Errorf("failed to apply defaults to Seed: %w", err)
	}

	caBundle, err := certificates.GlobalCABundle(ctx, r.masterClient, cfg)
	if err != nil {
		return fmt.Errorf("failed to get CA bundle ConfigMap: %w", err)
	}

	// Ensure that Old version of ApplicationInstallation CRD is removed.
	if err := controllerutil.RemoveOldApplicationInstallationCRD(ctx, client); err != nil {
		return err
	}

	// This is a migration that is required to ensure that with the release of KKP v2.21
	// we don't enable OSM for existing clusters.
	// TODO: Remove this with KKP 2.22 release.
	if err := r.disableOperatingSystemManager(ctx, client, log); err != nil {
		return err
	}

	if err := r.reconcileCRDs(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileServiceAccounts(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileRoles(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileRoleBindings(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileClusterRoles(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(ctx, cfg, seed, client, log, caBundle); err != nil {
		return err
	}

	if err := r.reconcileSecrets(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileAdmissionWebhooks(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileDeployments(ctx, cfg, seed, client, log, caBundle); err != nil {
		return err
	}

	if err := r.reconcilePodDisruptionBudgets(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	if err := r.reconcileServices(ctx, cfg, seed, client, log); err != nil {
		return err
	}

	// Since the new standalone webhook, the old service is not required anymore.
	// Once the webhooks are reconciled above, we can now clean up unneeded services.
	common.CleanupWebhookServices(ctx, client, log, cfg.Namespace)

	if err := metering.ReconcileMeteringResources(ctx, client, r.scheme, cfg, seed); err != nil {
		return err
	}

	// During the KKP 2.20->2.21 upgrade we must delete the aws-node-termination-handler addon exactly once.
	// TODO: Remove this with KKP 2.22 release.
	if err := r.migrateAWSNodeTerminationAddon(ctx, client, log); err != nil {
		return fmt.Errorf("failed to migrate AWS node-termination handler addon: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileCRDs(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling CRDs")

	creators := []reconciling.NamedCustomResourceDefinitionCreatorGetter{}

	groups, err := crd.Groups()
	if err != nil {
		return fmt.Errorf("failed to list CRD groups in operator: %w", err)
	}

	for _, group := range groups {
		crds, err := crd.CRDsForGroup(group)
		if err != nil {
			return fmt.Errorf("failed to list CRDs for API group %q in the operator: %w", group, err)
		}

		for i := range crds {
			creators = append(creators, common.CRDCreator(&crds[i], log, r.versions))
		}
	}

	if err := reconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile CRDs: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileServiceAccounts(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Kubermatic ServiceAccounts")

	creators := []reconciling.NamedServiceAccountCreatorGetter{
		kubermaticseed.ServiceAccountCreator(cfg, seed),
		common.WebhookServiceAccountCreator(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ServiceAccountCreator(cfg))
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, r.namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic ServiceAccounts: %w", err)
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators := []reconciling.NamedServiceAccountCreatorGetter{
			vpa.RecommenderServiceAccountCreator(),
			vpa.UpdaterServiceAccountCreator(),
			vpa.AdmissionControllerServiceAccountCreator(),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileServiceAccounts(ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile VPA ServiceAccounts: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileRoles(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Roles")

	creators := []reconciling.NamedRoleCreatorGetter{
		common.WebhookRoleCreator(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.RoleCreator())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, r.namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileRoleBindings(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling RoleBindings")

	creators := []reconciling.NamedRoleBindingCreatorGetter{
		common.WebhookRoleBindingCreator(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.RoleBindingCreator(cfg))
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, r.namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoles(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoles")

	creators := []reconciling.NamedClusterRoleCreatorGetter{
		common.WebhookClusterRoleCreator(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ClusterRoleCreator(cfg))
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators = append(creators, vpa.ClusterRoleCreators()...)
	}

	if err := reconciling.ReconcileClusterRoles(ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		kubermaticseed.ClusterRoleBindingCreator(cfg, seed),
		common.WebhookClusterRoleBindingCreator(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ClusterRoleBindingCreator(cfg))
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators = append(creators, vpa.ClusterRoleBindingCreators()...)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMaps(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger, caBundle *corev1.ConfigMap) error {
	log.Debug("reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapCreatorGetter{
		kubermaticseed.CABundleConfigMapCreator(caBundle),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Secrets")

	creators := []reconciling.NamedSecretCreatorGetter{
		common.WebhookServingCASecretCreator(cfg),
		common.WebhookServingCertSecretCreator(ctx, cfg, client),
	}

	if cfg.Spec.ImagePullSecret != "" {
		creators = append(creators, common.DockercfgSecretCreator(cfg))
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Secrets: %w", err)
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators := []reconciling.NamedSecretCreatorGetter{
			vpa.AdmissionControllerServingCertCreator(),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile VPA Secrets: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileDeployments(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger, caBundle *corev1.ConfigMap) error {
	log.Debug("reconciling Deployments")

	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubermaticseed.SeedControllerManagerDeploymentCreator(r.workerName, r.versions, cfg, seed),
		common.WebhookDeploymentCreator(cfg, r.versions, seed, false),
	}

	supportsFailureDomainZoneAntiAffinity, err := resources.SupportsFailureDomainZoneAntiAffinity(ctx, client)
	if err != nil {
		return err
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(
			creators,
			nodeportproxy.EnvoyDeploymentCreator(cfg, seed, supportsFailureDomainZoneAntiAffinity, r.versions),
			nodeportproxy.UpdaterDeploymentCreator(cfg, seed, r.versions),
		)
	}

	volumeLabelModifier := common.VolumeRevisionLabelsModifierFactory(ctx, client)
	modifiers := []reconciling.ObjectModifier{
		common.OwnershipModifierFactory(seed, r.scheme),
		volumeLabelModifier,
	}
	// add the image pull secret wrapper only when an image pull secret is
	// provided
	if cfg.Spec.ImagePullSecret != "" {
		modifiers = append(modifiers, reconciling.ImagePullSecretsWrapper(common.DockercfgSecretName))
	}

	if err := reconciling.ReconcileDeployments(ctx, creators, r.namespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Deployments: %w", err)
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators = []reconciling.NamedDeploymentCreatorGetter{
			vpa.RecommenderDeploymentCreator(cfg, r.versions),
			vpa.UpdaterDeploymentCreator(cfg, r.versions),
			vpa.AdmissionControllerDeploymentCreator(cfg, r.versions),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileDeployments(ctx, creators, metav1.NamespaceSystem, client, volumeLabelModifier); err != nil {
			return fmt.Errorf("failed to reconcile VPA Deployments: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcilePodDisruptionBudgets(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling PodDisruptionBudgets")

	creators := []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		kubermaticseed.SeedControllerManagerPDBCreator(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.EnvoyPDBCreator())
	}

	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileServices(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Services")

	creators := []reconciling.NamedServiceCreatorGetter{
		common.WebhookServiceCreator(cfg, client),
	}

	if err := reconciling.ReconcileServices(ctx, creators, cfg.Namespace, client, common.OwnershipModifierFactory(seed, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Services: %w", err)
	}

	// The nodeport-proxy LoadBalancer is not given an owner reference, so in case someone accidentally deletes
	// the Seed resource, the current LoadBalancer IP is not lost. To be truly destructive, users would need to
	// remove the entire Kubermatic namespace.
	if !seed.Spec.NodeportProxy.Disable {
		creators = []reconciling.NamedServiceCreatorGetter{
			nodeportproxy.ServiceCreator(seed),
		}

		if err := reconciling.ReconcileServices(ctx, creators, cfg.Namespace, client); err != nil {
			return fmt.Errorf("failed to reconcile nodeport-proxy Services: %w", err)
		}
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators := []reconciling.NamedServiceCreatorGetter{
			vpa.AdmissionControllerServiceCreator(),
		}

		// no ownership because these resources are most likely in a different namespace than Kubermatic
		if err := reconciling.ReconcileServices(ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile VPA Services: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileAdmissionWebhooks(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Admission Webhooks")

	validatingWebhookCreators := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{
		common.SeedAdmissionWebhookCreator(ctx, cfg, client),
		common.KubermaticConfigurationAdmissionWebhookCreator(ctx, cfg, client),
		kubermaticseed.ClusterValidatingWebhookConfigurationCreator(ctx, cfg, client),
		common.ApplicationDefinitionValidatingWebhookConfigurationCreator(ctx, cfg, client),
		kubermaticseed.IPAMPoolValidatingWebhookConfigurationCreator(ctx, cfg, client),
		kubermaticseed.OperatingSystemProfileValidatingWebhookConfigurationCreator(ctx, cfg, client),
		kubermaticseed.OperatingSystemConfigValidatingWebhookConfigurationCreator(ctx, cfg, client),
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, validatingWebhookCreators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile validating Admission Webhooks: %w", err)
	}

	mutatingWebhookCreators := []reconciling.NamedMutatingWebhookConfigurationCreatorGetter{
		kubermaticseed.ClusterMutatingWebhookConfigurationCreator(ctx, cfg, client),
		kubermaticseed.AddonMutatingWebhookConfigurationCreator(ctx, cfg, client),
		kubermaticseed.MLAAdminSettingMutatingWebhookConfigurationCreator(ctx, cfg, client),
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, mutatingWebhookCreators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile mutating Admission Webhooks: %w", err)
	}

	return nil
}

func (r *Reconciler) disableOperatingSystemManager(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("performing migration for .Spec.EnableOperatingSystemManager field in Clusters")

	// Master operator removes `cluster-webhook` service and at the time of execution of this piece of code, these webhook are effectively dead.
	// Any request to create, update, or delete the clusters will be rejected by the admission webhooks. Therefore, we need to remove the validating and mutating webhooks for clusters.
	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, kubermaticseed.ClusterAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up Cluster ValidatingWebhookConfiguration: %w", err)
	}

	if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.MutatingWebhookConfiguration{}, kubermaticseed.ClusterAdmissionWebhookName); err != nil {
		return fmt.Errorf("failed to clean up Cluster MutatingWebhookConfiguration: %w", err)
	}

	clusterList := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, clusterList); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	var errors []error
	// We need to ensure that we explicitly set `.spec.enableOperatingSystemManager` to false for existing clusters.
	// Although by default if this value is `nil` it should be treated as a truthy case. This migration is completely safe
	// to do since for all the new clusters this field will be explicitly set to `true` via the webhooks.
	for _, cluster := range clusterList.Items {
		if cluster.Spec.EnableOperatingSystemManager == nil {
			cluster.Spec.EnableOperatingSystemManager = pointer.Bool(false)

			if err := client.Update(ctx, &cluster); err != nil {
				// Instead of breaking on the first error just aggregate them to []errors and try to cover as much resources
				// as we can in each run.
				errors = append(errors, err)
			}
		}
	}

	return kerrors.NewAggregate(errors)
}

func (r *Reconciler) migrateAWSNodeTerminationAddon(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("Migrating addons...")

	// Before we can migrate we need to ensure that the seed-ctrl-mgr is actually up-to-date,
	// otherwise it could happen that an old seed-ctrl-mgr is re-installing an old aws-node
	// addon.
	seedCtrlMgrDeployment := &appsv1.Deployment{}
	key := types.NamespacedName{
		Name:      common.SeedControllerManagerDeploymentName,
		Namespace: r.namespace,
	}
	if err := client.Get(ctx, key, seedCtrlMgrDeployment); err != nil {
		return fmt.Errorf("failed to retrieve seed-controller-manager Deployment: %w", err)
	}

	ready, err := kubernetes.IsDeploymentRolloutComplete(seedCtrlMgrDeployment, -1)
	if err != nil {
		return err
	}

	if !ready {
		log.Debug("seed-controller-manager Deployment is not yet ready")
		return nil
	}

	// Now that we're sure the current seed-ctrl-mgr is running, we can tell it to
	// delete the affected addons and rely on the addoninstaller controller to bring
	// it back later.

	clusterList := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, clusterList); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	var errors []error
	for _, cluster := range clusterList.Items {
		// Have we already migrated this cluster?
		if _, exists := cluster.Annotations[resources.AWSNodeTerminationHandlerMigrationAnnotation]; exists {
			continue
		}

		// Not yet reconciled.
		if cluster.Status.NamespaceName == "" {
			continue
		}

		// Not an AWS cluster, so there's nothing to do.
		if cluster.Spec.Cloud.ProviderName != string(kubermaticv1.AWSCloudProvider) {
			continue
		}

		// Here we go!
		err := deleteAddon(ctx, client, &cluster)

		// Something bad happened when trying to delete the addon.
		if err != nil {
			errors = append(errors, err)
			continue
		}

		// Addon was deleted, let's mark the cluster accordingly.
		oldCluster := cluster.DeepCopy()

		if cluster.Annotations == nil {
			cluster.Annotations = map[string]string{}
		}
		cluster.Annotations[resources.AWSNodeTerminationHandlerMigrationAnnotation] = "yes"

		patchOpts := ctrlruntimeclient.MergeFromWithOptions(oldCluster, ctrlruntimeclient.MergeFromWithOptimisticLock{})
		if err := client.Patch(ctx, &cluster, patchOpts); err != nil {
			errors = append(errors, err)
		}
	}

	return kerrors.NewAggregate(errors)
}

func deleteAddon(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	addon := &kubermaticv1.Addon{}
	key := types.NamespacedName{
		Name:      "aws-node-termination-handler",
		Namespace: cluster.Status.NamespaceName,
	}
	if err := client.Get(ctx, key, addon); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// Addon already in deletion.
	if addon.DeletionTimestamp != nil {
		return nil
	}

	return client.Delete(ctx, addon)
}
