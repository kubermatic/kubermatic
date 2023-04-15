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

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/cni/cilium"
	operatorresources "k8c.io/kubermatic/v3/pkg/controller/operator/seed/resources"
	kubermaticseed "k8c.io/kubermatic/v3/pkg/controller/operator/seed/resources/kubermatic"
	"k8c.io/kubermatic/v3/pkg/controller/operator/seed/resources/nodeportproxy"
	"k8c.io/kubermatic/v3/pkg/crd"
	"k8c.io/kubermatic/v3/pkg/features"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/resources/certificates"
	kkpreconciling "k8c.io/kubermatic/v3/pkg/resources/reconciling"
	kubermaticversion "k8c.io/kubermatic/v3/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for running a Kubermatic
// seed cluster.
type Reconciler struct {
	log          *zap.SugaredLogger
	scheme       *runtime.Scheme
	namespace    string
	seedClient   ctrlruntimeclient.Client
	seedRecorder record.EventRecorder
	configGetter provider.KubermaticConfigurationGetter
	workerName   string
	versions     kubermaticversion.Versions
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Reconciling")

	err := r.reconcile(ctx, r.log)
	if err != nil {
		r.log.Errorw("failed to reconcile", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger) error {
	// find the owning KubermaticConfiguration
	config, err := r.configGetter(ctx)
	if err != nil || config == nil {
		return err
	}

	if config.DeletionTimestamp != nil {
		return r.cleanupDeletedConfiguration(ctx, log, config)
	}

	if err := r.reconcileResources(ctx, log, config); err != nil {
		r.seedRecorder.Event(config, corev1.EventTypeWarning, "ReconcilingError", err.Error())

		// TODO: Maybe set a condition on the config?
		// if err := r.setKubermaticConfigurationCondition(ctx, config, corev1.ConditionFalse, "ReconcilingError", err.Error()); err != nil {
		// 	log.Errorw("Failed to update configuration status", zap.Error(err))
		// }

		return err
	}

	// TODO: Maybe set a condition on the config?
	// if err := r.setKubermaticConfigurationCondition(ctx, config, corev1.ConditionTrue, "ReconcilingSuccess", ""); err != nil {
	// 	log.Errorw("Failed to update configuration status", zap.Error(err))
	// }

	return nil
}

func (r *Reconciler) getRawConfiguration(ctx context.Context, config *kubermaticv1.KubermaticConfiguration) (*kubermaticv1.KubermaticConfiguration, error) {
	// For managing finalizers we need access to the raw KubermaticConfiguration without
	// the defaulting being applied to it, as we otherwise might accidentally persist
	// runtime defaults.
	// There is a RawConfigurationGetter available, but it's meant for read-only usecases.
	// As this controller is actively managing the configuration, it only uses a getter
	// to determine the name/namespace of the relevant configuration.
	rawConfig := &kubermaticv1.KubermaticConfiguration{}
	if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(config), rawConfig); err != nil {
		return nil, err
	}

	return rawConfig, nil
}

func (r *Reconciler) removeCleanupFinalizer(ctx context.Context, config *kubermaticv1.KubermaticConfiguration) error {
	rawConfig, err := r.getRawConfiguration(ctx, config)
	if err != nil {
		return err
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, rawConfig, operatorresources.CleanupFinalizer)
}

func (r *Reconciler) addCleanupFinalizer(ctx context.Context, config *kubermaticv1.KubermaticConfiguration) error {
	rawConfig, err := r.getRawConfiguration(ctx, config)
	if err != nil {
		return err
	}

	return kubernetes.TryAddFinalizer(ctx, r.seedClient, rawConfig, operatorresources.CleanupFinalizer)
}

func (r *Reconciler) cleanupDeletedConfiguration(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	if !kubernetes.HasAnyFinalizer(config, operatorresources.CleanupFinalizer) {
		return nil
	}

	// Note that this function does not remove CRDs, not because we're lazy, but because
	// it's safer to keep them around and old resources do not hurt cluster performance.

	log.Debug("Configuration was deleted, cleaning up cluster-wide resources")

	if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &rbacv1.ClusterRoleBinding{}, kubermaticseed.ClusterRoleBindingName(config)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &rbacv1.ClusterRoleBinding{}, nodeportproxy.ClusterRoleBindingName(config)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &rbacv1.ClusterRoleBinding{}, kubermaticseed.APIClusterRoleBindingName(config)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &rbacv1.ClusterRoleBinding{}, operatorresources.WebhookClusterRoleBindingName(config)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRoleBinding: %w", err)
	}

	if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &rbacv1.ClusterRole{}, nodeportproxy.ClusterRoleName(config)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRole: %w", err)
	}

	if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &rbacv1.ClusterRole{}, operatorresources.WebhookClusterRoleName(config)); err != nil {
		return fmt.Errorf("failed to clean up ClusterRole: %w", err)
	}

	validating := []string{
		operatorresources.ApplicationDefinitionAdmissionWebhookName,
		operatorresources.GroupProjectBindingAdmissionWebhookName,
		operatorresources.KubermaticConfigurationAdmissionWebhookName(config),
		operatorresources.ResourceQuotaAdmissionWebhookName,
		operatorresources.SeedAdmissionWebhookName(config),
		operatorresources.UserAdmissionWebhookName,
		operatorresources.UserSSHKeyAdmissionWebhookName,
		kubermaticseed.ClusterAdmissionWebhookName,
		kubermaticseed.IPAMPoolAdmissionWebhookName,
	}

	mutating := []string{
		operatorresources.ExternalClusterAdmissionWebhookName,
		operatorresources.ResourceQuotaAdmissionWebhookName,
		operatorresources.UserSSHKeyAdmissionWebhookName,
		kubermaticseed.AddonAdmissionWebhookName,
		kubermaticseed.MLAAdminSettingAdmissionWebhookName,
	}

	for _, webhook := range validating {
		if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &admissionregistrationv1.ValidatingWebhookConfiguration{}, webhook); err != nil {
			return fmt.Errorf("failed to clean up validating webhook for %q: %w", webhook, err)
		}
	}

	for _, webhook := range mutating {
		if err := operatorresources.CleanupClusterResource(ctx, r.seedClient, &admissionregistrationv1.MutatingWebhookConfiguration{}, webhook); err != nil {
			return fmt.Errorf("failed to clean up mutating webhook for %q: %w", webhook, err)
		}
	}

	return r.removeCleanupFinalizer(ctx, config)
}

