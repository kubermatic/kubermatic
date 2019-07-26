package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
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
	kubeconfig         string
	dcFile             string
	masterURL          string
	internalAddr       string
	dynamicDatacenters bool
	log                kubermaticlog.Options

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

	mgr                  manager.Manager
	kubeconfig           *clientcmdapi.Config
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	labelSelectorFunc    func(*metav1.ListOptions)
}

func main() {
	var g run.Group
	ctrlCtx := &controllerContext{}
	flag.StringVar(&ctrlCtx.runOptions.kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	flag.StringVar(&ctrlCtx.runOptions.dcFile, "datacenters", "", "The datacenters.yaml file path")
	flag.StringVar(&ctrlCtx.runOptions.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&ctrlCtx.runOptions.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.runOptions.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&ctrlCtx.runOptions.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.BoolVar(&ctrlCtx.runOptions.dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters")
	flag.BoolVar(&ctrlCtx.runOptions.log.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&ctrlCtx.runOptions.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())
	flag.Parse()

	log.SetLogger(log.ZapLogger(false))
	rawLog := kubermaticlog.New(ctrlCtx.runOptions.log.Debug, kubermaticlog.Format(ctrlCtx.runOptions.log.Format))
	sugarLog := rawLog.Sugar()
	defer func() {
		if err := sugarLog.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	kubermaticlog.Logger = sugarLog

	selector, err := workerlabel.LabelSelector(ctrlCtx.runOptions.workerName)
	if err != nil {
		sugarLog.Fatalw("Failed to create the label selector for the given worker", "workerName", ctrlCtx.runOptions.workerName, "error", err)
	}

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_master_controller_manager", prometheus.DefaultRegisterer)

	// register an operating system signals and context on which we will gracefully close the app
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()
	done := ctx.Done()
	ctrlCtx.stopCh = done

	ctrlCtx.kubeconfig, err = clientcmd.LoadFromFile(ctrlCtx.runOptions.kubeconfig)
	if err != nil {
		sugarLog.Fatalw("Failed to read the kubeconfig", "error", err)
	}

	config := clientcmd.NewNonInteractiveClientConfig(
		*ctrlCtx.kubeconfig,
		ctrlCtx.kubeconfig.CurrentContext,
		&clientcmd.ConfigOverrides{CurrentContext: ctrlCtx.kubeconfig.CurrentContext},
		nil,
	)

	cfg, err := config.ClientConfig()
	if err != nil {
		sugarLog.Fatalw("Failed to create client", "error", err)
	}

	ctrlCtx.kubeMasterClient = kubernetes.NewForConfigOrDie(cfg)
	ctrlCtx.kubermaticMasterClient = kubermaticclientset.NewForConfigOrDie(cfg)
	ctrlCtx.kubermaticMasterInformerFactory = externalversions.NewFilteredSharedInformerFactory(ctrlCtx.kubermaticMasterClient, informer.DefaultInformerResyncPeriod, metav1.NamespaceAll, selector)
	ctrlCtx.kubeMasterInformerFactory = kuberinformers.NewSharedInformerFactory(ctrlCtx.kubeMasterClient, informer.DefaultInformerResyncPeriod)
	ctrlCtx.labelSelectorFunc = selector

	ctrlCtx.kubeconfig, err = clientcmd.LoadFromFile(ctrlCtx.runOptions.kubeconfig)
	if err != nil {
		sugarLog.Fatalw("Failed to read the kubeconfig", "error", err)
	}

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: ctrlCtx.runOptions.internalAddr})
	if err != nil {
		sugarLog.Fatalw("failed to create Controller Manager instance: %v", err)
	}
	if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		sugarLog.Fatalw("failed to register types in Scheme", "error", err)
	}
	ctrlCtx.mgr = mgr
	ctrlCtx.seedsGetter, err = provider.SeedsGetterFactory(ctx, ctrlCtx.mgr.GetClient(), ctrlCtx.runOptions.dcFile, ctrlCtx.runOptions.workerName, ctrlCtx.runOptions.dynamicDatacenters)
	if err != nil {
		sugarLog.Fatalw("failed to get construct seedsGetter", "error", err)
	}
	ctrlCtx.seedKubeconfigGetter, err = provider.SeedKubeconfigGetterFactory(
		ctx, mgr.GetClient(), ctrlCtx.runOptions.kubeconfig, ctrlCtx.runOptions.dynamicDatacenters)
	if err != nil {
		sugarLog.Fatalf("failed to construct seedKubeconfigGetter", "error", err)
	}

	controllers, err := createAllControllers(ctrlCtx)
	if err != nil {
		sugarLog.Fatalw("could not create all controllers", "error", err)
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
			sugarLog.Infow("Stopping Master Controller", "error", err)
			ctxDone()
		})
	}

	if err := g.Run(); err != nil {
		sugarLog.Fatalw("Can not start Master Controller", "error", err)
	}
}
