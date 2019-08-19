package seedsync

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler copies seed CRs into their respective clusters,
// assuming that Kubermatic and the seed CRD have already been
// installed.
type Reconciler struct {
	ctrlruntimeclient.Client

	seedKubeconfigGetter provider.SeedKubeconfigGetter
	log                  *zap.SugaredLogger
	ctx                  context.Context
	recorder             record.EventRecorder
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := r.log.With("seed", request.Name)
	logger.Info("Reconciling seed")

	seed := &kubermaticv1.Seed{}
	if err := r.Get(r.ctx, request.NamespacedName, seed); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get seed: %v", err)
	}

	if err := r.reconcile(seed, logger); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %v", err)
	}

	logger.Info("Successfully reconciled")
	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(seed *kubermaticv1.Seed, logger *zap.SugaredLogger) error {
	kubeconfig, err := r.seedKubeconfigGetter(seed)
	if err != nil {
		return fmt.Errorf("failed to retrieve kubeconfig: %v", err)
	}

	client, err := ctrlruntimeclient.New(kubeconfig, ctrlruntimeclient.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client for seed: %v", err)
	}

	seedCreators := []reconciling.NamedSeedCreatorGetter{
		seedCreator(seed),
	}

	if err := reconciling.ReconcileSeeds(r.ctx, seedCreators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile seed: %v", err)
	}

	return nil
}