// func (r *Reconciler) setKubermaticConfigurationCondition(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, status corev1.ConditionStatus, reason string, message string) error {
// 	return kubernetes.UpdateKubermaticConfigurationStatus(ctx, r.masterClient, config, func(s *kubermaticv1.KubermaticConfiguration) {
// 		kubermaticv1helper.SetKubermaticConfigurationCondition(s, kubermaticv1.KubermaticConfigurationConditionResourcesReconciled, status, reason, message)
// 	})
// }

func (r *Reconciler) reconcileResources(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	if err := r.addCleanupFinalizer(ctx, config); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	caBundle, err := certificates.GlobalCABundle(ctx, r.seedClient, config)
	if err != nil {
		return fmt.Errorf("failed to get CA bundle ConfigMap: %w", err)
	}

	// TODO: Do we still want this?
	if err := r.reconcileCRDs(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileServiceAccounts(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileRoles(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileRoleBindings(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileClusterRoles(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(ctx, log, config, caBundle); err != nil {
		return err
	}

	if err := r.reconcileSecrets(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileAdmissionWebhooks(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileDeployments(ctx, log, config, caBundle); err != nil {
		return err
	}

	if err := r.reconcilePodDisruptionBudgets(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileServices(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileIngresses(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileAddonConfigs(ctx, log, config); err != nil {
		return err
	}

	if err := r.reconcileApplicationDefinitions(ctx, log, config); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) reconcileCRDs(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
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

		for i := range crds {
			creators = append(creators, operatorresources.CRDReconciler(&crds[i], log, r.versions))
		}
	}

	if err := kkpreconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile CRDs: %w", err)
	}

	return nil
}

func nodePortProxyEnabled(config *kubermaticv1.KubermaticConfiguration) bool {
	return config.Spec.NodeportProxy != nil && !config.Spec.NodeportProxy.Disable
}

func (r *Reconciler) reconcileServiceAccounts(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling Kubermatic ServiceAccounts")

	creators := []reconciling.NamedServiceAccountReconcilerFactory{
		operatorresources.WebhookServiceAccountReconciler(config),
		kubermaticseed.APIServiceAccountReconciler(),
		kubermaticseed.ServiceAccountReconciler(config),
	}

	if nodePortProxyEnabled(config) {
		creators = append(creators, nodeportproxy.ServiceAccountReconciler(config))
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, r.namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic ServiceAccounts: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileRoles(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling Roles")

	creators := []reconciling.NamedRoleReconcilerFactory{
		operatorresources.WebhookRoleReconciler(config),
		kubermaticseed.APIRoleReconciler(),
	}

	if nodePortProxyEnabled(config) {
		creators = append(creators, nodeportproxy.RoleReconciler())
	}

	if err := reconciling.ReconcileRoles(ctx, creators, r.namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileRoleBindings(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling RoleBindings")

	creators := []reconciling.NamedRoleBindingReconcilerFactory{
		operatorresources.WebhookRoleBindingReconciler(config),
		kubermaticseed.APIRoleBindingReconciler(),
	}

	if nodePortProxyEnabled(config) {
		creators = append(creators, nodeportproxy.RoleBindingReconciler(config))
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, r.namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoles(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling ClusterRoles")

	creators := []reconciling.NamedClusterRoleReconcilerFactory{
		operatorresources.WebhookClusterRoleReconciler(config),
		kubermaticseed.APIClusterRoleReconciler(config),
	}

	if nodePortProxyEnabled(config) {
		creators = append(creators, nodeportproxy.ClusterRoleReconciler(config))
	}

	if err := reconciling.ReconcileClusterRoles(ctx, creators, "", r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileClusterRoleBindings(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		operatorresources.WebhookClusterRoleBindingReconciler(config),
		kubermaticseed.APIClusterRoleBindingReconciler(config),
		kubermaticseed.ClusterRoleBindingReconciler(config),
	}

	if nodePortProxyEnabled(config) {
		creators = append(creators, nodeportproxy.ClusterRoleBindingReconciler(config))
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, creators, "", r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMaps(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration, caBundle *corev1.ConfigMap) error {
	log.Debug("reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapReconcilerFactory{
		kubermaticseed.CABundleConfigMapReconciler(caBundle),
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		creators = append(creators, kubermaticseed.UIConfigConfigMapReconciler(config))
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, config.Namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling Secrets")

	creators := []reconciling.NamedSecretReconcilerFactory{
		operatorresources.WebhookServingCASecretReconciler(config),
		operatorresources.WebhookServingCertSecretReconciler(ctx, config, r.seedClient),
	}

	if config.Spec.ImagePullSecret != "" {
		creators = append(creators, operatorresources.DockercfgSecretReconciler(config))
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, config.Namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileDeployments(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration, caBundle *corev1.ConfigMap) error {
	log.Debug("reconciling Deployments")

	creators := []reconciling.NamedDeploymentReconcilerFactory{
		kubermaticseed.SeedControllerManagerDeploymentReconciler(r.workerName, r.versions, config),
		operatorresources.WebhookDeploymentReconciler(config, r.versions),
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		creators = append(creators,
			kubermaticseed.APIDeploymentReconciler(config, r.workerName, r.versions),
			kubermaticseed.UIDeploymentReconciler(config, r.versions),
		)
	}

	if nodePortProxyEnabled(config) {
		supportsFailureDomainZoneAntiAffinity, err := resources.SupportsFailureDomainZoneAntiAffinity(ctx, r.seedClient)
		if err != nil {
			return err
		}

		creators = append(
			creators,
			nodeportproxy.EnvoyDeploymentReconciler(config, supportsFailureDomainZoneAntiAffinity, r.versions),
			nodeportproxy.UpdaterDeploymentReconciler(config, r.versions),
		)
	}

	volumeLabelModifier := kkpreconciling.NewVolumeRevisionLabelsModifier(ctx, r.seedClient)
	modifiers := []reconciling.ObjectModifier{
		kkpreconciling.NewOwnershipModifier(config, r.scheme),
		volumeLabelModifier,
	}

	// add the image pull secret wrapper only when an image pull secret is provided
	if config.Spec.ImagePullSecret != "" {
		modifiers = append(modifiers, reconciling.ImagePullSecretsWrapper(operatorresources.DockercfgSecretName))
	}

	if err := reconciling.ReconcileDeployments(ctx, creators, r.namespace, r.seedClient, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Deployments: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcilePodDisruptionBudgets(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling PodDisruptionBudgets")

	creators := []reconciling.NamedPodDisruptionBudgetReconcilerFactory{
		kubermaticseed.SeedControllerManagerPDBReconciler(config),
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		creators = append(creators,
			kubermaticseed.APIPDBReconciler(config),
			kubermaticseed.UIPDBReconciler(config),
		)
	}

	if nodePortProxyEnabled(config) {
		creators = append(creators, nodeportproxy.EnvoyPDBReconciler())
	}

	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, config.Namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileServices(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling Services")

	creators := []reconciling.NamedServiceReconcilerFactory{
		operatorresources.WebhookServiceReconciler(config, r.seedClient),
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		creators = append(creators,
			kubermaticseed.APIServiceReconciler(config),
			kubermaticseed.UIServiceReconciler(config),
		)
	}

	if err := reconciling.ReconcileServices(ctx, creators, config.Namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Services: %w", err)
	}

	// The nodeport-proxy LoadBalancer is not given an owner reference, so in case someone accidentally deletes
	// the Seed resource, the current LoadBalancer IP is not lost. To be truly destructive, users would need to
	// remove the entire Kubermatic namespace.
	if nodePortProxyEnabled(config) {
		creators = []reconciling.NamedServiceReconcilerFactory{
			nodeportproxy.ServiceReconciler(config),
		}

		if err := reconciling.ReconcileServices(ctx, creators, config.Namespace, r.seedClient); err != nil {
			return fmt.Errorf("failed to reconcile nodeport-proxy Services: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) reconcileIngresses(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	if config.Spec.Ingress.Disable {
		log.Debug("Skipping Ingress creation because it was explicitly disabled")
		return nil
	}

	if config.Spec.FeatureGates[features.HeadlessInstallation] {
		log.Debug("Headless installation requested, skipping.")
		return nil
	}

	log.Debug("Reconciling Ingresses")

	reconcilers := []reconciling.NamedIngressReconcilerFactory{
		kubermaticseed.IngressReconciler(config),
	}

	if err := reconciling.ReconcileIngresses(ctx, reconcilers, config.Namespace, r.seedClient, kkpreconciling.NewOwnershipModifier(config, r.scheme)); err != nil {
		return fmt.Errorf("failed to reconcile Ingresses: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileAdmissionWebhooks(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("reconciling Admission Webhooks")

	validatingWebhookReconcilers := []reconciling.NamedValidatingWebhookConfigurationReconcilerFactory{
		operatorresources.ApplicationDefinitionValidatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		operatorresources.KubermaticConfigurationAdmissionWebhookReconciler(ctx, config, r.seedClient),
		operatorresources.SeedAdmissionWebhookReconciler(ctx, config, r.seedClient),
		kubermaticseed.ClusterValidatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		kubermaticseed.IPAMPoolValidatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		kubermaticseed.UserSSHKeyValidatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		kubermaticseed.UserValidatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
	}

	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, validatingWebhookReconcilers, "", r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile validating Admission Webhooks: %w", err)
	}

	mutatingWebhookReconcilers := []reconciling.NamedMutatingWebhookConfigurationReconcilerFactory{
		operatorresources.ApplicationDefinitionMutatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		kubermaticseed.AddonMutatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		kubermaticseed.ClusterMutatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		kubermaticseed.MLAAdminSettingMutatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
		kubermaticseed.UserSSHKeyMutatingWebhookConfigurationReconciler(ctx, config, r.seedClient),
	}

	if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, mutatingWebhookReconcilers, "", r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile mutating Admission Webhooks: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileAddonConfigs(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("Reconciling AddonConfigs")

	reconcilers := kubermaticseed.AddonConfigsReconcilers()
	if err := kkpreconciling.ReconcileAddonConfigs(ctx, reconcilers, "", r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile AddonConfigs: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileApplicationDefinitions(ctx context.Context, log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration) error {
	log.Debug("Reconciling ApplicationDefinitions")

	reconcilers := []kkpreconciling.NamedApplicationDefinitionReconcilerFactory{
		cilium.ApplicationDefinitionReconciler(config),
	}
	if err := kkpreconciling.ReconcileApplicationDefinitions(ctx, reconcilers, "", r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile ApplicationDefinitions: %w", err)
	}

	return nil
}
