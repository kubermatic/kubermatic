package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	certificatectrl "github.com/kubermatic/kubermatic/api/pkg/controller/operator/certificate"
	masterctrl "github.com/kubermatic/kubermatic/api/pkg/controller/operator/master"
	seedctrl "github.com/kubermatic/kubermatic/api/pkg/controller/operator/seed"
	seedcontrollerlifecycle "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-lifecycle"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/pprof"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	// Do not import "sigs.k8s.io/controller-runtime/pkg" to prevent
	// duplicate kubeconfig flags being defined.
)

type controllerRunOptions struct {
	kubeconfig   string
	namespace    string
	internalAddr string
	log          kubermaticlog.Options
	workerCount  int
	workerName   string
}

func main() {
	klog.InitFlags(nil)
	pprofOpts := &pprof.Opts{}
	pprofOpts.AddFlags(flag.CommandLine)
	opt := &controllerRunOptions{}
	flag.StringVar(&opt.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if outside of cluster.")
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace the operator runs in, uses to determine where to look for KubermaticConfigurations.")
	flag.IntVar(&opt.workerCount, "worker-count", 4, "Number of workers which process reconcilings in parallel.")
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

	if len(opt.namespace) == 0 {
		log.Fatal("-namespace is a mandatory flag")
	}

	config, err := clientcmd.BuildConfigFromFlags("", opt.kubeconfig)
	if err != nil {
		log.Fatalw("Failed to build config", zap.Error(err))
	}

	mgr, err := manager.New(config, manager.Options{MetricsBindAddress: opt.internalAddr})
	if err != nil {
		log.Fatalw("Failed to create Controller Manager instance: %v", err)
	}

	if err := mgr.Add(pprofOpts); err != nil {
		log.Fatalw("Failed to add pprof endpoint", zap.Error(err))
	}

	if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", operatorv1alpha1.SchemeGroupVersion), zap.Error(err))
	}

	if err := apiextensionsv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", apiextensionsv1beta1.SchemeGroupVersion), zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedsGetter, err := provider.SeedsGetterFactory(ctx, mgr.GetClient(), "", opt.namespace, true)
	if err != nil {
		log.Fatalw("Failed to construct seedsGetter", zap.Error(err))
	}

	seedKubeconfigGetter, err := provider.SeedKubeconfigGetterFactory(ctx, mgr.GetClient(), "", opt.namespace, true)
	if err != nil {
		log.Fatalw("Failed to construct seedKubeconfigGetter", zap.Error(err))
	}

	if err := masterctrl.Add(ctx, mgr, log, opt.namespace, opt.workerCount, opt.workerName); err != nil {
		log.Fatalw("Failed to add operator-master controller", zap.Error(err))
	}

	if err := certificatectrl.Add(ctx, mgr, log, opt.namespace, opt.workerCount); err != nil {
		log.Fatalw("Failed to add operator-master-certificate controller", zap.Error(err))
	}

	ctrlCtx := seedOperatorContext{
		ctx:                  ctx,
		log:                  log,
		namespace:            opt.namespace,
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		workerCount:          opt.workerCount,
		workerName:           opt.workerName,
	}

	seedOperatorControllerFactory := seedOperatorControllerFactoryCreator(ctrlCtx)

	if err := seedcontrollerlifecycle.Add(ctx, log, mgr, opt.namespace, seedsGetter, seedKubeconfigGetter, seedOperatorControllerFactory); err != nil {
		log.Fatalw("Failed to create seed-lifecycle controller", zap.Error(err))
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalw("Cannot start manager", zap.Error(err))
	}
}

type seedOperatorContext struct {
	ctx                  context.Context
	log                  *zap.SugaredLogger
	namespace            string
	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	workerCount          int
	workerName           string
}

func seedOperatorControllerFactoryCreator(ctrlCtx seedOperatorContext) seedcontrollerlifecycle.ControllerFactory {
	factory := func(mgr manager.Manager) error {
		log := ctrlCtx.log.Named("operator-seed-controller-factory")

		seeds, err := ctrlCtx.seedsGetter()
		if err != nil {
			log.Errorw("Failed to get seeds", zap.Error(err))
			return fmt.Errorf("failed to get seeds: %v", err)
		}

		seedManagerMap := map[string]manager.Manager{}
		for seedName, seed := range seeds {
			log := log.With("seed", seed.Name)

			kubeconfig, err := ctrlCtx.seedKubeconfigGetter(seed)
			if err != nil {
				log.Errorw("Failed to get kubeconfig for seed", zap.Error(err))
				continue
			}

			seedMgr, err := manager.New(kubeconfig, manager.Options{MetricsBindAddress: "0"})
			if err != nil {
				log.Errorw("Failed to construct mgr for seed", zap.Error(err))
				continue
			}
			seedManagerMap[seedName] = seedMgr

			if err := mgr.Add(seedMgr); err != nil {
				return fmt.Errorf("failed to add controller manager for seed %q to mgr: %v", seedName, err)
			}
		}

		return seedctrl.Add(
			ctrlCtx.ctx,
			ctrlCtx.log,
			ctrlCtx.namespace,
			mgr,
			seedManagerMap,
			ctrlCtx.seedsGetter,
			ctrlCtx.workerCount,
			ctrlCtx.workerName)
	}

	return func(mgr manager.Manager) (string, error) {
		return seedctrl.ControllerName, factory(mgr)
	}
}
