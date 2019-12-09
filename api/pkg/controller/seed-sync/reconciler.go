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
	"k8s.io/apimachinery/pkg/util/wait"
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

	if err := r.reconcile(seed, logger); err != nil {
		r.recorder.Eventf(seed, corev1.EventTypeWarning, "ReconcilingFailed", "%v", err)
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %v", err)
	}

	logger.Info("Successfully reconciled")
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(seed *kubermaticv1.Seed, logger *zap.SugaredLogger) error {
	client, err := r.seedClientGetter(seed)
	if err != nil {
		return fmt.Errorf("failed to create client for seed: %v", err)
	}

	// cleanup once a Seed was deleted in the master cluster
	if seed.DeletionTimestamp != nil {
		return r.cleanupDeletedSeed(seed, client, logger)
	}

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

	if err := reconciling.ReconcileSeeds(r.ctx, seedCreators, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile seed: %v", err)
	}

	return nil
}

func (r *Reconciler) cleanupDeletedSeed(seedInMaster *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	if kubernetes.HasAnyFinalizer(seedInMaster, CleanupFinalizer) {
		logger.Debug("Seed was deleted, removing copy in seed cluster")

		key, err := ctrlruntimeclient.ObjectKeyFromObject(seedInMaster)
		if err != nil {
			return fmt.Errorf("failed to create object key for Seed CR: %v", err)
		}

		// The DELETE op can block indefinitely if the Kubermatic Operator
		// has put a finalizer on the Seed inside the seed cluster and
		// no operator is running at the moment.
		cleanupCtx, cancel := context.WithTimeout(r.ctx, 60*time.Second)
		defer cancel()

		seedInSeed := &kubermaticv1.Seed{}

		err = seedClient.Get(cleanupCtx, key, seedInSeed)
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to probe for %s: %v", key, err)
		}

		// Now we can delete the Seed CR copy inside the seed cluster. This will not block
		// until the operator had time to cleanup, so after this call we must wait for
		// the Seed CR to be actually gone.
		// If seed==master cluster, we must not wait for the deletion, because the Seed
		// "copy" has a finalizer from ourselves and we would therefore wait indefinitely
		// for ourselves.
		if err == nil {
			logger.Debug("Deleting Seed now")
			if err := seedClient.Delete(cleanupCtx, seedInSeed); err != nil {
				return fmt.Errorf("failed to delete %s: %v", key, err)
			}

			if seedInMaster.UID != seedInSeed.UID {
				logger.Debug("Waiting for Seed CR to be actually removed....")

				err := wait.PollImmediate(1*time.Second, 30*time.Second, func() (done bool, err error) {
					tmpSeed := &kubermaticv1.Seed{}
					err = seedClient.Get(cleanupCtx, key, tmpSeed)
					if err == nil {
						return false, nil
					}

					if kerrors.IsNotFound(err) {
						return true, nil
					}

					return false, fmt.Errorf("failed to probe for %s: %v", key, err)
				})
				if err != nil {
					return fmt.Errorf("failed to wait for Seed to be removed in seed cluster: %v", err)
				}

				logger.Debug("Seed CR has been removed.")
			}
		}

		oldSeed := seedInMaster.DeepCopy()
		kubernetes.RemoveFinalizer(seedInMaster, CleanupFinalizer)

		if err := r.Patch(r.ctx, seedInMaster, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
			return fmt.Errorf("failed to remove finalizer from Seed in master cluster: %v", err)
		}

		return nil
	}

	return nil
}
