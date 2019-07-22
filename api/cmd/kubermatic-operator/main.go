package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/go-logr/zapr"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
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
	internalAddr string
	log          kubermaticlog.Options
	workerCount  int
	workerName   string
}

type operatorContext struct {
	runOptions                      controllerRunOptions
	stopCh                          <-chan struct{}
	kubeMasterClient                kubernetes.Interface
	kubermaticMasterClient          kubermaticclientset.Interface
	kubermaticMasterInformerFactory externalversions.SharedInformerFactory
	kubeMasterInformerFactory       kuberinformers.SharedInformerFactory
	mgr                             manager.Manager
	clientConfig                    *clientcmdapi.Config
	labelSelectorFunc               func(*metav1.ListOptions)
}

func main() {
	var (
		g   run.Group
		err error
	)

	opCtx := &operatorContext{}
	flag.StringVar(&opCtx.runOptions.kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	flag.IntVar(&opCtx.runOptions.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&opCtx.runOptions.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.BoolVar(&opCtx.runOptions.log.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&opCtx.runOptions.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format, one of "+kubermaticlog.AvailableFormats.String())
	flag.StringVar(&opCtx.runOptions.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.Parse()

	rawLog := kubermaticlog.New(opCtx.runOptions.log.Debug, kubermaticlog.Format(opCtx.runOptions.log.Format)).Named(opCtx.runOptions.workerName)
	sugarLog := rawLog.Sugar()
	defer func() {
		if err := sugarLog.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	// update global logger instance
	kubermaticlog.Logger = sugarLog

	// set the logger used by sigs.k8s.io/controller-runtime
	log.SetLogger(zapr.NewLogger(rawLog))

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_operator", prometheus.DefaultRegisterer)

	// register an operating system signals and context on which we will gracefully close the app
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()
	done := ctx.Done()
	opCtx.stopCh = done

	selector, err := workerlabel.LabelSelector(opCtx.runOptions.workerName)
	if err != nil {
		sugarLog.Fatalw("Failed to create the label selector for the given worker", "error", err)
	}

	opCtx.clientConfig, err = clientcmd.LoadFromFile(opCtx.runOptions.kubeconfig)
	if err != nil {
		sugarLog.Fatalw("Failed to read the kubeconfig", "error", err)
	}

	config := clientcmd.NewNonInteractiveClientConfig(
		*opCtx.clientConfig,
		opCtx.clientConfig.CurrentContext,
		&clientcmd.ConfigOverrides{CurrentContext: opCtx.clientConfig.CurrentContext},
		nil,
	)

	cfg, err := config.ClientConfig()
	if err != nil {
		sugarLog.Fatalw("Failed to create client", "error", err)
	}

	opCtx.kubeMasterClient = kubernetes.NewForConfigOrDie(cfg)
	opCtx.kubermaticMasterClient = kubermaticclientset.NewForConfigOrDie(cfg)
	opCtx.kubermaticMasterInformerFactory = externalversions.NewFilteredSharedInformerFactory(opCtx.kubermaticMasterClient, informer.DefaultInformerResyncPeriod, metav1.NamespaceAll, selector)
	opCtx.kubeMasterInformerFactory = kuberinformers.NewSharedInformerFactory(opCtx.kubeMasterClient, informer.DefaultInformerResyncPeriod)
	opCtx.labelSelectorFunc = selector

	{
		mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: opCtx.runOptions.internalAddr})
		if err != nil {
			sugarLog.Fatalw("failed to create Controller Manager instance: %v", err)
		}
		if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			sugarLog.Fatalw("failed to register types in Scheme", "error", err)
		}
		opCtx.mgr = mgr
	}

	controllers, err := createAllControllers(opCtx, sugarLog)
	if err != nil {
		sugarLog.Fatalw("could not create all controllers", "error", err)
	}

	opCtx.kubermaticMasterInformerFactory.Start(opCtx.stopCh)
	opCtx.kubeMasterInformerFactory.Start(opCtx.stopCh)
	opCtx.kubermaticMasterInformerFactory.WaitForCacheSync(opCtx.stopCh)
	opCtx.kubeMasterInformerFactory.WaitForCacheSync(opCtx.stopCh)

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("a user has requested to stop the operator")
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
			return runAllControllersAndCtrlManager(opCtx.runOptions.workerCount, done, ctxDone, opCtx.mgr, controllers, sugarLog)
		}, func(err error) {
			sugarLog.Infow("Stopping Operator...", "error", err)
			ctxDone()
		})
	}

	if err := g.Run(); err != nil {
		sugarLog.Fatalw("Cannot start Operator", "error", err)
	}
}
