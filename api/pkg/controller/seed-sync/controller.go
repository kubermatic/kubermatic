package seedsync

import (
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/controller/util"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "seed-sync-controller"
)

// Add creates a new Seed-Sync controller and sets up Watches
func Add(
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	namespace string,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
) error {
	reconciler := &Reconciler{
		Client:               mgr.GetClient(),
		recorder:             mgr.GetRecorder(ControllerName),
		log:                  log.Named(ControllerName),
		seedKubeconfigGetter: seedKubeconfigGetter,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	// watch all seeds in the given namespace
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Seed{}}, &handler.EnqueueRequestForObject{}, util.NamespacePredicate(namespace)); err != nil {
		return fmt.Errorf("failed to create watcher: %v", err)
	}

	return nil
}
