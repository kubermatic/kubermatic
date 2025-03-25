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
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	logger.Debug("Reconciling")

	seed := &kubermaticv1.Seed{}
	if err := r.Get(ctx, request.NamespacedName, seed); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get seed: %w", err)
	}

	// do nothing until the operator has prepared the seed cluster
	if !seed.Status.IsInitialized() {
		logger.Debug("Seed cluster has not yet been initialized, skipping.")
		return reconcile.Result{}, nil
	}

	seedClient, err := r.seedClientGetter(seed)
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
		result, err := r.cleanupDeletedSeed(ctx, config, seed, seedClient, logger)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to cleanup deleted Seed: %w", err)
		}
		if result != nil {
			return *result, nil
		}

		return reconcile.Result{}, nil
	}

	if err := r.reconcile(ctx, config, seed, seedClient, logger); err != nil {
		r.recorder.Event(seed, corev1.EventTypeWarning, "ReconcilingFailed", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %w", err)
	}

	logger.Debug("Successfully reconciled")
	return reconcile.Result{}, nil
}

func (r *Reconciler) getKubermaticConfiguration(ctx context.Context, namespace string) (*kubermaticv1.KubermaticConfiguration, error) {
	// retrieve the _undefaulted_ config (which is why this cannot use the KubermaticConfigurationGetter)
	config, err := kubernetesprovider.GetRawKubermaticConfiguration(ctx, r, namespace)

	if errors.Is(err, provider.ErrNoKubermaticConfigurationFound) {
		r.log.Debug("ignoring request for namespace without KubermaticConfiguration")
		return nil, nil
	}

	if errors.Is(err, provider.ErrTooManyKubermaticConfigurationFound) {
		r.log.Warnw("there are multiple KubermaticConfiguration objects, cannot reconcile", "namespace", namespace)
		return nil, nil
	}

	return config, nil
}

func (r *Reconciler) reconcile(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	// ensure we always have a cleanup finalizer on the original
	// Seed CR inside the master cluster
	if err := kubernetes.TryAddFinalizer(ctx, r, seed, CleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	nsReconcilers := []reconciling.NamedNamespaceReconcilerFactory{
		namespaceReconciler(seed.Namespace),
	}

	if err := reconciling.ReconcileNamespaces(ctx, nsReconcilers, "", seedClient); err != nil {
		return fmt.Errorf("failed to reconcile namespace: %w", err)
	}

	seedInSeed := &kubermaticv1.Seed{}
	if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(seed), seedInSeed); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get seed: %w", err)
	}

	// see project-synchronizer's syncAllSeeds comment
	if seedInSeed.UID == "" || seedInSeed.UID != seed.UID {
		seedKubeconfig, err := kubernetesprovider.GetSeedKubeconfigSecret(ctx, r, seed)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig for seed: %w", err)
		}

		seedKubeconfigReconcilers := []reconciling.NamedSecretReconcilerFactory{
			secretReconciler(seedKubeconfig),
		}

		if err := reconciling.ReconcileSecrets(ctx, seedKubeconfigReconcilers, seedKubeconfig.Namespace, seedClient); err != nil {
			return fmt.Errorf("failed to reconcile seed kubeconfig: %w", err)
		}

		seedReconcilers := []kkpreconciling.NamedSeedReconcilerFactory{
			seedReconciler(seed),
		}

		if err := kkpreconciling.ReconcileSeeds(ctx, seedReconcilers, seed.Namespace, seedClient); err != nil {
			return fmt.Errorf("failed to reconcile seed: %w", err)
		}
	}

	configReconcilers := []kkpreconciling.NamedKubermaticConfigurationReconcilerFactory{
		configReconciler(config),
	}

	if err := kkpreconciling.ReconcileKubermaticConfigurations(ctx, configReconcilers, seed.Namespace, seedClient); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic configuration: %w", err)
	}

	return nil
}

// cleanupDeletedSeed is triggered when a Seed CR inside the master cluster has been deleted
// and is responsible for removing the Seed CR copy inside the seed cluster. This can end up
// in a Retry if other components like the Kubermatic Operator still have finalizers on the
// Seed CR copy.
func (r *Reconciler) cleanupDeletedSeed(ctx context.Context, configInMaster *kubermaticv1.KubermaticConfiguration, seedInMaster *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, logger *zap.SugaredLogger) (*reconcile.Result, error) {
	if !kubernetes.HasAnyFinalizer(seedInMaster, CleanupFinalizer) {
		return nil, nil
	}

	logger.Debug("Seed was deleted, removing copy in seed cluster")

	seedKey := ctrlruntimeclient.ObjectKeyFromObject(seedInMaster)
	configKey := ctrlruntimeclient.ObjectKeyFromObject(configInMaster)

	// when master==seed cluster, this is the same as seedInMaster
	seedInSeed := &kubermaticv1.Seed{}
	err := seedClient.Get(ctx, seedKey, seedInSeed)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %w", seedKey, err)
	}

	// if the copy still exists, attempt to delete it unless it has only our own finalizer
	// (we have a master==seed situation) and deleting it again would be futile.
	if err == nil && !kubernetes.HasOnlyFinalizer(seedInSeed, CleanupFinalizer) {
		if seedInSeed.DeletionTimestamp == nil {
			logger.Debug("Issuing DELETE call for Seed copy now")
			if err := seedClient.Delete(ctx, seedInSeed); err != nil {
				return nil, fmt.Errorf("failed to delete Seed %s in seed cluster: %w", seedKey, err)
			}
		} else {
			logger.Debug("Seed copy does still exist, requeuing.")
		}

		return &reconcile.Result{
			// cleanup in remote seed clusters can be slow over long distances
			RequeueAfter: 3 * time.Second,
		}, nil
	}

	// when master==seed cluster, this is the same as configInMaster
	configInSeed := &kubermaticv1.KubermaticConfiguration{}
	err = seedClient.Get(ctx, configKey, configInSeed)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %w", configKey, err)
	}

	// if the config on the seed is not identical to the one on the master,
	// delete it now
	if err == nil && configInMaster.UID != configInSeed.UID {
		if configInSeed.DeletionTimestamp == nil {
			logger.Debug("Issuing DELETE call for KubermaticConfiguration copy now")
			if err := seedClient.Delete(ctx, configInSeed); err != nil {
				return nil, fmt.Errorf("failed to delete KubermaticConfiguration %s in seed cluster: %w", configKey, err)
			}
		} else {
			logger.Debug("KubermaticConfiguration copy does still exist, requeuing.")
		}

		return &reconcile.Result{
			// cleanup in remote seed clusters can be slow over long distances
			RequeueAfter: 3 * time.Second,
		}, nil
	}

	// at this point either the Seed CR copy is gone or it has only our own finalizer left
	if err := kubernetes.TryRemoveFinalizer(ctx, r, seedInMaster, CleanupFinalizer); err != nil {
		return nil, fmt.Errorf("failed to remove finalizer from Seed in master cluster: %w", err)
	}

	return nil, nil
}
