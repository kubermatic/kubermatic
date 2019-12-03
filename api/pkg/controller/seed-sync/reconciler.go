package seedsync

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
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
	if err := common.PatchSeed(r.Client, seed, func(s *kubermaticv1.Seed) error {
		kubernetes.AddFinalizer(s, common.CleanupFinalizer)
		return nil
	}); err != nil {
		return err
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
	if sets.NewString(seedInMaster.Finalizers...).Has(CleanupFinalizer) {
		logger.Debug("Seed was deleted, removing copy in seed cluster")

		key, err := ctrlruntimeclient.ObjectKeyFromObject(seedInMaster)
		if err != nil {
			return fmt.Errorf("failed to create object key for Seed CR: %v", err)
		}

		// The DELETE op can block indefinitely if the Kubermatic Operator
		// has put a finalizer on the Seed inside the seed cluster and
		// no operator is running at the moment.
		cleanupCtx, cancel := context.WithTimeout(r.ctx, 30*time.Second)
		defer cancel()

		seedInSeed := &kubermaticv1.Seed{}

		err = seedClient.Get(cleanupCtx, key, seedInSeed)
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to probe for %s: %v", key, err)
		}

		// In cases where master and seed cluster are inside the same Kubernetes
		// cluster, the Seed CR's copy is not actually a copy, but the same resource.
		// This means that the "copy" has our CleanupFinalizer attached as well.
		// Attempting to delete seedInMaster would therefore be blocked by
		// our own finalizer.
		// For this reason we check if master==seed and if so, only remove the
		// finalizer later on.
		if err == nil {
			if seedInMaster.UID == seedInSeed.UID {
				logger.Debug("Seed CR is identical to master version, not performing additional DELETE operation", "uid", seedInSeed.UID)
			} else if err := seedClient.Delete(cleanupCtx, seedInSeed); err != nil {
				return fmt.Errorf("failed to delete %s: %v", key, err)
			}
		}

		return common.PatchSeed(r.Client, seedInMaster, func(s *kubermaticv1.Seed) error {
			kubernetes.RemoveFinalizer(s, CleanupFinalizer)
			return nil
		})
	}

	return nil
}
