package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	projectlabelsynchronizer "github.com/kubermatic/kubermatic/api/pkg/controller/project-label-synchronizer"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	seedcontrollerlifecycle "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-lifecycle"
	seedproxy "github.com/kubermatic/kubermatic/api/pkg/controller/seed-proxy"
	seedsync "github.com/kubermatic/kubermatic/api/pkg/controller/seed-sync"
	serviceaccount "github.com/kubermatic/kubermatic/api/pkg/controller/service-account"
	userprojectbinding "github.com/kubermatic/kubermatic/api/pkg/controller/user-project-binding"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usersshkeyssynchronizer"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/prometheus/client_golang/prometheus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func createAllControllers(ctrlCtx *controllerContext) error {
	rbacControllerFactory := rbacControllerFactoryCreator(
		ctrlCtx.mgr.GetConfig(),
		ctrlCtx.seedsGetter,
		ctrlCtx.seedKubeconfigGetter,
		ctrlCtx.workerCount,
		ctrlCtx.labelSelectorFunc)
	projectLabelSynchronizerFactory := projectLabelSynchronizerFactoryCreator(ctrlCtx)
	userSSHKeysSynchronizerFactory := userSSHKeysSynchronizerFactoryCreator(ctrlCtx)

	if err := seedcontrollerlifecycle.Add(ctrlCtx.ctx,
		kubermaticlog.Logger,
		ctrlCtx.mgr,
		ctrlCtx.namespace,
		ctrlCtx.seedsGetter,
		ctrlCtx.seedKubeconfigGetter,
		rbacControllerFactory,
		projectLabelSynchronizerFactory,
		userSSHKeysSynchronizerFactory); err != nil {
		//TODO: Find a better name
		return fmt.Errorf("failed to create seedcontrollerlifecycle: %v", err)
	}
	if err := userprojectbinding.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create userprojectbinding controller: %v", err)
	}
	if err := serviceaccount.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create serviceaccount controller: %v", err)
	}
	if err := seedsync.Add(ctrlCtx.ctx, ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create seedsync controller: %v", err)
	}
	if err := seedproxy.Add(ctrlCtx.ctx, ctrlCtx.mgr, 1, ctrlCtx.log, ctrlCtx.namespace, ctrlCtx.seedsGetter, ctrlCtx.seedKubeconfigGetter); err != nil {
		return fmt.Errorf("failed to create seedproxy controller: %v", err)
	}
	return nil
}

func rbacControllerFactoryCreator(
	mastercfg *rest.Config,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	workerCount int,
	selectorOps func(*metav1.ListOptions),
) seedcontrollerlifecycle.ControllerFactory {

	rbacMetrics := rbac.NewMetrics()
	seedKubeconfigRetrievalSuccessMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "kubermatic",
		Subsystem: "master_controller_manager",
		Name:      "seed_kubeconfig_retrieval_success",
		Help:      "Indicates if retrieving the kubeconfig for the given seed was successful",
	}, []string{"seed"})
	// GaugeVec doesn't have a func to only keep metrics with a given label value, it only
	// has a func to delete metrics with a given label value. Hence we have to track ourselves
	// which label values get removed
	seedsWithMetrics := sets.NewString()
	prometheus.MustRegister(rbacMetrics.Workers, seedKubeconfigRetrievalSuccessMetric)

	factory := func(mgr manager.Manager) error {
		seeds, err := seedsGetter()
		if err != nil {
			return fmt.Errorf("failed to get seeds: %v", err)
		}
		masterClusterProvider, err := rbacClusterProvider(mastercfg, "master", true, selectorOps)
		if err != nil {
			return fmt.Errorf("failed to create master rbac provider: %v", err)
		}
		allClusterProviders := []*rbac.ClusterProvider{masterClusterProvider}

		newSeedsWithMetrics := sets.NewString()
		for _, seed := range seeds {
			kubeConfig, err := seedKubeconfigGetter(seed)
			if err != nil {
				kubermaticlog.Logger.With("error", err).With("seed", seed.Name).Error("error getting kubeconfig")
				// Dont let a single broken kubeconfig break the whole controller creation
				seedKubeconfigRetrievalSuccessMetric.WithLabelValues(seed.Name).Set(0)
				continue
			}
			seedKubeconfigRetrievalSuccessMetric.WithLabelValues(seed.Name).Set(1)
			clusterProvider, err := rbacClusterProvider(kubeConfig, seed.Name, false, selectorOps)
			if err != nil {
				return fmt.Errorf("failed to create rbac provider for seed %q: %v", seed.Name, err)
			}
			allClusterProviders = append(allClusterProviders, clusterProvider)
		}
		removedSeeds := sets.NewString(seedsWithMetrics.List()...)
		removedSeeds.Delete(newSeedsWithMetrics.List()...)
		_ = seedKubeconfigRetrievalSuccessMetric.DeleteLabelValues(removedSeeds.List()...)
		seedsWithMetrics = newSeedsWithMetrics

		ctrl, err := rbac.New(rbacMetrics, allClusterProviders, workerCount)
		if err != nil {
			return fmt.Errorf("failed to create rbac controller: %v", err)
		}

		return mgr.Add(manager.RunnableFunc(func(stopCh <-chan struct{}) error {
			// This is an implementation of sigs.k8s.io/controller-runtime/pkg/manager.Runnable
			// It wraps the actual controllers implementation to make sure informers are started first
			for _, clusterProvider := range allClusterProviders {
				clusterProvider.StartInformers(stopCh)
				if err := clusterProvider.WaitForCachesToSync(stopCh); err != nil {
					return fmt.Errorf("RBAC controller failed to sync cache: %v", err)
				}
			}
			return ctrl.Start(stopCh)
		}))
	}
	return func(mgr manager.Manager) (string, error) {
		return "rbac-controller", factory(mgr)
	}
}

