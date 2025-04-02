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

package seedproxy

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for proxying requests to
// seed clusters. It also takes care of creating a nice ConfigMap for
// Grafana's provisioning mechanism.
type Reconciler struct {
	ctrlruntimeclient.Client

	log       *zap.SugaredLogger
	namespace string

	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	seedClientGetter     provider.SeedClientGetter
	configGetter         provider.KubermaticConfigurationGetter

	recorder record.EventRecorder
}

// Reconcile acts upon requests and will restore the state of resources
// for the given seed cluster context (the request's name). Will return
// an error if any API operation failed, otherwise will return an empty
// dummy Result struct.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("seed", request.String())
	log.Debug("reconciling seed")

	return reconcile.Result{}, r.reconcile(ctx, request.Name, log)
}

func (r *Reconciler) reconcile(ctx context.Context, seedName string, log *zap.SugaredLogger) error {
	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %w", err)
	}

	log.Debug("garbage-collecting orphaned resources...")
	if err := r.garbageCollect(ctx, seeds, log); err != nil {
		return fmt.Errorf("failed to garbage collect: %w", err)
	}

	seed, found := seeds[seedName]
	if !found {
		return fmt.Errorf("didn't find seed %q", seedName)
	}

	// do nothing until the seed-status-controller has validated the kubeconfig
	if !seed.Status.HasConditionValue(kubermaticv1.SeedConditionKubeconfigValid, corev1.ConditionTrue) {
		log.Debug("Seed cluster has not yet been initialized, skipping.")
		return nil
	}

	client, err := r.seedClientGetter(seed)
	if err != nil {
		return fmt.Errorf("failed to get seed client: %w", err)
	}

	err = client.Get(ctx, types.NamespacedName{Name: seed.Namespace}, &corev1.Namespace{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to check for namespace %s in seed cluster: %w", seed.Namespace, err)
		}

		log.Debug("skipping because seed namespace does not exist", "namespace", seed.Namespace)
		return nil
	}

	log.Debug("reconciling seed cluster...")
	if err := r.reconcileSeedProxy(ctx, seed, client, log); err != nil {
		return fmt.Errorf("failed to reconcile: %w", err)
	}

	log.Debug("reconciling Grafana provisioning...")
	if err := r.reconcileMasterGrafanaProvisioning(ctx, seeds, log); err != nil {
		return fmt.Errorf("failed to reconcile Grafana: %w", err)
	}

	log.Debug("successfully reconciled")
	return nil
}

// garbageCollect finds secrets referencing non-existing seeds and deletes
// those. It relies on the owner references on all other master-cluster
// resources to let the apiserver remove them automatically.
func (r *Reconciler) garbageCollect(ctx context.Context, seeds map[string]*kubermaticv1.Seed, log *zap.SugaredLogger) error {
	list := &corev1.SecretList{}
	options := &ctrlruntimeclient.ListOptions{
		Namespace: r.namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{
			ManagedByLabel: ControllerName,
		}),
	}

	if err := r.List(ctx, list, options); err != nil {
		return fmt.Errorf("failed to list Secrets: %w", err)
	}

	for _, item := range list.Items {
		seed := item.Labels[InstanceLabel]

		if _, exists := seeds[seed]; !exists {
			log.Debugw("deleting orphaned Secret referencing non-existing seed", "secret", item, "seed", seed)
			if err := r.Delete(ctx, &item); err != nil {
				return fmt.Errorf("failed to delete Secret: %w", err)
			}
		}
	}

	return nil
}

