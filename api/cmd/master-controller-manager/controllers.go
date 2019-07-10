package main

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/run"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	serviceaccount "github.com/kubermatic/kubermatic/api/pkg/controller/service-account"
	userprojectbinding "github.com/kubermatic/kubermatic/api/pkg/controller/user-project-binding"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type controllerCreator func(*controllerContext) (runnerFn, error)
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

func createAllControllers(ctrlCtx *controllerContext) (map[string]runnerFn, error) {
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

func createRBACContoller(ctrlCtx *controllerContext) (runnerFn, error) {
	allClusterProviders := []*rbac.ClusterProvider{}
	{
		clientcmdConfig, err := clientcmd.LoadFromFile(ctrlCtx.runOptions.kubeconfig)
		if err != nil {
			log.Logger.Fatal(err)
		}

		for ctxName := range clientcmdConfig.Contexts {
			logger := log.Logger.With("ctxName", ctxName)
			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*clientcmdConfig,
				ctxName,
				&clientcmd.ConfigOverrides{CurrentContext: ctxName},
				nil,
			)
			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				logger.Fatal(err)
			}

			var clusterPrefix string
			if ctxName == clientcmdConfig.CurrentContext {
				logger.Info("Adding as master cluster")
				clusterPrefix = rbac.MasterProviderPrefix
			} else {
				logger.Info("Adding as seed cluster")
				clusterPrefix = rbac.SeedProviderPrefix
			}
			kubeClient, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				logger.Fatal(err)
			}

			kubermaticClient := kubermaticclientset.NewForConfigOrDie(cfg)
			kubermaticInformerFactory := externalversions.NewFilteredSharedInformerFactory(kubermaticClient, time.Minute*5, metav1.NamespaceAll, ctrlCtx.labelSelectorFunc)
			kubeInformerProvider := rbac.NewInformerProvider(kubeClient, time.Minute*5)
			allClusterProviders = append(allClusterProviders, rbac.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, ctxName), kubeClient, kubeInformerProvider, kubermaticClient, kubermaticInformerFactory))

			// special case the current context/master is also a seed cluster
			// we keep cluster resources also on master
			if ctxName == clientcmdConfig.CurrentContext {
				logger.Info("Special case adding current context also as seed cluster")
				clusterPrefix = rbac.SeedProviderPrefix
				allClusterProviders = append(allClusterProviders, rbac.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, ctxName), kubeClient, kubeInformerProvider, kubermaticClient, kubermaticInformerFactory))
			}
		}
	}

	ctrl, err := rbac.New(rbac.NewMetrics(), allClusterProviders)
	if err != nil {
		return nil, err
	}

	return func(workerCount int, stopCh <-chan struct{}) error {

		for _, clusterProvider := range allClusterProviders {
			clusterProvider.StartInformers(ctrlCtx.stopCh)
			if err := clusterProvider.WaitForCachesToSync(ctrlCtx.stopCh); err != nil {
				return fmt.Errorf("RBAC controller failed to sync cache: %v", err)
			}
		}

		// TODO: change ctrl.Run to return an err
		ctrl.Run(workerCount, stopCh)
		return nil
	}, nil
}

func createUserProjectBindingController(ctrlCtx *controllerContext) (runnerFn, error) {
	if err := userprojectbinding.Add(ctrlCtx.mgr); err != nil {
		return nil, err
	}
	return noop, nil
}

func createServiceAccountsController(ctrlCtx *controllerContext) (runnerFn, error) {
	if err := serviceaccount.Add(ctrlCtx.mgr); err != nil {
		return nil, err
	}
	return noop, nil
}

func createSeedProxyController(ctrlCtx *controllerContext) (runnerFn, error) {
	// if err := seedproxy.Add(ctrlCtx.mgr, 1, ctrlCtx.kubeconfig, ctrlCtx.datacenters); err != nil {
	// 	return nil, err
	// }
	return noop, nil
}
