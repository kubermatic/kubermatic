package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac/user-project-binding"
	"github.com/kubermatic/kubermatic/api/pkg/controller/service-account"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type controllerRunOptions struct {
	kubeconfig   string
	masterURL    string
	internalAddr string

	workerName  string
	workerCount int
}

type controllerContext struct {
	runOptions                      controllerRunOptions
	stopCh                          <-chan struct{}
	kubeMasterClient                kubernetes.Interface
	kubermaticMasterClient          kubermaticclientset.Interface
	kubermaticMasterInformerFactory externalversions.SharedInformerFactory
	kubeMasterInformerFactory       kuberinformers.SharedInformerFactory
	allClusterProviders             []*rbac.ClusterProvider
}

func main() {
	var g run.Group
	ctrlCtx := controllerContext{}
	flag.StringVar(&ctrlCtx.runOptions.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&ctrlCtx.runOptions.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&ctrlCtx.runOptions.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.runOptions.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&ctrlCtx.runOptions.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(ctrlCtx.runOptions.masterURL, ctrlCtx.runOptions.kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	selector, err := workerlabel.LabelSelector(ctrlCtx.runOptions.workerName)
	if err != nil {
		glog.Fatal(err)
	}

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_rbac_generator", prometheus.DefaultRegisterer)

	// register an operating system signals on which we will gracefully close the app
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()
	done := ctx.Done()

	ctrlCtx.stopCh = done
	ctrlCtx.kubeMasterClient = kubernetes.NewForConfigOrDie(config)
	ctrlCtx.kubermaticMasterClient = kubermaticclientset.NewForConfigOrDie(config)
	ctrlCtx.kubermaticMasterInformerFactory = externalversions.NewFilteredSharedInformerFactory(ctrlCtx.kubermaticMasterClient, informer.DefaultInformerResyncPeriod, metav1.NamespaceAll, selector)
	ctrlCtx.kubeMasterInformerFactory = kuberinformers.NewSharedInformerFactory(ctrlCtx.kubeMasterClient, informer.DefaultInformerResyncPeriod)

	ctrlCtx.allClusterProviders = []*rbac.ClusterProvider{}
	{
		clientcmdConfig, err := clientcmd.LoadFromFile(ctrlCtx.runOptions.kubeconfig)
		if err != nil {
			glog.Fatal(err)
		}

		for ctxName := range clientcmdConfig.Contexts {
			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*clientcmdConfig,
				ctxName,
				&clientcmd.ConfigOverrides{CurrentContext: ctxName},
				nil,
			)
			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				glog.Fatal(err)
			}

			var clusterPrefix string
			if ctxName == clientcmdConfig.CurrentContext {
				glog.V(2).Infof("Adding %s as master cluster", ctxName)
				clusterPrefix = rbac.MasterProviderPrefix
			} else {
				glog.V(2).Infof("Adding %s as seed cluster", ctxName)
				clusterPrefix = rbac.SeedProviderPrefix
			}
			kubeClient, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				glog.Fatal(err)
			}

			kubermaticClient := kubermaticclientset.NewForConfigOrDie(cfg)
			kubermaticInformerFactory := externalversions.NewFilteredSharedInformerFactory(kubermaticClient, time.Minute*5, metav1.NamespaceAll, selector)
			kubeInformerProvider := rbac.NewInformerProvider(kubeClient, time.Minute*5)
			ctrlCtx.allClusterProviders = append(ctrlCtx.allClusterProviders, rbac.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, ctxName), kubeClient, kubeInformerProvider, kubermaticClient, kubermaticInformerFactory))

			// special case the current context/master is also a seed cluster
			// we keep cluster resources also on master
			if ctxName == clientcmdConfig.CurrentContext {
				glog.V(2).Infof("Special case adding %s (current context) also as seed cluster", ctxName)
				clusterPrefix = rbac.SeedProviderPrefix
				ctrlCtx.allClusterProviders = append(ctrlCtx.allClusterProviders, rbac.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, ctxName), kubeClient, kubeInformerProvider, kubermaticClient, kubermaticInformerFactory))
			}
		}
	}

	ctrl, err := rbac.New(
		rbac.NewMetrics(),
		ctrlCtx.allClusterProviders)
	if err != nil {
		glog.Fatal(err)
	}

	ctrlCtx.kubermaticMasterInformerFactory.Start(ctrlCtx.stopCh)
	ctrlCtx.kubeMasterInformerFactory.Start(ctrlCtx.stopCh)
	ctrlCtx.kubermaticMasterInformerFactory.WaitForCacheSync(ctrlCtx.stopCh)
	ctrlCtx.kubeMasterInformerFactory.WaitForCacheSync(ctrlCtx.stopCh)

	for _, seedClusterProvider := range ctrlCtx.allClusterProviders {
		seedClusterProvider.StartInformers(ctrlCtx.stopCh)
		if err := seedClusterProvider.WaitForCachesToSync(ctrlCtx.stopCh); err != nil {
			glog.Fatalf("Closing the controller, failed to sync cache: %v", err)
		}
	}

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("a user has requested to stop the controller")
			case <-done:
				return errors.New("parent context has been closed - propagating the request")
			}
		}, func(err error) {
			ctxDone()
		})
	}

	// This group is running the actual rbac logic
	{
		g.Add(func() error {
			// controller will return iff ctrlCtx is stopped
			ctrl.Run(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh)
			return nil
		}, func(err error) {
			glog.Infof("Stopping RBACGenerator controller, err = %v", err)
		})
	}

	// This group is running the controller manager
	{
		g.Add(func() error {
			cfg, err := clientcmd.BuildConfigFromFlags(ctrlCtx.runOptions.masterURL, ctrlCtx.runOptions.kubeconfig)
			if err != nil {
				return err
			}

			mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: ctrlCtx.runOptions.internalAddr})
			if err != nil {
				glog.Errorf("failed to start RBACGenerator manager: %v", err)
				return err
			}
			if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
				return err
			}

			if err := userprojectbinding.Add(mgr); err != nil {
				return err
			}
			if err := serviceaccount.Add(mgr); err != nil {
				return err
			}

			if err := mgr.Start(ctrlCtx.stopCh); err != nil {
				glog.Errorf("failed to start RBACGenerator manager: %v", err)
				return err
			}
			return nil
		}, func(err error) {
			glog.Infof("Stopping RBACGenerator manager, err = %v", err)
		})
	}

	if err := g.Run(); err != nil {
		glog.Fatal(err)
	}
}
