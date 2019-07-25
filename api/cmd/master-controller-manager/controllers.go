package main

import (
	"fmt"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	seedcontrollerlifecycle "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-lifecycle"
	seedproxy "github.com/kubermatic/kubermatic/api/pkg/controller/seed-proxy"
	"github.com/kubermatic/kubermatic/api/pkg/controller/service-account"
	"github.com/kubermatic/kubermatic/api/pkg/controller/user-project-binding"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/prometheus/client_golang/prometheus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if err := seedcontrollerlifecycle.Add(ctrlCtx.ctx,
		kubermaticlog.Logger,
		ctrlCtx.mgr,
		ctrlCtx.seedsGetter,
		ctrlCtx.seedKubeconfigGetter,
		rbacControllerFactory); err != nil {
		//TODO: Find a better name
		return fmt.Errorf("failed to create seedcontrollerlifecycle: %v", err)
	}
	if err := userprojectbinding.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create userprojectbinding controller: %v", err)
	}
	if err := serviceaccount.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create serviceaccount controller: %v", err)
	}
	if err := seedproxy.Add(ctrlCtx.mgr, 1, ctrlCtx.seedsGetter, ctrlCtx.seedKubeconfigGetter); err != nil {
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
) func() (manager.Runnable, error) {

	rbacMetrics := rbac.NewMetrics()
	prometheus.MustRegister(rbacMetrics.Workers)

	return func() (manager.Runnable, error) {
		seeds, err := seedsGetter()
		if err != nil {
			return nil, fmt.Errorf("failed to get seeds: %v", err)
		}
		masterClusterProvider, err := rbacClusterProvider(mastercfg, "master", true, selectorOps)
		if err != nil {
			return nil, fmt.Errorf("failed to create master rbac provider: %v", err)
		}
		allClusterProviders := []*rbac.ClusterProvider{masterClusterProvider}

		for seedName := range seeds {
			kubeConfig, err := seedKubeconfigGetter(seedName)
			if err != nil {
				kubermaticlog.Logger.With("error", err).With("seed", seedName).Error("error getting kubeconfig")
				// Dont let a single broken kubeconfig break the whole controller creation
				continue
			}
			clusterProvider, err := rbacClusterProvider(kubeConfig, seedName, false, selectorOps)
			if err != nil {
				return nil, fmt.Errorf("failed to create rbac provider for seed %q: %v", seedName, err)
			}
			allClusterProviders = append(allClusterProviders, clusterProvider)
		}

		ctrl, err := rbac.New(rbacMetrics, allClusterProviders, workerCount)
		if err != nil {
			return nil, fmt.Errorf("failed to create rbac controller: %v", err)
		}

		// This is an implementation of sigs.k8s.io/controller-runtime/pkg/manager.Runnable
		// It wraps the actual controllers implementation to make sure informers are started first
		runnableFunc := func(stopCh <-chan struct{}) error {
			for _, clusterProvider := range allClusterProviders {
				clusterProvider.StartInformers(stopCh)
				if err := clusterProvider.WaitForCachesToSync(stopCh); err != nil {
					return fmt.Errorf("RBAC controller failed to sync cache: %v", err)
				}
			}
			return ctrl.Start(stopCh)
		}
		return manager.RunnableFunc(runnableFunc), nil
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
