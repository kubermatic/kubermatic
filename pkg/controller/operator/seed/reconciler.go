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

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common/vpa"
	kubermaticseed "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/metering"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/networkpolicy"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/crd"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
	crdutil "k8c.io/kubermatic/v2/pkg/util/crd"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	names := []string{
		common.SeedAdmissionWebhookName(cfg),
		common.KubermaticConfigurationAdmissionWebhookName(cfg),
		common.ApplicationDefinitionAdmissionWebhookName,
		common.PoliciesAdmissionWebhookName,
		common.PolicyTemplateAdmissionWebhookName,
		kubermaticseed.ClusterAdmissionWebhookName,
		kubermaticseed.IPAMPoolAdmissionWebhookName,
	}

	for _, name := range names {
		if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.ValidatingWebhookConfiguration{}, name); err != nil {
			return fmt.Errorf("failed to clean up %s ValidatingWebhookConfiguration: %w", name, err)
		}
	}

	names = []string{
		kubermaticseed.ClusterAdmissionWebhookName,
		kubermaticseed.AddonAdmissionWebhookName,
		kubermaticseed.MLAAdminSettingAdmissionWebhookName,
	}

	for _, name := range names {
		if err := common.CleanupClusterResource(ctx, client, &admissionregistrationv1.MutatingWebhookConfiguration{}, name); err != nil {
			return fmt.Errorf("failed to clean up %s MutatingWebhookConfiguration: %w", name, err)
		}
	}

	// On shared master+seed clusters, the kubermatic-webhook currently has the -seed-name
	// flag set; now that the seed (maybe the shared seed, maybe another) is gone, we must
	// trigger a reconciliation once to get rid of the flag. If the deleted Seed is just
	// a standalone cluster, no problem, the webhook Deployment will simply be deleted when
	// the kubermatic namespace is deleted. On shared seeds though the master-operator will
	// continue to reconcile the Deployment, but would itself not remove the -seed-name flag.
	creators := []reconciling.NamedDeploymentReconcilerFactory{
		common.WebhookDeploymentReconciler(cfg, r.versions, nil, true), // true is the important thing here
	}

	modifiers := []reconciling.ObjectModifier{
		modifier.Ownership(seed, common.OperatorName, r.scheme),
		modifier.RelatedRevisionsLabels(ctx, client),
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
	return util.UpdateSeedStatus(ctx, r.masterClient, seed, func(s *kubermaticv1.Seed) {
		util.SetSeedCondition(s, kubermaticv1.SeedConditionResourcesReconciled, status, reason, message)
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

	isCiliumDeployed, err := networkpolicy.CiliumCRDExists(ctx, client)
	if err != nil {
		return err
	}

	if isCiliumDeployed {
		if err := r.reconcileCiliumNetworkPolicies(ctx, cfg, seed, client, log); err != nil {
			return err
		}
	}

	// Since the new standalone webhook, the old service is not required anymore.
	// Once the webhooks are reconciled above, we can now clean up unneeded services.
	common.CleanupWebhookServices(ctx, client, log, cfg.Namespace)

	if err := metering.ReconcileMeteringResources(ctx, client, r.scheme, cfg, seed); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) reconcileCRDs(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling CRDs")

	creators := []kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory{}

	groups, err := crd.Groups()
	if err != nil {
		return fmt.Errorf("failed to list CRD groups in operator: %w", err)
	}

	for _, group := range groups {
		crds, err := crd.CRDsForGroup(group)
		if err != nil {
			return fmt.Errorf("failed to list CRDs for API group %q in the operator: %w", group, err)
		}

		for i, crdObject := range crds {
			if crdutil.SkipCRDOnCluster(&crdObject, crdutil.SeedCluster) {
				continue
			}

			// Skip installation of the UserSSHKeys CRD when SSH key functionality is disabled.
			// The CRD is automatically included by default when DisableUserSSHKeys featuregate is false/unset.
			if cfg.Spec.FeatureGates[features.DisableUserSSHKey] && crdObject.Name == "usersshkeys.kubermatic.k8c.io" {
				continue
			}

			creators = append(creators, common.CRDReconciler(&crds[i], log, r.versions))
		}
	}

	if err := kkpreconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile CRDs: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileServiceAccounts(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Kubermatic ServiceAccounts")

	creators := []reconciling.NamedServiceAccountReconcilerFactory{
		kubermaticseed.ServiceAccountReconciler(cfg, seed),
		common.WebhookServiceAccountReconciler(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ServiceAccountReconciler(cfg))
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, r.namespace, client, modifier.Ownership(seed, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic ServiceAccounts: %w", err)
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators := []reconciling.NamedServiceAccountReconcilerFactory{
			vpa.RecommenderServiceAccountReconciler(),
			vpa.UpdaterServiceAccountReconciler(),
			vpa.AdmissionControllerServiceAccountReconciler(),
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

	creators := []reconciling.NamedRoleReconcilerFactory{
		common.WebhookRoleReconciler(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.RoleReconciler())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, r.namespace, client, modifier.Ownership(seed, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileRoleBindings(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling RoleBindings")

	creators := []reconciling.NamedRoleBindingReconcilerFactory{
		common.WebhookRoleBindingReconciler(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.RoleBindingReconciler(cfg))
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, r.namespace, client, modifier.Ownership(seed, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoles(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoles")

	creators := []reconciling.NamedClusterRoleReconcilerFactory{
		common.WebhookClusterRoleReconciler(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ClusterRoleReconciler(cfg))
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators = append(creators, vpa.ClusterRoleReconcilers()...)
	}

	if err := reconciling.ReconcileClusterRoles(ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		kubermaticseed.ClusterRoleBindingReconciler(cfg, seed),
		common.WebhookClusterRoleBindingReconciler(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.ClusterRoleBindingReconciler(cfg))
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators = append(creators, vpa.ClusterRoleBindingReconcilers()...)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, creators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMaps(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger, caBundle *corev1.ConfigMap) error {
	log.Debug("reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapReconcilerFactory{
		kubermaticseed.CABundleConfigMapReconciler(caBundle),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, cfg.Namespace, client, modifier.Ownership(seed, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Secrets")

	creators := []reconciling.NamedSecretReconcilerFactory{
		common.WebhookServingCASecretReconciler(cfg),
		common.WebhookServingCertSecretReconciler(ctx, cfg, client),
	}

	if cfg.Spec.ImagePullSecret != "" {
		creators = append(creators, common.DockercfgSecretReconciler(cfg))
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, cfg.Namespace, client, modifier.Ownership(seed, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Secrets: %w", err)
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators := []reconciling.NamedSecretReconcilerFactory{
			vpa.AdmissionControllerServingCertReconciler(),
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

	creators := []reconciling.NamedDeploymentReconcilerFactory{
		kubermaticseed.SeedControllerManagerDeploymentReconciler(r.workerName, r.versions, cfg, seed),
		common.WebhookDeploymentReconciler(cfg, r.versions, seed, false),
	}

	supportsFailureDomainZoneAntiAffinity, err := resources.SupportsFailureDomainZoneAntiAffinity(ctx, client)
	if err != nil {
		return err
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(
			creators,
			nodeportproxy.EnvoyDeploymentReconciler(cfg, seed, supportsFailureDomainZoneAntiAffinity, r.versions),
			nodeportproxy.UpdaterDeploymentReconciler(cfg, seed, r.versions),
		)
	}

	volumeLabelModifier := modifier.RelatedRevisionsLabels(ctx, client)
	modifiers := []reconciling.ObjectModifier{
		modifier.Ownership(seed, common.OperatorName, r.scheme),
		modifier.VersionLabel(r.versions.GitVersion),
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
		creators = []reconciling.NamedDeploymentReconcilerFactory{
			vpa.RecommenderDeploymentReconciler(cfg, r.versions),
			vpa.UpdaterDeploymentReconciler(cfg, r.versions),
			vpa.AdmissionControllerDeploymentReconciler(cfg, r.versions),
		}

		// no ownership because these resources are not in the kubermatic namespace
		if err := reconciling.ReconcileDeployments(ctx, creators, metav1.NamespaceSystem, client, volumeLabelModifier); err != nil {
			return fmt.Errorf("failed to reconcile VPA Deployments: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileCiliumNetworkPolicies(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling CiliumNetworkPolicies")

	netpol := &ciliumv2.CiliumClusterwideNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: networkpolicy.CiliumSeedApiserverAllow}}
	if _, err := controllerutil.CreateOrUpdate(ctx, client, netpol, func() error {
		netpol.Spec = networkpolicy.SeedApiserverRule()
		return nil
	}); err != nil {
		return fmt.Errorf("failed to reconcile CiliumClusterwideNetworkPolicies: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcilePodDisruptionBudgets(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling PodDisruptionBudgets")

	creators := []reconciling.NamedPodDisruptionBudgetReconcilerFactory{
		kubermaticseed.SeedControllerManagerPDBReconciler(cfg),
	}

	if !seed.Spec.NodeportProxy.Disable {
		creators = append(creators, nodeportproxy.EnvoyPDBReconciler())
	}

	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, cfg.Namespace, client, modifier.Ownership(seed, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %w", err)
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators = append(creators,
			vpa.RecommenderPDBReconciler(),
			vpa.UpdaterPDBReconciler(),
			vpa.AdmissionControllerPDBReconciler(),
		)

		// no ownership because these resources are not in the kubermatic namespace
		if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileServices(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Services")

	creators := []reconciling.NamedServiceReconcilerFactory{
		common.WebhookServiceReconciler(cfg, client),
	}

	if err := reconciling.ReconcileServices(ctx, creators, cfg.Namespace, client, modifier.Ownership(seed, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Services: %w", err)
	}

	// The nodeport-proxy LoadBalancer is not given an owner reference, so in case someone accidentally deletes
	// the Seed resource, the current LoadBalancer IP is not lost. To be truly destructive, users would need to
	// remove the entire Kubermatic namespace.
	if !seed.Spec.NodeportProxy.Disable {
		creators = []reconciling.NamedServiceReconcilerFactory{
			nodeportproxy.ServiceReconciler(seed),
		}

		if err := reconciling.ReconcileServices(ctx, creators, cfg.Namespace, client); err != nil {
			return fmt.Errorf("failed to reconcile nodeport-proxy Services: %w", err)
		}
	}

	if cfg.Spec.FeatureGates[features.VerticalPodAutoscaler] {
		creators := []reconciling.NamedServiceReconcilerFactory{
			vpa.AdmissionControllerServiceReconciler(),
		}

		// no ownership because these resources are not in the kubermatic namespace
		if err := reconciling.ReconcileServices(ctx, creators, metav1.NamespaceSystem, client); err != nil {
			return fmt.Errorf("failed to reconcile VPA Services: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileAdmissionWebhooks(ctx context.Context, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	log.Debug("reconciling Admission Webhooks")

	validatingWebhookReconcilers := []reconciling.NamedValidatingWebhookConfigurationReconcilerFactory{
		common.SeedAdmissionWebhookReconciler(ctx, cfg, client),
		common.KubermaticConfigurationAdmissionWebhookReconciler(ctx, cfg, client),
		kubermaticseed.ClusterValidatingWebhookConfigurationReconciler(ctx, cfg, client),
		common.ApplicationDefinitionValidatingWebhookConfigurationReconciler(ctx, cfg, client),
		kubermaticseed.IPAMPoolValidatingWebhookConfigurationReconciler(ctx, cfg, client),
		common.PoliciesWebhookConfigurationReconciler(ctx, cfg, client),
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, validatingWebhookReconcilers, "", client); err != nil {
		return fmt.Errorf("failed to reconcile validating Admission Webhooks: %w", err)
	}

	mutatingWebhookReconcilers := []reconciling.NamedMutatingWebhookConfigurationReconcilerFactory{
		kubermaticseed.ClusterMutatingWebhookConfigurationReconciler(ctx, cfg, client),
		kubermaticseed.AddonMutatingWebhookConfigurationReconciler(ctx, cfg, client),
		kubermaticseed.MLAAdminSettingMutatingWebhookConfigurationReconciler(ctx, cfg, client),
		common.ApplicationDefinitionMutatingWebhookConfigurationReconciler(ctx, cfg, client),
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, mutatingWebhookReconcilers, "", client); err != nil {
		return fmt.Errorf("failed to reconcile mutating Admission Webhooks: %w", err)
	}

	return nil
}
