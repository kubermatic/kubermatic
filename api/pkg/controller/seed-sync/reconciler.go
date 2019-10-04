package seedsync

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
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
			logger.Warn("Seed has been deleted, skipping reconciling")
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

	seedCreators := []reconciling.NamedSeedCreatorGetter{
		seedCreator(seed),
	}

	if err := reconciling.ReconcileSeeds(r.ctx, seedCreators, seed.Namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile seed: %v", err)
	}

	return nil
}
