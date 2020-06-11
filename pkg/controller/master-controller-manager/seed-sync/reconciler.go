package seedsync

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

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
	ctx              context.Context
	recorder         record.EventRecorder
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := r.log.With("seed", request.Name)
	logger.Info("Reconciling seed")

	seed := &kubermaticv1.Seed{}
	if err := r.Get(r.ctx, request.NamespacedName, seed); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get seed: %v", err)
	}

	client, err := r.seedClientGetter(seed)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create client for seed: %v", err)
	}

	// cleanup once a Seed was deleted in the master cluster
	if seed.DeletionTimestamp != nil {
		result, err := r.cleanupDeletedSeed(seed, client, logger)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to cleanup deleted Seed: %v", err)
		}
		if result != nil {
			return *result, nil
		}

		return reconcile.Result{}, nil
	}

	if err := r.reconcile(seed, client, logger); err != nil {
		r.recorder.Eventf(seed, corev1.EventTypeWarning, "ReconcilingFailed", "%v", err)
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %v", err)
	}

	logger.Info("Successfully reconciled")
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	// ensure we always have a cleanup finalizer on the original
	// Seed CR inside the master cluster
	oldSeed := seed.DeepCopy()
	kubernetes.AddFinalizer(seed, CleanupFinalizer)
	if err := r.Patch(r.ctx, seed, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
		return fmt.Errorf("failed to add finalizer to Seed: %v", err)
	}

	seedCreators := []reconciling.NamedSeedCreatorGetter{
		seedCreator(seed),
	}
	supportedStrategies := map[corev1.ServiceType]struct{}{
		corev1.ServiceTypeNodePort:     {},
		corev1.ServiceTypeLoadBalancer: {},
	}
	if seed.Spec.ExposeStrategy != "" {
		if _, ok := supportedStrategies[seed.Spec.ExposeStrategy]; !ok {
			return fmt.Errorf("failed to validate seed: invalid Seed Expose Strategy %s", seed.Spec.ExposeStrategy)
		}
	}
	if err := reconciling.ReconcileSeeds(r.ctx, seedCreators, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile seed: %v", err)
	}

	return nil
}

// cleanupDeletedSeed is triggered when a Seed CR inside the master cluster has been deleted
// and is responsible for removing the Seed CR copy inside the seed cluster. This can end up
// in a Retry if other components like the Kubermatic Operator still have finalizers on the
// Seed CR copy.
func (r *Reconciler) cleanupDeletedSeed(seedInMaster *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, logger *zap.SugaredLogger) (*reconcile.Result, error) {
	if !kubernetes.HasAnyFinalizer(seedInMaster, CleanupFinalizer) {
		return nil, nil
	}

	logger.Debug("Seed was deleted, removing copy in seed cluster")

	key, err := ctrlruntimeclient.ObjectKeyFromObject(seedInMaster)
	if err != nil {
		return nil, fmt.Errorf("failed to create object key for Seed CR: %v", err)
	}

	// when master==seed cluster, this is the same as seedInMaster
	seedInSeed := &kubermaticv1.Seed{}

	err = seedClient.Get(r.ctx, key, seedInSeed)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to probe for %s: %v", key, err)
	}

	// if the copy still exists, attempt to delete it unless it has only our own finalizer
	// (we have a master==seed situation) and deleting it again would be futile.
	if err == nil && !kubernetes.HasOnlyFinalizer(seedInSeed, CleanupFinalizer) {
		logger.Debug("Issuing DELETE call for Seed copy now")
		if err := seedClient.Delete(r.ctx, seedInSeed); err != nil {
			return nil, fmt.Errorf("failed to delete %s: %v", key, err)
		}

		return &reconcile.Result{
			// cleanup in remote seed clusters can be slow over long distances
			RequeueAfter: 3 * time.Second,
		}, nil
	}

	// at this point either the Seed CR copy is gone or it has only our own finalizer left
	oldSeed := seedInMaster.DeepCopy()
	kubernetes.RemoveFinalizer(seedInMaster, CleanupFinalizer)

	if err := r.Patch(r.ctx, seedInMaster, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
		return nil, fmt.Errorf("failed to remove finalizer from Seed in master cluster: %v", err)
	}

	return nil, nil
}
