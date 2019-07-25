package main

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/run"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	seedproxy "github.com/kubermatic/kubermatic/api/pkg/controller/seed-proxy"
	"github.com/kubermatic/kubermatic/api/pkg/controller/service-account"
	"github.com/kubermatic/kubermatic/api/pkg/controller/user-project-binding"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type minimalControllerContext struct {
	mgr                  manager.Manager
	workerCount          int
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	selector             func(*metav1.ListOptions)
}

type controllerCreator func(minimalControllerContext) (runnerFn, error)
type runnerFn func(workerCount int, stopCh <-chan struct{}) error

func noop(workerCount int, stopCh <-chan struct{}) error { <-stopCh; return nil }

// allControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller
var allControllers = map[string]controllerCreator{
	"RBAC":               createRBACContoller,
	"UserProjectBinding": createUserProjectBindingController,
	"ServiceAccounts":    createServiceAccountsController,
	"SeedProxy":          createSeedProxyController,
}

func createAllControllers(ctrlCtx minimalControllerContext) error {
	if err := createRBACContoller(ctrlCtx); err != nil {
		return fmt.Errorf("failed to create rbac controller: %v", err)
	}
	if err := userprojectbinding.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create userprojectbinding controller: %v", err)
	}
	if err := serviceaccount.Add(ctrlCtx.mgr); err != nil {
		return fmt.Errorf("failed to create serviceaccount controller: %v", err)
	}
}

func bla() {
	controllers := map[string]runnerFn{}
	for name, create := range allControllers {
		logger := log.Logger.With("controller", name)
		logger.Info("Creating controller")
		controller, err := create(ctrlCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create '%s' controller: %v", name, err)
		}
		// The controllers managed by the mgr don't have a dedicated runner
		if controller != nil {
			controllers[name] = controller
		}
	}
	return controllers, nil
}

func runAllControllersAndCtrlManager(workerCnt int,
	done <-chan struct{},
	cancel context.CancelFunc,
	mgr manager.Runnable,
	controllers map[string]runnerFn) error {
	var g run.Group

	wrapController := func(workerCnt int, done <-chan struct{}, cancel context.CancelFunc, name string, controller runnerFn) (func() error, func(err error)) {
		startControllerWrapped := func() error {
			logger := log.Logger.With("controller", name)
			logger.Info("Starting controller...")
			err := controller(workerCnt, done)
			logger.Infow("Controller finished/died", "error", err)
			return err
		}

		cancelControllerFunc := func(err error) {
			logger := log.Logger.With("controller", name)
			logger.Infow("Killing controller as group member finished/died", "error", err)
			cancel()
		}
		return startControllerWrapped, cancelControllerFunc
	}

	// run controller manager
	g.Add(func() error { return mgr.Start(done) }, func(_ error) { cancel() })

	// run controllers
	for name, startController := range controllers {
		startControllerWrapped, cancelControllerFunc := wrapController(workerCnt, done, cancel, name, startController)
		g.Add(startControllerWrapped, cancelControllerFunc)
	}

	return g.Run()
}

func createRBACContollerV2(ctrlCtx minimalControllerContext) error {
	masterClusterProvider, err := rbacClusterProvider(ctrlCtx.mgr.GetConfig(), "master", true, ctrlCtx.labelSelectorFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create master rbac provider: %v", err)
	}
	allClusterProviders := []*rbac.ClusterProvider{masterClusterProvider}

	seeds, err := ctrlCtx.seedsGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get seeds: %v", err)
	}
	for seedName := range seeds {
		kubeConfig, err := ctrlCtx.seedKubeconfigGetter(seedName)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig for seed %q: %v", seedName, err)
		}
		clusterProvider, err := rbacClusterProvider(kubeConfig, seedName, false, ctrlCtx.selector)
		if err != nil {
			return nil, fmt.Errorf("failed to create rbac provider for seed %q: %v", seedName, err)
		}
		allClusterProviders = append(allClusterProviders, clusterProvider)
	}

	ctrl, err := rbac.New(rbac.NewMetrics(), allClusterProviders, ctrlCtx.workerCount)
	if err != nil {
		return nil, err
	}

	// This is an implementation of
	// sigs.k8s.io/controller-runtime/pkg/manager.Runnable
	// It wraps the actual controllers implementation to defer the informer start
	runnable := func(stopCh <-chan struct{}) error {
		for _, clusterProvider := range allClusterProviders {
			clusterProvider.StartInformers(stopCh)
			if err := clusterProvider.WaitForCachesToSync(stopCh); err != nil {
				return fmt.Errorf("RBAC controller failed to sync cache: %v", err)
			}
		}
		return ctrl.Run(stopCh)
	}

	return ctrlCtx.mgr.Add(runnable)
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

	return rbac.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, name), kubeClient, kubermaticInformerFactory, kubermaticClient, kubermaticInformerFactory), nil
}

func createServiceAccountsController(ctrlCtx *controllerContext) (runnerFn, error) {
	if err := serviceaccount.Add(ctrlCtx.mgr); err != nil {
		return nil, err
	}
	return noop, nil
}

func createSeedProxyController(ctrlCtx *controllerContext) (runnerFn, error) {
	if err := seedproxy.Add(ctrlCtx.mgr, 1, ctrlCtx.seedsGetter, ctrlCtx.seedKubeconfigGetter); err != nil {
		return nil, err
	}
	return noop, nil
}
