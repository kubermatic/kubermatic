package main

import (
	"fmt"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	seedproxy "github.com/kubermatic/kubermatic/api/pkg/controller/seed-proxy"
	"github.com/kubermatic/kubermatic/api/pkg/controller/service-account"
	"github.com/kubermatic/kubermatic/api/pkg/controller/user-project-binding"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func createAllControllers(ctrlCtx *controllerContext) error {
	if err := createRBACContoller(ctrlCtx); err != nil {
		return fmt.Errorf("failed to create rbac controller: %v", err)
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

func createRBACContoller(ctrlCtx *controllerContext) error {
	masterClusterProvider, err := rbacClusterProvider(ctrlCtx.mgr.GetConfig(), "master", true, ctrlCtx.labelSelectorFunc)
	if err != nil {
		return fmt.Errorf("failed to create master rbac provider: %v", err)
	}
	allClusterProviders := []*rbac.ClusterProvider{masterClusterProvider}

	seeds, err := ctrlCtx.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seeds: %v", err)
	}
	for seedName := range seeds {
		kubeConfig, err := ctrlCtx.seedKubeconfigGetter(seedName)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig for seed %q: %v", seedName, err)
		}
		clusterProvider, err := rbacClusterProvider(kubeConfig, seedName, false, ctrlCtx.labelSelectorFunc)
		if err != nil {
			return fmt.Errorf("failed to create rbac provider for seed %q: %v", seedName, err)
		}
		allClusterProviders = append(allClusterProviders, clusterProvider)
	}

	ctrl, err := rbac.New(rbac.NewMetrics(), allClusterProviders, ctrlCtx.workerCount)
	if err != nil {
		return err
	}

	// This is an implementation of
	// sigs.k8s.io/controller-runtime/pkg/manager.Runnable
	// It wraps the actual controllers implementation to defer the informer start
	runnableFunc := func(stopCh <-chan struct{}) error {
		for _, clusterProvider := range allClusterProviders {
			clusterProvider.StartInformers(stopCh)
			if err := clusterProvider.WaitForCachesToSync(stopCh); err != nil {
				return fmt.Errorf("RBAC controller failed to sync cache: %v", err)
			}
		}
		return ctrl.Run(stopCh)
	}
	return ctrlCtx.mgr.Add(manager.RunnableFunc(runnableFunc))
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
