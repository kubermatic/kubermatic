package seedsync

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/pkg/controller/util/predicate"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "seed-sync-controller"

	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// CleanupFinalizer is put on Seed CRs to facilitate proper
	// cleanup when a Seed is deleted.
	CleanupFinalizer = "kubermatic.io/cleanup-seed-sync"
)

// Add creates a new Seed-Sync controller and sets up Watches
func Add(
	ctx context.Context,
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	namespace string,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
) error {
	reconciler := &Reconciler{
		Client:           mgr.GetClient(),
		ctx:              ctx,
		recorder:         mgr.GetEventRecorderFor(ControllerName),
		log:              log.Named(ControllerName),
		seedClientGetter: provider.SeedClientGetterFactory(seedKubeconfigGetter),
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	// watch all seeds in the given namespace
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Seed{}}, &handler.EnqueueRequestForObject{}, predicate.ByNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to create watcher: %v", err)
	}

	return nil
}
