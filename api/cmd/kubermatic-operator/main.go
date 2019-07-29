package main

import (
	"flag"
	"fmt"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	operatormaster "github.com/kubermatic/kubermatic/api/pkg/controller/operator-master"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/signals"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	// Do not import "sigs.k8s.io/controller-runtime/pkg" to prevent
	// duplicate kubeconfig flags being defined.
)

type controllerRunOptions struct {
	kubeconfig   string
	internalAddr string
	log          kubermaticlog.Options
	workerCount  int
	workerName   string
}

func main() {
	opt := &controllerRunOptions{}
	flag.StringVar(&opt.kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	flag.IntVar(&opt.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&opt.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served")
	flag.BoolVar(&opt.log.Debug, "log-debug", false, "Enables debug logging")
	flag.StringVar(&opt.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format, one of "+kubermaticlog.AvailableFormats.String())
	flag.StringVar(&opt.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.Parse()

	rawLog := kubermaticlog.New(opt.log.Debug, kubermaticlog.Format(opt.log.Format)).Named(opt.workerName)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	// update global logger instance
	kubermaticlog.Logger = log

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrllog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	clientConfig, err := clientcmd.LoadFromFile(opt.kubeconfig)
	if err != nil {
		log.Fatalw("Failed to read the kubeconfig", "error", err)
	}

	config := clientcmd.NewNonInteractiveClientConfig(
		*clientConfig,
		clientConfig.CurrentContext,
		&clientcmd.ConfigOverrides{CurrentContext: clientConfig.CurrentContext},
		nil,
	)

	cfg, err := config.ClientConfig()
	if err != nil {
		log.Fatalw("Failed to create client", "error", err)
	}

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: opt.internalAddr})
	if err != nil {
		log.Fatalw("Failed to create Controller Manager instance: %v", err)
	}

	if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register types in Scheme", "error", err)
	}

	if err := operatormaster.Add(mgr, 1, clientConfig, log, opt.workerName); err != nil {
		log.Fatalw("Failed to add operator-master controller", "error", err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalw("Cannot start manager", "error", err)
	}
}
