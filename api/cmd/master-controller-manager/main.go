package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	mastermigrations "github.com/kubermatic/kubermatic/api/pkg/crd/migrations/master"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	metricserver "github.com/kubermatic/kubermatic/api/pkg/metrics/server"
	"github.com/kubermatic/kubermatic/api/pkg/pprof"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"
	seedvalidation "github.com/kubermatic/kubermatic/api/pkg/validation/seed"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrlruntimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	controllerName = "kubermatic-master-controller-manager"
)

type controllerRunOptions struct {
	kubeconfig         string
	dcFile             string
	masterURL          string
	internalAddr       string
	dynamicDatacenters bool
	seedvalidationHook seedvalidation.WebhookOpts

	workerName string
}

type controllerContext struct {
	ctx                     context.Context
	mgr                     manager.Manager
	log                     *zap.SugaredLogger
	workerCount             int
	workerName              string
	workerNameLabelSelector labels.Selector
	seedsGetter             provider.SeedsGetter
	seedKubeconfigGetter    provider.SeedKubeconfigGetter
	labelSelectorFunc       func(*metav1.ListOptions)
	namespace               string
}

func main() {
	var g run.Group
	ctrlCtx := &controllerContext{}
	runOpts := controllerRunOptions{}
	klog.InitFlags(nil)
	pprofOpts := &pprof.Opts{}
	pprofOpts.AddFlags(flag.CommandLine)
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	runOpts.seedvalidationHook.AddFlags(flag.CommandLine)
	flag.StringVar(&runOpts.kubeconfig, "kubeconfig", "", "Path to a kubeconfig.")
	flag.StringVar(&runOpts.dcFile, "datacenters", "", "The datacenters.yaml file path.")
	flag.StringVar(&runOpts.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&runOpts.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&runOpts.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served.")
	flag.BoolVar(&runOpts.dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters. Enabling this and defining the datcenters flag will enable the migration of the datacenters defined in datancenters.yaml to Seed custom resources.")
	flag.StringVar(&ctrlCtx.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources.")
	flag.Parse()

	ctrlruntimelog.SetLogger(ctrlruntimelog.ZapLogger(false))
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	kubermaticlog.Logger = log
	ctrlCtx.log = log
	ctrlCtx.workerName = runOpts.workerName

	selector, err := workerlabel.LabelSelector(runOpts.workerName)
	if err != nil {
		log.Fatalw("failed to create the label selector for the given worker", "workerName", runOpts.workerName, zap.Error(err))
	}
	ctrlCtx.workerNameLabelSelector = selector

	// register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_master_controller_manager", prometheus.DefaultRegisterer)

	// register an operating system signals and context on which we will gracefully close the app
	stopCh := signals.SetupSignalHandler()

	// prepare a context to use throughout the controller manager
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	ctrlCtx.ctx = ctx

	// prepare migration options
	migrationOptions := mastermigrations.MigrationOptions{
		DatacentersFile:    runOpts.dcFile,
		DynamicDatacenters: runOpts.dynamicDatacenters,
	}

	// load kubeconfig and create API client; kubeconfig can be empty if no Seed CRD migrations need to run
	var cfg *restclient.Config

	if runOpts.kubeconfig == "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			log.Fatalw("failed to load in-cluster credentials", zap.Error(err))
		}
	} else {
		kubeconfig, err := clientcmd.LoadFromFile(runOpts.kubeconfig)
		if err != nil {
			log.Fatalw("failed to read the kubeconfig", zap.Error(err))
		}

		config := clientcmd.NewNonInteractiveClientConfig(
			*kubeconfig,
			kubeconfig.CurrentContext,
			&clientcmd.ConfigOverrides{CurrentContext: kubeconfig.CurrentContext},
			nil,
		)

		cfg, err = config.ClientConfig()
		if err != nil {
			log.Fatalw("failed to create client", zap.Error(err))
		}

		migrationOptions.Kubeconfig = kubeconfig
	}

	ctrlCtx.labelSelectorFunc = func(listOpts *metav1.ListOptions) {
		listOpts.LabelSelector = selector.String()
	}

	// create a new cache that we can start independently of the manager, so that other
	// components work even if the manager has not yet been started
	cache, err := ctrlruntimecache.New(cfg, ctrlruntimecache.Options{})
	if err != nil {
		log.Fatalw("failed to create cache", zap.Error(err))
	}

	go func() {
		if err := cache.Start(ctx.Done()); err != nil {
			log.Fatalw("failed to start cache", zap.Error(err))
		}
	}()

	if !cache.WaitForCacheSync(ctx.Done()) {
		log.Fatal("failed to wait for caches to synchronize")
	}

	mgr, err := manager.New(cfg, manager.Options{
		MetricsBindAddress: "",
		// inject the handcrafted cache into the manager
		NewCache: func(config *rest.Config, opts ctrlruntimecache.Options) (ctrlruntimecache.Cache, error) {
			return &unstartableCache{cache}, nil
		},
	})
	if err != nil {
		log.Fatalw("failed to create Controller Manager instance", zap.Error(err))
	}
	ctrlCtx.mgr = mgr

	if err := mgr.Add(pprofOpts); err != nil {
		log.Fatalw("Failed to add pprof endpoint", zap.Error(err))
	}

	// these two getters rely on the ctrlruntime manager being started; they are
	// only used inside controllers
	ctrlCtx.seedsGetter, err = provider.SeedsGetterFactory(ctx, mgr.GetClient(), runOpts.dcFile, ctrlCtx.namespace, runOpts.dynamicDatacenters)
	if err != nil {
		log.Fatalw("failed to construct seedsGetter", zap.Error(err))
	}
	ctrlCtx.seedKubeconfigGetter, err = provider.SeedKubeconfigGetterFactory(
		ctx, mgr.GetClient(), runOpts.kubeconfig, ctrlCtx.namespace, runOpts.dynamicDatacenters)
	if err != nil {
		log.Fatalw("failed to construct seedKubeconfigGetter", zap.Error(err))
	}

	if runOpts.seedvalidationHook.CertFile != "" || runOpts.seedvalidationHook.KeyFile != "" {
		seedValidationWebhookServer, err := runOpts.seedvalidationHook.Server(
			ctx,
			log,
			ctrlCtx.namespace,
			runOpts.workerName,
			ctrlCtx.seedsGetter,
			provider.SeedClientGetterFactory(ctrlCtx.seedKubeconfigGetter),
			migrationOptions.SeedMigrationEnabled())
		if err != nil {
			log.Fatalw("failed to create validatingAdmissionWebhook server for seeds", zap.Error(err))
		}
		if err := mgr.Add(seedValidationWebhookServer); err != nil {
			log.Fatalw("failed to add the seedValidationWebhookServer to the mgr", zap.Error(err))
		}

	} else {
		log.Info("the validatingAdmissionWebhook server can not be started because seed-admissionwebhook-cert-file and seed-admissionwebhook-key-file are empty")
	}

	if err := createAllControllers(ctrlCtx); err != nil {
		log.Fatalw("could not create all controllers", zap.Error(err))
	}

	if err := mgr.Add(metricserver.New(runOpts.internalAddr)); err != nil {
		log.Fatalw("failed to add metrics server", zap.Error(err))
	}

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("a user has requested to stop the controller")
			case <-ctx.Done():
				return errors.New("parent context has been closed - propagating the request")
			}
		}, func(err error) {
			ctxCancel()
		})
	}

	// This group is running the actual controller logic
	{
		// This group is running the actual controller logic
		leaderCtx, stopLeaderElection := context.WithCancel(ctx)
		defer stopLeaderElection()

		g.Add(func() error {
			electionName := controllerName + "-leader-election"
			if runOpts.workerName != "" {
				electionName += "-" + runOpts.workerName
			}

			return leaderelection.RunAsLeader(leaderCtx, log, cfg, mgr.GetEventRecorderFor(controllerName), electionName, func(ctx context.Context) error {
				if migrationOptions.MigrationEnabled() {
					log.Info("executing migrations...")

					if err := mastermigrations.RunAll(ctx, log, mgr.GetClient(), ctrlCtx.namespace, migrationOptions); err != nil {
						return fmt.Errorf("failed to run migrations: %v", err)
					}

					log.Info("migrations executed successfully")
				}

				log.Info("starting the master-controller-manager...")
				if err := mgr.Start(ctx.Done()); err != nil {
					return fmt.Errorf("the controller-manager stopped with an error: %v", err)
				}
				return nil
			})
		}, func(err error) {
			stopLeaderElection()
		})
	}

	if err := g.Run(); err != nil {
		log.Fatalw("cannot start the master-controller-manager", zap.Error(err))
	}
}

// unstartableCache is used to prevent the ctrlruntime manager from starting the
// cache *again*, just after we started and initialized it.
type unstartableCache struct {
	ctrlruntimecache.Cache
}

func (m *unstartableCache) Start(_ <-chan struct{}) error {
	return nil
}

func (m *unstartableCache) WaitForCacheSync(_ <-chan struct{}) bool {
	return true
}