func rbacClusterProvider(cfg *rest.Config, name string, master bool, labelSelectorFunc func(*metav1.ListOptions)) (*rbac.ClusterProvider, error) {
	clusterPrefix := rbac.SeedProviderPrefix
	if master {
		clusterPrefix = rbac.MasterProviderPrefix
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeClient: %v", err)
	}
	kubermaticClient, err := kubermaticclientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubermaticClient: %v", err)
	}
	kubermaticInformerFactory := externalversions.NewFilteredSharedInformerFactory(kubermaticClient, time.Minute*5, metav1.NamespaceAll, labelSelectorFunc)
	kubeInformerProvider := rbac.NewInformerProvider(kubeClient, time.Minute*5)

	return rbac.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, name), kubeClient, kubeInformerProvider, kubermaticClient, kubermaticInformerFactory), nil
}

func projectLabelSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	factory := func(mgr manager.Manager) error {
		seedManagerMap, err := createSeedManagers("project-label-synchronizer-factory", ctrlCtx, mgr)
		if err != nil {
			return fmt.Errorf("failed build seeds managers: %v", err)
		}

		return projectlabelsynchronizer.Add(
			ctrlCtx.ctx,
			mgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerCount,
			ctrlCtx.workerName)
	}
	return func(mgr manager.Manager) (string, error) {
		return projectlabelsynchronizer.ControllerName, factory(mgr)
	}
}

func userSSHKeysSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	factory := func(mgr manager.Manager) error {
		seedManagerMap, err := createSeedManagers("usersshkeys-synchronizer-factory", ctrlCtx, mgr)
		if err != nil {
			return fmt.Errorf("failed build seeds managers: %v", err)
		}

		return usersshkeyssynchronizer.Add(
			ctrlCtx.ctx,
			mgr,
			seedManagerMap,
			ctrlCtx.log,
			ctrlCtx.workerName,
			ctrlCtx.workerCount,
		)
	}
	return func(mgr manager.Manager) (string, error) {
		return usersshkeyssynchronizer.ControllerName, factory(mgr)
	}
}

func createSeedManagers(name string, ctrlCtx *controllerContext, mgr manager.Manager) (map[string]manager.Manager, error) {
	log := ctrlCtx.log.Named(name)
	seeds, err := ctrlCtx.seedsGetter()
	if err != nil {
		log.Errorw("Failed to get seeds", zap.Error(err))
		return nil, fmt.Errorf("failed to get seeds: %v", err)
	}

	seedManagerMap := make(map[string]manager.Manager, len(seeds))
	for seedName, seed := range seeds {
		log := ctrlCtx.log.With("seed", seed.Name)
		kubeconfig, err := ctrlCtx.seedKubeconfigGetter(seed)
		if err != nil {
			log.Errorw("Failed to get kubeconfig for seed", zap.Error(err))
			// Don't let one defunct seed break everything. We have a metric for this
			// in the rbac controller factory, so just log it here
			continue
		}
		seedMgr, err := manager.New(kubeconfig, manager.Options{
			MetricsBindAddress: "0",
		})
		if err != nil {
			log.Errorw("Failed to construct mgr for seed", zap.Error(err))
			continue
		}
		seedManagerMap[seedName] = seedMgr
		if err := mgr.Add(seedMgr); err != nil {
			return nil, fmt.Errorf("failed to add controller manager for seed %q to mgr: %v", seedName, err)
		}
	}

	return seedManagerMap, nil
}
