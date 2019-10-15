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
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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
	masterKubeCfg        *rest.Config
	masterClient         ctrlruntimeclient.Client
	masterCache          cache.Cache
	log                  *zap.SugaredLogger
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	controllerFactories  []ControllerFactory
	enqueue              func()
	activeManager        *managerInstance
}

// managerInstance represents an instance of controllerManager
type managerInstance struct {
	config   map[string]rest.Config
	mgr      manager.Manager
	running  bool
	stopChan chan struct{}
}

// ControllerFactory is a function to create a new controller instance
type ControllerFactory func(manager.Manager) (controllerName string, err error)

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	namespace string,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	controllerFactories ...ControllerFactory,
) error {

	reconciler := &Reconciler{
		ctx:                  ctx,
		masterKubeCfg:        mgr.GetConfig(),
		masterClient:         mgr.GetClient(),
		masterCache:          mgr.GetCache(),
		log:                  log.Named(ControllerName),
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		controllerFactories:  controllerFactories,
	}
	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: 1})
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

	if r.activeManager != nil && r.activeManager.running && reflect.DeepEqual(r.activeManager.config, seedKubeconfigMap) {
		r.log.Debug("All controllers running and config up-to-date, nothing to do.")
		return nil
	}

	// We let a controller manager run the controllers for us.
	mgr, err := manager.New(r.masterKubeCfg, manager.Options{
		LeaderElection:     false,
		MetricsBindAddress: "0",
		// Avoid duplicating caches or client for master cluster, as its static.
		NewCache: func(_ *rest.Config, _ cache.Options) (cache.Cache, error) {
			return r.masterCache, nil
		},
		NewClient: func(_ cache.Cache, _ *rest.Config, _ ctrlruntimeclient.Options) (ctrlruntimeclient.Client, error) {
			return r.masterClient, nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create controller manager: %v", err)
	}

	for _, factory := range r.controllerFactories {
		controllerName, err := factory(mgr)
		if err != nil {
			return fmt.Errorf("failed to construct controller %s: %v", controllerName, err)
		}
	}

	if r.activeManager != nil {
		r.log.Info("Stopping old instance of controller manager")
		close(r.activeManager.stopChan)
	}

	mi := &managerInstance{
		config:   seedKubeconfigMap,
		mgr:      mgr,
		stopChan: make(chan struct{}),
	}
	go func() {
		r.log.Info("Starting controller manager")
		mi.running = true
		if err := mi.mgr.Start(mi.stopChan); err != nil {
			r.log.Errorw("Controller manager stopped with error", zap.Error(err))
		}
		mi.running = false
		r.log.Info("Controller manager stopped")
		// Make sure we check on this
		r.enqueue()
	}()

	r.activeManager = mi
	return nil
}
