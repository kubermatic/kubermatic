package operatormaster

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kubermatic-master-operator"

	// WorkerNameLabel is the label containing the worker-name,
	// restricting the operator that is willing to work on a given
	// resource.
	WorkerNameLabel = "operator.kubermatic.io/worker"
)

func Add(
	ctx context.Context,
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	workerName string,
) error {
	reconciler := &Reconciler{
		Client:     mgr.GetClient(),
		recorder:   mgr.GetRecorder(ControllerName),
		log:        log.Named(ControllerName),
		workerName: workerName,
		ctx:        ctx,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	obj := &operatorv1alpha1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: obj}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", obj, err)
	}

	return nil
}
