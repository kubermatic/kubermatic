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

package master

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cni/cilium"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/kubermatic"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	versions   kubermaticversion.Versions
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// find the requested configuration
	config := &kubermaticv1.KubermaticConfiguration{}
	if err := r.Get(ctx, request.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not get KubermaticConfiguration %q: %w", request, err)
	}

	identifier, err := cache.MetaNamespaceKeyFunc(config)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to determine string key for KubermaticConfiguration: %w", err)
	}

	logger := r.log.With("config", identifier)

	err = r.reconcile(ctx, config, logger)
	if err != nil {
		r.recorder.Event(config, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Kubermatic configuration")

	// config was deleted, let's clean up
	if config.DeletionTimestamp != nil {
		return r.cleanupDeletedConfiguration(ctx, config, logger)
	}

	// ensure we always have a cleanup finalizer
	if err := kubernetes.TryAddFinalizer(ctx, r, config, common.CleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	// patching the config will refresh the object, so any attempts to set the default values
	// before calling Patch() are pointless, as the defaults would be gone after the call
	defaulted, err := defaulting.DefaultConfiguration(config, logger)
	if err != nil {
		return fmt.Errorf("failed to apply defaults: %w", err)
	}

	if err := r.reconcileServiceAccounts(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileRoles(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileRoleBindings(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileClusterRoles(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileSecrets(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileDeployments(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcilePodDisruptionBudgets(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileServices(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileIngresses(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileValidatingWebhooks(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileMutatingWebhooks(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileAddonConfigs(ctx, defaulted, logger); err != nil {
		return err
	}

	if err := r.reconcileApplicationDefinitions(ctx, defaulted, logger); err != nil {
		return err
	}

	// Since the new standalone webhook, the old service is not required anymore.
	// Once the webhooks are reconciled above, we can now clean up unneeded services.
	common.CleanupWebhookServices(ctx, r, logger, defaulted.Namespace)

	return nil
}

func (r *Reconciler) cleanupDeletedConfiguration(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	if !kubernetes.HasAnyFinalizer(config, common.CleanupFinalizer) {
		return nil
	}

	logger.Debug("KubermaticConfiguration was deleted, cleaning up cluster-wide resources")

	if err := common.CleanupClusterResource(ctx, r, &rbacv1.ClusterRoleBinding{}, kubermatic.ClusterRoleBindingName(config)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	validating := []string{
		common.UserAdmissionWebhookName,
		common.UserSSHKeyAdmissionWebhookName,
		common.SeedAdmissionWebhookName(config),
		common.KubermaticConfigurationAdmissionWebhookName(config),
		common.GroupProjectBindingAdmissionWebhookName,
		common.ResourceQuotaAdmissionWebhookName,
		common.PolicyTemplateAdmissionWebhookName,
	}

	mutating := []string{
		common.UserSSHKeyAdmissionWebhookName,
		common.ExternalClusterAdmissionWebhookName,
		common.ResourceQuotaAdmissionWebhookName,
	}

	for _, webhook := range validating {
		if err := common.CleanupClusterResource(ctx, r, &admissionregistrationv1.ValidatingWebhookConfiguration{}, webhook); err != nil {
			return fmt.Errorf("failed to clean up validating webhook for %q: %w", webhook, err)
		}
	}

	for _, webhook := range mutating {
		if err := common.CleanupClusterResource(ctx, r, &admissionregistrationv1.MutatingWebhookConfiguration{}, webhook); err != nil {
			return fmt.Errorf("failed to clean up mutating webhook for %q: %w", webhook, err)
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r, config, common.CleanupFinalizer)
}

func (r *Reconciler) reconcileConfigMaps(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ConfigMaps")

	reconcilers := []reconciling.NamedConfigMapReconcilerFactory{}
	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		reconcilers = append(reconcilers, kubermatic.UIConfigConfigMapReconciler(config))
	}

	if err := reconciling.ReconcileConfigMaps(ctx, reconcilers, config.Namespace, r.Client, modifier.Ownership(config, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Secrets")

	reconcilers := []reconciling.NamedSecretReconcilerFactory{
		common.WebhookServingCASecretReconciler(config),
		common.WebhookServingCertSecretReconciler(ctx, config, r.Client),
	}

	if config.Spec.ImagePullSecret != "" {
		reconcilers = append(reconcilers, common.DockercfgSecretReconciler(config))
	}

	if err := reconciling.ReconcileSecrets(ctx, reconcilers, config.Namespace, r.Client, modifier.Ownership(config, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileServiceAccounts(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ServiceAccounts")

	reconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		kubermatic.ServiceAccountReconciler(config),
		kubermatic.APIServiceAccountReconciler(),
		common.WebhookServiceAccountReconciler(config),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, reconcilers, config.Namespace, r.Client, modifier.Ownership(config, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileRoles(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Roles")

	reconcilers := []reconciling.NamedRoleReconcilerFactory{
		common.WebhookRoleReconciler(config),
		kubermatic.APIRoleReconciler(),
	}

	if err := reconciling.ReconcileRoles(ctx, reconcilers, config.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileRoleBindings(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling RoleBindings")

	reconcilers := []reconciling.NamedRoleBindingReconcilerFactory{
		common.WebhookRoleBindingReconciler(config),
		kubermatic.APIRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, reconcilers, config.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoles(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ClusterRoles")

	reconcilers := []reconciling.NamedClusterRoleReconcilerFactory{
		kubermatic.APIClusterRoleReconciler(config),
		common.WebhookClusterRoleReconciler(config),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, reconcilers, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ClusterRoleBindings")

	reconcilers := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		kubermatic.ClusterRoleBindingReconciler(config),
		kubermatic.APIClusterRoleBindingReconciler(config),
		common.WebhookClusterRoleBindingReconciler(config),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, reconcilers, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileDeployments(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Deployments")

	reconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		kubermatic.MasterControllerManagerDeploymentReconciler(config, r.workerName, r.versions),
		common.WebhookDeploymentReconciler(config, r.versions, nil, false),
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		reconcilers = append(reconcilers,
			kubermatic.APIDeploymentReconciler(config, r.workerName, r.versions),
			kubermatic.UIDeploymentReconciler(config, r.versions),
		)
	}

	modifiers := []reconciling.ObjectModifier{
		modifier.Ownership(config, common.OperatorName, r.scheme),
		modifier.RelatedRevisionsLabels(ctx, r.Client),
		modifier.VersionLabel(r.versions.GitVersion),
	}

	// add the image pull secret wrapper only when an image pull secret is provided
	if config.Spec.ImagePullSecret != "" {
		modifiers = append(modifiers, reconciling.ImagePullSecretsWrapper(common.DockercfgSecretName))
	}

	if err := reconciling.ReconcileDeployments(ctx, reconcilers, config.Namespace, r.Client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcilePodDisruptionBudgets(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling PodDisruptionBudgets")

	reconcilers := []reconciling.NamedPodDisruptionBudgetReconcilerFactory{
		kubermatic.MasterControllerManagerPDBReconciler(config),
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		reconcilers = append(reconcilers,
			kubermatic.APIPDBReconciler(config),
			kubermatic.UIPDBReconciler(config),
		)
	}

	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, reconcilers, config.Namespace, r.Client, modifier.Ownership(config, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileServices(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Services")

	reconcilers := []reconciling.NamedServiceReconcilerFactory{
		common.WebhookServiceReconciler(config, r.Client),
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		reconcilers = append(reconcilers,
			kubermatic.APIServiceReconciler(config),
			kubermatic.UIServiceReconciler(config),
		)
	}

	if err := reconciling.ReconcileServices(ctx, reconcilers, config.Namespace, r.Client, modifier.Ownership(config, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Services: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileIngresses(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	if config.Spec.Ingress.Disable {
		logger.Debug("Skipping Ingress creation because it was explicitly disabled")
		return nil
	}

	if config.Spec.FeatureGates[features.HeadlessInstallation] {
		logger.Debug("Headless installation requested, skipping.")
		return nil
	}

	logger.Debug("Reconciling Ingresses")

	reconcilers := []reconciling.NamedIngressReconcilerFactory{
		kubermatic.IngressReconciler(config),
	}

	if err := reconciling.ReconcileIngresses(ctx, reconcilers, config.Namespace, r.Client, modifier.Ownership(config, common.OperatorName, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Ingresses: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileValidatingWebhooks(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Validating Webhooks")

	reconcilers := []reconciling.NamedValidatingWebhookConfigurationReconcilerFactory{
		common.SeedAdmissionWebhookReconciler(ctx, config, r.Client),
		common.KubermaticConfigurationAdmissionWebhookReconciler(ctx, config, r.Client),
		kubermatic.UserValidatingWebhookConfigurationReconciler(ctx, config, r.Client),
		common.ApplicationDefinitionValidatingWebhookConfigurationReconciler(ctx, config, r.Client),
		kubermatic.ResourceQuotaValidatingWebhookConfigurationReconciler(ctx, config, r.Client),
		kubermatic.GroupProjectBindingValidatingWebhookConfigurationReconciler(ctx, config, r.Client),
		common.PoliciesWebhookConfigurationReconciler(ctx, config, r.Client),
		common.PolicyTemplateValidatingWebhookConfigurationReconciler(ctx, config, r.Client),
	}

	if !config.Spec.FeatureGates[features.DisableUserSSHKey] {
		reconcilers = append(reconcilers, kubermatic.UserSSHKeyValidatingWebhookConfigurationReconciler(ctx, config, r.Client))
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, reconcilers, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Validating Webhooks: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileMutatingWebhooks(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling Mutating Webhooks")

	reconcilers := []reconciling.NamedMutatingWebhookConfigurationReconcilerFactory{
		kubermatic.ExternalClusterMutatingWebhookConfigurationReconciler(ctx, config, r.Client),
		common.ApplicationDefinitionMutatingWebhookConfigurationReconciler(ctx, config, r.Client),
	}

	if !config.Spec.FeatureGates[features.DisableUserSSHKey] {
		reconcilers = append(reconcilers, kubermatic.UserSSHKeyMutatingWebhookConfigurationReconciler(ctx, config, r.Client))
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, reconcilers, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Mutating Webhooks: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileAddonConfigs(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling AddonConfigs")

	reconcilers := kubermatic.AddonConfigsReconcilers()
	if err := kkpreconciling.ReconcileAddonConfigs(ctx, reconcilers, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile AddonConfigs: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileApplicationDefinitions(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	logger.Debug("Reconciling ApplicationDefinitions")

	reconcilers := []kkpreconciling.NamedApplicationDefinitionReconcilerFactory{
		cilium.ApplicationDefinitionReconciler(config),
	}
	if err := kkpreconciling.ReconcileApplicationDefinitions(ctx, reconcilers, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ApplicationDefinitions: %w", err)
	}

	return nil
}
