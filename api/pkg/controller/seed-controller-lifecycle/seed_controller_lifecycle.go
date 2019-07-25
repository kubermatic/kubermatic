package seedcontrollerlifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
	enqueuer             *enqueuer
	activeControllers    *controllersInstance
}

// controllersInstance represents an instance of a set of running controllers
type controllersInstance struct {
	log         *zap.SugaredLogger
	configHash  string
	controllers manager.Runnable
	running     bool
	stopChan    chan struct{}
	enqueue     func()
}

// enqueuer implements source.Source and is used to allow controllerInstances
// to trigger a re-enqueue if they stop
type enqueuer struct {
	queue workqueue.RateLimitingInterface
}

func (e *enqueuer) Start(_ handler.EventHandler, queue workqueue.RateLimitingInterface, _ ...predicate.Predicate) error {
	e.queue = queue
	return nil
}

func (e *enqueuer) Enqueue() {
	e.queue.AddRateLimited(queueKey)
}

func (ci *controllersInstance) Start(stopCh <-chan struct{}) {
	ci.log.Info("Starting controllers")
	err := ci.controllers.Start(stopCh)
	if err != nil {
		ci.log.With("error", err).Error("controllers stopped with error")
	} else {
		ci.log.Info("controllers stopped")
	}
	ci.running = false

	// Make sure a reconciliation happens
	ci.enqueue()
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
		enqueuer:             &enqueuer{},
	}
	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: 1})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for _, t := range []runtime.Object{&kubermaticv1.Seed{}, &corev1.Secret{}} {
		if err := c.Watch(&source.Kind{Type: t}, controllerutil.EnqueueConst(queueKey)); err != nil {
			return fmt.Errorf("failed to create watch for type %T: %v", t, err)
		}
	}
	if err := c.Watch(reconciler.enqueuer, controllerutil.EnqueueConst(queueKey)); err != nil {
		return fmt.Errorf("failed to create watch for enqueuer: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	err := r.reconcile()
	if err != nil {
		r.log.With("error", err).Error("reconciliation failed")
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
			return fmt.Errorf("failed to get kubeconfig for seed %q: %v", seedName, err)
		}
		seedKubeconfigMap[seedName] = *cfg
	}

	configHash, err := r.seedKubeconfigHashStable(seedKubeconfigMap)
	if err != nil {
		return fmt.Errorf("failed to calculate hash for seed kubeconfigs: %v", err)
	}
	log := r.log.With("config_hash", configHash)

	if r.activeControllers != nil && r.activeControllers.configHash == configHash {
		if !r.activeControllers.running {
			log.Info("found matching controllers but were not running, starting them")
			r.activeControllers.Start(r.activeControllers.stopChan)
		}
		return nil
	}

	controllers, err := r.controllerFactory()
	if err != nil {
		return fmt.Errorf("failed to construct controllers: %v", err)
	}

	if r.activeControllers != nil && r.activeControllers.running {
		log.Info("Stopping old version of controllers")
		close(r.activeControllers.stopChan)
	}

	r.activeControllers = &controllersInstance{
		log:         log,
		configHash:  configHash,
		controllers: controllers,
		enqueue:     r.enqueuer.Enqueue,
		stopChan:    make(chan struct{}),
	}
	go r.activeControllers.Start(r.activeControllers.stopChan)
	return nil
}

// seedKubeconfigStableHash returns a hash over the seedNames + associated kubeconfigs
// it is used to determine config changes, so it must not rely on ordering of items it gets
func (r *Reconciler) seedKubeconfigHashStable(seedKubeConfigs map[string]rest.Config) (string, error) {

	var sortedNames []string
	for name := range seedKubeConfigs {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	var rawConfigData []byte
	for _, seed := range sortedNames {
		// Function types can not be marshalled, so we have to extract the marshallable parts...
		cfg := seedKubeConfigs[seed]
		identifier := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s/%s/%s",
			seed, cfg.Host, cfg.APIPath, cfg.Username, cfg.Password, cfg.BearerToken, cfg.BearerTokenFile,
			string(cfg.CertData), string(cfg.KeyData), string(cfg.CAData))
		jsonEncodedRestConfig, err := json.Marshal([]byte(identifier))
		if err != nil {
			return "", fmt.Errorf("failed to marshal restconfig: %v", err)
		}
		rawConfigData = append(rawConfigData, jsonEncodedRestConfig...)
	}

	checksumRaw := sha256.Sum256(rawConfigData)
	return string(checksumRaw[:]), nil
}
