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

package seedsync

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler copies seed CRs into their respective clusters,
// assuming that Kubermatic and the seed CRD have already been
// installed.
type Reconciler struct {
	ctrlruntimeclient.Client

	seedClientGetter provider.SeedClientGetter
	log              *zap.SugaredLogger
	recorder         record.EventRecorder
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.log.With("seed", request.Name)
	logger.Info("Reconciling seed cluster")

	seed := &kubermaticv1.Seed{}
	if err := r.Get(ctx, request.NamespacedName, seed); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get seed: %w", err)
	}

	client, err := r.seedClientGetter(seed)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create client for seed: %w", err)
	}

	config, err := r.getKubermaticConfiguration(ctx, request.Namespace)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get config: %w", err)
	}

	// not having a config at all should never really happen
	if config == nil {
		return reconcile.Result{}, nil
	}

	// cleanup once a Seed was deleted in the master cluster
	if seed.DeletionTimestamp != nil {
		result, err := r.cleanupDeletedSeed(ctx, config, seed, client, logger)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to cleanup deleted Seed: %w", err)
		}
		if result != nil {
			return *result, nil
		}

		return reconcile.Result{}, nil
	}

	if err := r.reconcile(ctx, config, seed, client, logger); err != nil {
		r.recorder.Eventf(seed, corev1.EventTypeWarning, "ReconcilingFailed", "%v", err)
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %w", err)
	}

	logger.Info("Successfully reconciled")
	return reconcile.Result{}, nil
}

func (r *Reconciler) getKubermaticConfiguration(ctx context.Context, namespace string) (*operatorv1alpha1.KubermaticConfiguration, error) {
	configList := &operatorv1alpha1.KubermaticConfigurationList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace: namespace,
	}

	if err := r.List(ctx, configList, listOpts); err != nil {
		return nil, fmt.Errorf("failed to find KubermaticConfigurations: %w", err)
	}

	if len(configList.Items) == 0 {
		r.log.Debug("ignoring request for namespace without KubermaticConfiguration")
		return nil, nil
	}

	if len(configList.Items) > 1 {
		r.log.Warnw("there are multiple KubermaticConfiguration objects, cannot reconcile", "namespace", namespace)
		return nil, nil
	}

	return &configList.Items[0], nil
}

func (r *Reconciler) reconcile(ctx context.Context, config *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	// ensure we always have a cleanup finalizer on the original
	// Seed CR inside the master cluster
	oldSeed := seed.DeepCopy()
	kubernetes.AddFinalizer(seed, CleanupFinalizer)
	if err := r.Patch(ctx, seed, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
		return fmt.Errorf("failed to add finalizer to Seed: %w", err)
	}

	if seed.Spec.ExposeStrategy != "" {
		if !kubermaticv1.AllExposeStrategies.Has(seed.Spec.ExposeStrategy) {
			return fmt.Errorf("failed to validate seed: invalid expose strategy %q, must be one of %v", seed.Spec.ExposeStrategy, kubermaticv1.AllExposeStrategies)
		}
	}

	seedCreators := []reconciling.NamedSeedCreatorGetter{
		seedCreator(seed),
	}

	if err := reconciling.ReconcileSeeds(ctx, seedCreators, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile seed: %w", err)
	}

	configCreators := []reconciling.NamedKubermaticConfigurationCreatorGetter{
		configCreator(config),
	}

	if err := reconciling.ReconcileKubermaticConfigurations(ctx, configCreators, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile seed: %w", err)
	}

	return nil
}

// cleanupDeletedSeed is triggered when a Seed CR inside the master cluster has been deleted
// and is responsible for removing the Seed CR copy inside the seed cluster. This can end up
// in a Retry if other components like the Kubermatic Operator still have finalizers on the
// Seed CR copy.
func (r *Reconciler) cleanupDeletedSeed(ctx context.Context, configInMaster *operatorv1alpha1.KubermaticConfiguration, seedInMaster *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, logger *zap.SugaredLogger) (*reconcile.Result, error) {
	if !kubernetes.HasAnyFinalizer(seedInMaster, CleanupFinalizer) {
		return nil, nil
	}

	logger.Debug("Seed was deleted, removing copy in seed cluster")

	seedKey := ctrlruntimeclient.ObjectKeyFromObject(seedInMaster)
	configKey := ctrlruntimeclient.ObjectKeyFromObject(configInMaster)

	// when master==seed cluster, this is the same as seedInMaster
	seedInSeed := &kubermaticv1.Seed{}

	err := seedClient.Get(ctx, seedKey, seedInSeed)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %w", seedKey, err)
	}

	configInSeed := &operatorv1alpha1.KubermaticConfiguration{}

	err = seedClient.Get(ctx, configKey, configInSeed)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %w", configKey, err)
	}

	// if the copy still exists, attempt to delete it unless it has only our own finalizer
	// (we have a master==seed situation) and deleting it again would be futile.
	if err == nil && !kubernetes.HasOnlyFinalizer(seedInSeed, CleanupFinalizer) {
		logger.Debug("Issuing DELETE call for Seed copy now")
		if err := seedClient.Delete(ctx, seedInSeed); err != nil {
			return nil, fmt.Errorf("failed to delete %s: %w", seedKey, err)
		}

		// older KKP setups do not yet sync the config into each seed, so it's fine
		// and expected if the config doesn't yet exist
		if configInSeed.Name != "" {
			logger.Debug("Issuing DELETE call for KubermaticConfiguration copy now")
			if err := seedClient.Delete(ctx, configInSeed); err != nil {
				return nil, fmt.Errorf("failed to delete %s: %w", configKey, err)
			}
		}

		return &reconcile.Result{
			// cleanup in remote seed clusters can be slow over long distances
			RequeueAfter: 3 * time.Second,
		}, nil
	}

	// at this point either the Seed CR copy is gone or it has only our own finalizer left
	oldSeed := seedInMaster.DeepCopy()
	kubernetes.RemoveFinalizer(seedInMaster, CleanupFinalizer)

	if err := r.Patch(ctx, seedInMaster, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
		return nil, fmt.Errorf("failed to remove finalizer from Seed in master cluster: %w", err)
	}

	return nil, nil
}
