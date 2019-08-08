package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	metricserver "github.com/kubermatic/kubermatic/api/pkg/metrics/server"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"
	seedvalidation "github.com/kubermatic/kubermatic/api/pkg/validation/seed"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
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
	seedvalidationHook seedvalidation.WebhookOpts

	workerName string
}

type controllerContext struct {
	ctx                  context.Context
	mgr                  manager.Manager
	workerCount          int
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	labelSelectorFunc    func(*metav1.ListOptions)
	namespace            string
}

func main() {
	var g run.Group
	ctrlCtx := &controllerContext{}
	runOpts := controllerRunOptions{}
	runOpts.seedvalidationHook.AddFlags(flag.CommandLine)
	flag.StringVar(&runOpts.kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	flag.StringVar(&runOpts.dcFile, "datacenters", "", "The datacenters.yaml file path")
	flag.StringVar(&runOpts.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&runOpts.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&runOpts.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.BoolVar(&runOpts.dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters")
	flag.StringVar(&ctrlCtx.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources")
	flag.BoolVar(&runOpts.log.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&runOpts.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())
	flag.Parse()

	log.SetLogger(log.ZapLogger(false))
	rawLog := kubermaticlog.New(runOpts.log.Debug, kubermaticlog.Format(runOpts.log.Format))
	sugarLog := rawLog.Sugar()
	defer func() {
		if err := sugarLog.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	kubermaticlog.Logger = sugarLog

	selector, err := workerlabel.LabelSelector(runOpts.workerName)
	if err != nil {
		sugarLog.Fatalw("Failed to create the label selector for the given worker", "workerName", runOpts.workerName, zap.Error(err))
	}

	if err := kubermaticv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to add kubermaticv1 scheme to scheme.Scheme", zap.Error(err))
	}

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_master_controller_manager", prometheus.DefaultRegisterer)

	// register an operating system signals and context on which we will gracefully close the app
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()
	done := ctx.Done()
	ctrlCtx.ctx = ctx

	kubeconfig, err := clientcmd.LoadFromFile(runOpts.kubeconfig)
	if err != nil {
		sugarLog.Fatalw("Failed to read the kubeconfig", zap.Error(err))
	}

	config := clientcmd.NewNonInteractiveClientConfig(
		*kubeconfig,
		kubeconfig.CurrentContext,
		&clientcmd.ConfigOverrides{CurrentContext: kubeconfig.CurrentContext},
		nil,
	)

	cfg, err := config.ClientConfig()
	if err != nil {
		sugarLog.Fatalw("Failed to create client", zap.Error(err))
	}

	ctrlCtx.labelSelectorFunc = func(listOpts *metav1.ListOptions) {
		listOpts.LabelSelector = selector.String()
	}

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: ""})
	if err != nil {
		sugarLog.Fatalw("failed to create Controller Manager instance", zap.Error(err))
	}
	if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		sugarLog.Fatalw("failed to register types in Scheme", zap.Error(err))
	}
	ctrlCtx.mgr = mgr
	ctrlCtx.seedsGetter, err = provider.SeedsGetterFactory(ctx, ctrlCtx.mgr.GetClient(), runOpts.dcFile, ctrlCtx.namespace, runOpts.workerName, runOpts.dynamicDatacenters)
	if err != nil {
		sugarLog.Fatalw("failed to construct seedsGetter", zap.Error(err))
	}
	ctrlCtx.seedKubeconfigGetter, err = provider.SeedKubeconfigGetterFactory(
		ctx, mgr.GetClient(), runOpts.kubeconfig, ctrlCtx.namespace, runOpts.dynamicDatacenters)
	if err != nil {
		sugarLog.Fatalw("failed to construct seedKubeconfigGetter", zap.Error(err))
	}

	seedValidationWebhookServer, err := runOpts.seedvalidationHook.Server(
		ctx, sugarLog, runOpts.workerName, ctrlCtx.seedsGetter, ctrlCtx.seedKubeconfigGetter)
	if err != nil {
		sugarLog.Fatalw("failed to create validatingAdmissionWebhook server for seeds", zap.Error(err))
	}
	if err := mgr.Add(seedValidationWebhookServer); err != nil {
		sugarLog.Fatalf("failed to add validatingAdmissionWebhook server to mgr", zap.Error(err))
	}

	if err := createAllControllers(ctrlCtx); err != nil {
		sugarLog.Fatalw("could not create all controllers", zap.Error(err))
	}

	if err := mgr.Add(metricserver.New(runOpts.internalAddr)); err != nil {
		sugarLog.Fatalw("failed to add metrics server", zap.Error(err))
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

	// This group is running all controllers
	{
		g.Add(func() error {
			return mgr.Start(ctx.Done())
		}, func(err error) {
			sugarLog.Infow("Stopping Master Controller", zap.Error(err))
			ctxDone()
		})
	}

	if err := g.Run(); err != nil {
		sugarLog.Fatalw("Can not start Master Controller", zap.Error(err))
	}
}
