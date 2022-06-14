/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package seedcontrollerlifecycle

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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
	ControllerName = "kp-seed-lifecycle-controller"

	// We must only enqueue this one key.
	queueKey = ControllerName
)

var (
	seedKubeconfigRetrievalSuccessMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "kubermatic",
		Subsystem: "master_controller_manager",
		Name:      "seed_kubeconfig_retrieval_success",
		Help:      "Indicates if retrieving the kubeconfig for the given seed was successful",
	}, []string{"seed"})
)

func init() {
	prometheus.MustRegister(seedKubeconfigRetrievalSuccessMetric)
}

type Reconciler struct {
	masterKubeCfg        *rest.Config
	masterClient         ctrlruntimeclient.Client
	masterCache          cache.Cache
	log                  *zap.SugaredLogger
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	controllerFactories  []ControllerFactory
	enqueue              func()
	activeManager        *managerInstance
	namespace            string
}

// managerInstance represents an instance of controllerManager.
type managerInstance struct {
	config    map[string]rest.Config
	mgr       manager.Manager
	running   bool
	cancelCtx context.CancelFunc
}

// ControllerFactory is a function to create a new controller instance.
type ControllerFactory func(context.Context, manager.Manager, map[string]manager.Manager) (controllerName string, err error)

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	namespace string,
	restrictedManager bool,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	controllerFactories ...ControllerFactory,
) error {
	// prepare a shared cache across all future master managers; this cache
	// will be wrapped so that ctrlruntime cannot start and stop it, which
	// would cause it to close the same channels multiple times and panic
	resync := 2 * time.Second
	cache, err := cache.New(mgr.GetConfig(), cache.Options{
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
		Resync: &resync,
	})
	if err != nil {
		return err
	}

	go func() {
		if err := cache.Start(ctx); err != nil {
			log.Fatalw("failed to start cache", zap.Error(err))
		}
	}()

	if !cache.WaitForCacheSync(ctx) {
		log.Fatal("failed to wait for caches to synchronize")
	}

	reconciler := &Reconciler{
		masterKubeCfg:        mgr.GetConfig(),
		masterClient:         mgr.GetClient(),
		masterCache:          &unstartableCache{cache},
		log:                  log.Named(ControllerName),
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		controllerFactories:  controllerFactories,
	}
	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: 1})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	// Only the KKP operator wants to actually restrict the temporary
	// managers to the given namespace, normally the namespace is only
	// used to filter resources; since restricting the manager could
	// result in chaos for the seed-controller-manager, the restriction
	// is optional.
	if restrictedManager {
		reconciler.namespace = namespace
	}

	for _, t := range []ctrlruntimeclient.Object{&kubermaticv1.Seed{}, &corev1.Secret{}} {
		if err := c.Watch(
			&source.Kind{Type: t},
			controllerutil.EnqueueConst(queueKey),
			predicateutil.ByNamespace(namespace),
		); err != nil {
			return fmt.Errorf("failed to create watch for type %T: %w", t, err)
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
		return fmt.Errorf("failed to create watch for channelSource: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {
	err := r.reconcile(ctx)
	if err != nil {
		r.log.Errorw("reconiliation failed", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context) error {
	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %w", err)
	}

	seedKubeconfigMap := map[string]rest.Config{}
	for _, seed := range seeds {
		cfg, err := r.seedKubeconfigGetter(seed)
		if err != nil {
			// Don't let a single broken kubeconfig break everything, just update the Seed
			// status and continue with the other seeds.
			r.log.Errorw("failed to get kubeconfig", "seed", seed.Name, zap.Error(err))
			seedKubeconfigRetrievalSuccessMetric.WithLabelValues(seed.Name).Set(0)
			continue
		}

		seedKubeconfigMap[seed.Name] = *cfg
		seedKubeconfigRetrievalSuccessMetric.WithLabelValues(seed.Name).Set(1)
	}

	// delete label combinations for non-existing seeds
	if r.activeManager != nil {
		removedSeeds := sets.StringKeySet(r.activeManager.config).Difference(sets.StringKeySet(seedKubeconfigMap))
		for _, seedName := range removedSeeds.List() {
			seedKubeconfigRetrievalSuccessMetric.DeleteLabelValues(seedName)
		}
	}

	if r.activeManager != nil && r.activeManager.running && reflect.DeepEqual(r.activeManager.config, seedKubeconfigMap) {
		r.log.Debug("All controllers running and config up-to-date, nothing to do.")
		return nil
	}

	// We let a master controller manager run the controllers for us.
	mgr, err := manager.New(r.masterKubeCfg, manager.Options{
		Namespace:          r.namespace, // can be empty if the manager should not be restricted
		LeaderElection:     false,
		MetricsBindAddress: "0",
		// Avoid duplicating caches or client for master cluster, as it's static.
		NewCache: func(_ *rest.Config, _ cache.Options) (cache.Cache, error) {
			return r.masterCache, nil
		},
		NewClient: func(_ cache.Cache, _ *rest.Config, _ ctrlruntimeclient.Options, _ ...ctrlruntimeclient.Object) (ctrlruntimeclient.Client, error) {
			return r.masterClient, nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create master controller manager: %w", err)
	}

	// create one manager per seed, so that all controllers share the same caches
	seedManagers, err := r.createSeedManagers(mgr, seeds, seedKubeconfigMap)
	if err != nil {
		return fmt.Errorf("failed to create managers for all seeds: %w", err)
	}

	// create a new, independent context, as the one given to reconcile() can possibly
	// be cancelled once the reconciliation is done, and we want our context to live
	// on for a long time
	ctrlCtx, cancelCtrlCtx := context.WithCancel(context.Background())

	for _, factory := range r.controllerFactories {
		controllerName, err := factory(ctrlCtx, mgr, seedManagers)
		if err != nil {
			cancelCtrlCtx()
			return fmt.Errorf("failed to construct controller %s: %w", controllerName, err)
		}
	}

	if r.activeManager != nil {
		r.log.Info("Stopping old instance of controller manager")
		r.activeManager.cancelCtx()
	}

	mi := &managerInstance{
		config:    seedKubeconfigMap,
		mgr:       mgr,
		cancelCtx: cancelCtrlCtx,
	}

	go func() {
		r.log.Info("Starting controller manager")
		mi.running = true
		if err := mi.mgr.Start(ctrlCtx); err != nil {
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

func (r *Reconciler) createSeedManagers(masterMgr manager.Manager, seeds map[string]*kubermaticv1.Seed, kubeconfigs map[string]rest.Config) (map[string]manager.Manager, error) {
	seedManagers := make(map[string]manager.Manager, len(seeds))

	for seedName, seed := range seeds {
		kubeconfig, exists := kubeconfigs[seedName]
		if !exists {
			continue // we already warned earlier about the inability to retrieve the kubeconfig
		}

		log := r.log.With("seed", seed.Name)

		seedMgr, err := manager.New(&kubeconfig, manager.Options{
			Namespace:          r.namespace, // can be empty if the manager should not be restricted
			MetricsBindAddress: "0",
		})
		if err != nil {
			log.Errorw("Failed to construct manager for seed", zap.Error(err))
			continue
		}

		seedManagers[seedName] = seedMgr
		if err := masterMgr.Add(seedMgr); err != nil {
			return nil, fmt.Errorf("failed to add controller manager for seed %q to master manager: %w", seedName, err)
		}
	}

	return seedManagers, nil
}

// unstartableCache is used to prevent the ctrlruntime manager from starting the
// cache *again*, just after we started and initialized it.
type unstartableCache struct {
	cache.Cache
}

func (m *unstartableCache) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (m *unstartableCache) WaitForCacheSync(ctx context.Context) bool {
	return true
}