func (r *Reconciler) reconcileSeedProxy(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	cfg, err := r.seedKubeconfigGetter(seed)
	if err != nil {
		return err
	}

	log.Debug("reconciling ServiceAccounts...")
	if err := r.reconcileSeedServiceAccounts(ctx, seed, client, log); err != nil {
		return fmt.Errorf("failed to ensure ServiceAccount: %w", err)
	}

	// Since Kubernetes 1.24, the LegacyServiceAccountTokenNoAutoGeneration was enabled
	// by default. To ensure the old behaviour, KKP has to create the token secrets
	// itself and wait for Kubernetes to fill in the token details.
	// On clusters using older Kubernetes versions, this code will create a second Secret
	// (next to the auto-generated one) and will use the token from it, ignoring the
	// auto-generated Secret entirely.
	log.Debug("reconciling Secrets...")
	if err := r.reconcileSeedSecrets(ctx, seed, client, log); err != nil {
		return fmt.Errorf("failed to ensure Secret: %w", err)
	}

	log.Debug("reconciling RBAC...")
	if err := r.reconcileSeedRBAC(ctx, seed, client, log); err != nil {
		return fmt.Errorf("failed to ensure RBAC: %w", err)
	}

	log.Debug("fetching ServiceAccount details from seed cluster...")
	serviceAccountSecret, err := r.fetchServiceAccountSecret(ctx, seed, client, log)
	if err != nil {
		return fmt.Errorf("failed to fetch ServiceAccount Secret: %w", err)
	}

	if err := r.reconcileMaster(ctx, seed, cfg, serviceAccountSecret, log); err != nil {
		return fmt.Errorf("failed to reconcile master cluster: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileSeedServiceAccounts(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedServiceAccountReconcilerFactory{
		seedServiceAccountReconciler(seed),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %w", seed.Namespace, err)
	}

	return nil
}

func (r *Reconciler) reconcileSeedSecrets(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedSecretReconcilerFactory{
		seedSecretReconciler(seed),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in the namespace %s: %w", seed.Namespace, err)
	}

	return nil
}

func (r *Reconciler) reconcileSeedRoles(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedRoleReconcilerFactory{
		seedMonitoringRoleReconciler(seed),
	}

	if err := reconciling.ReconcileRoles(ctx, creators, SeedMonitoringNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %w", SeedMonitoringNamespace, err)
	}

	return nil
}

func (r *Reconciler) reconcileSeedRoleBindings(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedRoleBindingReconcilerFactory{
		seedMonitoringRoleBindingReconciler(seed),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, SeedMonitoringNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in the namespace %s: %w", SeedMonitoringNamespace, err)
	}

	return nil
}

func (r *Reconciler) reconcileSeedRBAC(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	err := client.Get(ctx, types.NamespacedName{Name: SeedMonitoringNamespace}, &corev1.Namespace{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Debugw("skipping RBAC setup because monitoring namespace does not exist in master", "namespace", SeedMonitoringNamespace)
			return nil
		}

		return fmt.Errorf("failed to check for namespace %s: %w", SeedMonitoringNamespace, err)
	}

	log.Debug("reconciling Roles...")
	if err := r.reconcileSeedRoles(ctx, seed, client, log); err != nil {
		return fmt.Errorf("failed to ensure Role: %w", err)
	}

	log.Debug("reconciling RoleBindings...")
	if err := r.reconcileSeedRoleBindings(ctx, seed, client, log); err != nil {
		return fmt.Errorf("failed to ensure RoleBinding: %w", err)
	}

	return nil
}

func (r *Reconciler) fetchServiceAccountSecret(ctx context.Context, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	name := types.NamespacedName{
		Namespace: seed.Namespace,
		Name:      SeedSecretName,
	}

	if err := client.Get(ctx, name, secret); err != nil {
		return nil, fmt.Errorf("failed to retrieve token secret: %w", err)
	}

	return secret, nil
}

func (r *Reconciler) reconcileMaster(ctx context.Context, seed *kubermaticv1.Seed, kubeconfig *rest.Config, credentials *corev1.Secret, log *zap.SugaredLogger) error {
	log.Debug("reconciling Secrets...")
	secret, err := r.reconcileMasterSecrets(ctx, seed, kubeconfig, credentials)
	if err != nil {
		return fmt.Errorf("failed to ensure Secrets: %w", err)
	}

	log.Debug("reconciling Deployments...")
	if err := r.reconcileMasterDeployments(ctx, seed, secret); err != nil {
		return fmt.Errorf("failed to ensure Deployments: %w", err)
	}

	log.Debug("reconciling Services...")
	if err := r.reconcileMasterServices(ctx, seed, secret); err != nil {
		return fmt.Errorf("failed to ensure Services: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcileMasterSecrets(ctx context.Context, seed *kubermaticv1.Seed, kubeconfig *rest.Config, credentials *corev1.Secret) (*corev1.Secret, error) {
	creators := []reconciling.NamedSecretReconcilerFactory{
		masterSecretReconciler(seed, kubeconfig, credentials),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, seed.Namespace, r.Client); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets in the namespace %s: %w", seed.Namespace, err)
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{
		Namespace: seed.Namespace,
		Name:      secretName(seed),
	}

	if err := r.Get(ctx, name, secret); err != nil {
		return nil, fmt.Errorf("could not find Secret '%s'", name)
	}

	return secret, nil
}

func (r *Reconciler) reconcileMasterDeployments(ctx context.Context, seed *kubermaticv1.Seed, secret *corev1.Secret) error {
	config, err := r.configGetter(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve KubermaticConfiguration: %w", err)
	}

	creators := []reconciling.NamedDeploymentReconcilerFactory{
		masterDeploymentReconciler(seed, secret, registry.GetImageRewriterFunc(config.Spec.UserCluster.OverwriteRegistry)),
	}

	if err := reconciling.ReconcileDeployments(ctx, creators, seed.Namespace, r); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in the namespace %s: %w", seed.Namespace, err)
	}

	return nil
}

func (r *Reconciler) reconcileMasterServices(ctx context.Context, seed *kubermaticv1.Seed, secret *corev1.Secret) error {
	creators := []reconciling.NamedServiceReconcilerFactory{
		masterServiceReconciler(seed, secret),
	}

	if err := reconciling.ReconcileServices(ctx, creators, seed.Namespace, r); err != nil {
		return fmt.Errorf("failed to reconcile Services in the namespace %s: %w", seed.Namespace, err)
	}

	return nil
}

func (r *Reconciler) reconcileMasterGrafanaProvisioning(ctx context.Context, seeds map[string]*kubermaticv1.Seed, log *zap.SugaredLogger) error {
	err := r.Get(ctx, types.NamespacedName{Name: MasterGrafanaNamespace}, &corev1.Namespace{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to check for namespace %s: %w", MasterGrafanaNamespace, err)
		}

		log.Debugw("skipping Grafana setup because namespace does not exist in master", "namespace", MasterGrafanaNamespace)
		return nil
	}

	creators := []reconciling.NamedConfigMapReconcilerFactory{
		r.masterGrafanaConfigmapReconciler(seeds),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, MasterGrafanaNamespace, r); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in the namespace %s: %w", MasterGrafanaNamespace, err)
	}

	return nil
}
