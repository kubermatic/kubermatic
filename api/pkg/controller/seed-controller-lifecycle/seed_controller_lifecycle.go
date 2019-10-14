package seedcontrollerlifecycle

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
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
	controllerFactories  []func() (manager.Runnable, error)
	enqueue              func()
	activeControllers    controllerInstanceSet
}

// controllerInstanceSet represents a set of running
// controllers.
type controllerInstanceSet struct {
	config      map[string]rest.Config
	controllers []*controllerInstance
	stopChan    chan struct{}
}

// controllerInstance represents an instance of a running
// controller
type controllerInstance struct {
	controller manager.Runnable
	running    bool
}

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	namespace string,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	controllerFactories ...func() (manager.Runnable, error),
) error {

	reconciler := &Reconciler{
		ctx:                  ctx,
		log:                  log.Named(ControllerName),
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		controllerFactories:  controllerFactories,
	}
	c, err := ctrlruntimecontroller.New(ControllerName, mgr,
		ctrlruntimecontroller.Options{Reconciler: reconciler, MaxConcurrentReconciles: 1})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for _, t := range []runtime.Object{&kubermaticv1.Seed{}, &corev1.Secret{}} {
		if err := c.Watch(
			&source.Kind{Type: t},
			controllerutil.EnqueueConst(queueKey),
			predicateutil.ByNamespace(namespace),
		); err != nil {
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
	for _, seed := range seeds {
		cfg, err := r.seedKubeconfigGetter(seed)
		if err != nil {
			// Don't let a single broken kubeconfig break everything.
			r.log.Errorw("failed to get kubeconfig", "seed", seed.Name, zap.Error(err))
			continue
		}
		seedKubeconfigMap[seed.Name] = *cfg
	}

	var allControllersRunning bool
	for _, controller := range r.activeControllers.controllers {
		if !controller.running {
			allControllersRunning = false
			break
		}
	}

	if allControllersRunning && reflect.DeepEqual(r.activeControllers.config, seedKubeconfigMap) {
		r.log.Debug("All controllers running and config up-to-date, nothing to do.")
		return nil
	}

	var controllers []manager.Runnable
	for _, factory := range r.controllerFactories {
		runnable, err := factory()
		if err != nil {
			return fmt.Errorf("failed to construct controller: %v", err)
		}
		controllers = append(controllers, runnable)
	}

	r.log.Info("Stopping old version of controllers")
	close(r.activeControllers.stopChan)

	r.activeControllers = controllerInstanceSet{
		config:   seedKubeconfigMap,
		stopChan: make(chan struct{}),
	}

	for idx := range controllers {
		controller := controllers[idx]
		ci := &controllerInstance{
			controller: controller,
		}
		r.activeControllers.controllers = append(r.activeControllers.controllers, ci)
		go func() {
			r.log.Info("Starting controller")
			ci.running = true
			if err := ci.controller.Start(r.activeControllers.stopChan); err != nil {
				r.log.Errorw("Controller stopped with error", zap.Error(err))
			}
			r.log.Info("Controller stopped")
			ci.running = false
			// Make sure we check on this
			r.enqueue()
		}()
	}
	return nil
}
