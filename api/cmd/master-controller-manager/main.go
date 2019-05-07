package main

import (
	"context"
	"errors"
	"flag"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type controllerRunOptions struct {
	kubeconfig   string
	dcFile       string
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

	mgr               manager.Manager
	kubeconfig        *clientcmdapi.Config
	datacenters       map[string]provider.DatacenterMeta
	labelSelectorFunc func(*metav1.ListOptions)
}

func main() {
	var g run.Group
	ctrlCtx := &controllerContext{}
	flag.StringVar(&ctrlCtx.runOptions.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&ctrlCtx.runOptions.dcFile, "datacenters", "", "The datacenters.yaml file path")
	flag.StringVar(&ctrlCtx.runOptions.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&ctrlCtx.runOptions.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.runOptions.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&ctrlCtx.runOptions.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.Parse()

	log.SetLogger(log.ZapLogger(false))

	config, err := clientcmd.BuildConfigFromFlags(ctrlCtx.runOptions.masterURL, ctrlCtx.runOptions.kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to create the config for the kubernetes client: %v", err)
	}

	selector, err := workerlabel.LabelSelector(ctrlCtx.runOptions.workerName)
	if err != nil {
		glog.Fatalf("Failed to create the label selector for the given worker name '%s': %v", ctrlCtx.runOptions.workerName, err)
	}

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_master_controller_manager", prometheus.DefaultRegisterer)

	// register an operating system signals and context on which we will gracefully close the app
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()
	done := ctx.Done()
	ctrlCtx.stopCh = done

	ctrlCtx.kubeMasterClient = kubernetes.NewForConfigOrDie(config)
	ctrlCtx.kubermaticMasterClient = kubermaticclientset.NewForConfigOrDie(config)
	ctrlCtx.kubermaticMasterInformerFactory = externalversions.NewFilteredSharedInformerFactory(ctrlCtx.kubermaticMasterClient, informer.DefaultInformerResyncPeriod, metav1.NamespaceAll, selector)
	ctrlCtx.kubeMasterInformerFactory = kuberinformers.NewSharedInformerFactory(ctrlCtx.kubeMasterClient, informer.DefaultInformerResyncPeriod)
	ctrlCtx.labelSelectorFunc = selector

	ctrlCtx.datacenters, err = provider.LoadDatacentersMeta(ctrlCtx.runOptions.dcFile)
	if err != nil {
		glog.Fatalf("Failed to read the datacenters definition: %v", err)
	}

	ctrlCtx.kubeconfig, err = clientcmd.LoadFromFile(ctrlCtx.runOptions.kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to read the kubeconfig: %v", err)
	}

	{
		cfg, err := clientcmd.BuildConfigFromFlags(ctrlCtx.runOptions.masterURL, ctrlCtx.runOptions.kubeconfig)
		if err != nil {
			glog.Fatalf("failed to build config: %v", err)
		}

		mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: ctrlCtx.runOptions.internalAddr})
		if err != nil {
			glog.Fatalf("failed to create Controller Manager instance: %v", err)
		}
		if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
			glog.Fatalf("failed to register types in Scheme: %v", err)
		}
		ctrlCtx.mgr = mgr
	}

	controllers, err := createAllControllers(ctrlCtx)
	if err != nil {
		glog.Fatalf("could not create all controllers: %v", err)
	}

	ctrlCtx.kubermaticMasterInformerFactory.Start(ctrlCtx.stopCh)
	ctrlCtx.kubeMasterInformerFactory.Start(ctrlCtx.stopCh)
	ctrlCtx.kubermaticMasterInformerFactory.WaitForCacheSync(ctrlCtx.stopCh)
	ctrlCtx.kubeMasterInformerFactory.WaitForCacheSync(ctrlCtx.stopCh)

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

	// This group is running all controllers
	{
		g.Add(func() error {
			return runAllControllersAndCtrlManager(ctrlCtx.runOptions.workerCount, done, ctxDone, ctrlCtx.mgr, controllers)
		}, func(err error) {
			glog.Infof("Stopping Master Controller, due to = %v", err)
			ctxDone()
		})
	}

	if err := g.Run(); err != nil {
		glog.Fatal(err)
	}
}
