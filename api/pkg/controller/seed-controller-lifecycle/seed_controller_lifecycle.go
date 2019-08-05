package seedcontrollerlifecycle

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrlruntimecontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "seedcontroller_lifecycle_manager"

	// We must only enqueue this one key
	queueKey = ControllerName
)

type Reconciler struct {
	ctx                  context.Context
	log                  *zap.SugaredLogger
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	controllerFactory    func() (manager.Runnable, error)
	enqueue              func()
	activeController     controllerInstance
}

// controllerInstance represents an instance of a running
// controller
type controllerInstance struct {
	config     map[string]rest.Config
	controller manager.Runnable
	running    bool
	stopChan   chan struct{}
}

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	controllerFactory func() (manager.Runnable, error),
) error {

	reconciler := &Reconciler{
		ctx:                  ctx,
		log:                  log.Named(ControllerName),
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		controllerFactory:    controllerFactory,
	}
	c, err := ctrlruntimecontroller.New(ControllerName, mgr,
		ctrlruntimecontroller.Options{Reconciler: reconciler, MaxConcurrentReconciles: 1})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for _, t := range []runtime.Object{&kubermaticv1.Seed{}, &corev1.Secret{}} {
		if err := c.Watch(&source.Kind{Type: t}, controllerutil.EnqueueConst(queueKey)); err != nil {
			return fmt.Errorf("failed to create watch for type %T: %v", t, err)
		}
	}

	sourceChannel := make(chan event.GenericEvent)
	reconciler.enqueue = func() {
		sourceChannel <- event.GenericEvent{
			// TODO: Is it needed to fill this?
			Object: &kubermaticv1.Seed{},
		}
	}
	if err := c.Watch(&source.Channel{Source: sourceChannel}, controllerutil.EnqueueConst(queueKey)); err != nil {
		return fmt.Errorf("failed to create watch for channelSource: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	err := r.reconcile()
	if err != nil {
		r.log.Errorw("reconiliation failed", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile() error {
	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %v", err)
	}

	seedKubeconfigMap := map[string]rest.Config{}
	for seedName := range seeds {
		cfg, err := r.seedKubeconfigGetter(seedName)
		if err != nil {
			// Don't let a single broken kubeconfig break everything.
			r.log.Errorw("failed to get kubeconfig", "seed", seedName, zap.Error(err))
			continue
		}
		seedKubeconfigMap[seedName] = *cfg
	}

	if r.activeController.running && reflect.DeepEqual(r.activeController.config, seedKubeconfigMap) {
		r.log.Debug("found running controller instance with up-to-date config, nothing to do")
		return nil
	}

	controller, err := r.controllerFactory()
	if err != nil {
		return fmt.Errorf("failed to construct controllers: %v", err)
	}

	if r.activeController.running {
		r.log.Info("Stopping old version of controllers")
		close(r.activeController.stopChan)
	}

	r.activeController = controllerInstance{
		config:     seedKubeconfigMap,
		controller: controller,
		stopChan:   make(chan struct{}),
	}
	go func() {
		r.log.Info("starting controllers")
		r.activeController.running = true
		if err := r.activeController.controller.Start(r.activeController.stopChan); err != nil {
			r.log.Errorw("controllers stopped with error", zap.Error(err))
		}
		r.log.Info("controllers stopped")
		// Make sure we check on this
		r.activeController.running = false
		r.enqueue()
	}()

	return nil
}
