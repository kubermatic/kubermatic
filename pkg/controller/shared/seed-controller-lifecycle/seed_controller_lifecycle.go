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

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-seed-lifecycle-controller"

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
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	controllerFactories ...ControllerFactory,
) error {
	// prepare a shared cache across all future master managers; this cache
	// will be wrapped so that ctrlruntime cannot start and stop it, which
	// would cause it to close the same channels multiple times and panic
	cache, err := cache.New(mgr.GetConfig(), cache.Options{
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
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

	sourceChannel := make(chan event.GenericEvent)
	reconciler.enqueue = func() {
		sourceChannel <- event.GenericEvent{
			Object: &kubermaticv1.Seed{},
		}
	}

	bldr := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		WatchesRawSource(source.Channel(sourceChannel, controllerutil.EnqueueConst(queueKey)))

	for _, t := range []ctrlruntimeclient.Object{
		&kubermaticv1.Seed{},
		&corev1.Secret{},
	} {
		bldr.Watches(t, controllerutil.EnqueueConst(queueKey), builder.WithPredicates(predicateutil.ByNamespace(namespace)))
	}

	_, err = bldr.Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {
	err := r.reconcile(ctx)

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context) error {
	seeds, err := r.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %w", err)
	}

	for name, seed := range seeds {
		// Before the seed-init controller in the operator has initialized the seed with our
		// CRDs, there is no point in trying to do anything useful with it, so to prevent
		// controllers from running into errors, we skip uninitialized seeds.
		if !seed.Status.IsInitialized() {
			r.log.Debugw("Seed has not yet been initialized, skipping.", "seed", seed.Name)
			delete(seeds, name)
		}
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
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
		// Avoid duplicating caches or client for master cluster, as it's static.
		NewCache: func(_ *rest.Config, _ cache.Options) (cache.Cache, error) {
			return r.masterCache, nil
		},
		NewClient: func(_ *rest.Config, _ ctrlruntimeclient.Options) (ctrlruntimeclient.Client, error) {
			return r.masterClient, nil
		},
		Controller: ctrlruntimeconfig.Controller{
			// As part of its operation, this controller-manager starts and stops individual controllers
			// during runtime (cf. seedlifecycle controller). Since controller-runtime's unique-name
			// check uses a global singleton with no regard for stopped controllers, we disable the
			// name validation.
			SkipNameValidation: ptr.To(true),
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
			Metrics: metricsserver.Options{BindAddress: "0"},
			Controller: ctrlruntimeconfig.Controller{
				SkipNameValidation: ptr.To(true),
			},
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
