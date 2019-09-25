package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	mastermigrations "github.com/kubermatic/kubermatic/api/pkg/crd/migrations/master"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	metricserver "github.com/kubermatic/kubermatic/api/pkg/metrics/server"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"
	seedvalidation "github.com/kubermatic/kubermatic/api/pkg/validation/seed"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	log                kubermaticlog.Options
	seedvalidationHook seedvalidation.WebhookOpts

	workerName string
}

type controllerContext struct {
	ctx                  context.Context
	mgr                  manager.Manager
	log                  *zap.SugaredLogger
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
	flag.StringVar(&runOpts.dcFile, "datacenters", "", "The datacenters.yaml file path.")
	flag.StringVar(&runOpts.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&runOpts.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&runOpts.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the /metrics endpoint will be served.")
	flag.BoolVar(&runOpts.dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters. Enabling this and defining the datcenters flag will enable the migration of the datacenters defined in datancenters.yaml to Seed custom resources.")
	flag.StringVar(&ctrlCtx.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources.")
	flag.BoolVar(&runOpts.log.Debug, "log-debug", false, "Enables debug logging.")
	flag.StringVar(&runOpts.log.Format, "log-format", string(kubermaticlog.FormatJSON), "Log format. Available are: "+kubermaticlog.AvailableFormats.String())
	flag.Parse()

	ctrlruntimelog.SetLogger(ctrlruntimelog.ZapLogger(false))
	rawLog := kubermaticlog.New(runOpts.log.Debug, kubermaticlog.Format(runOpts.log.Format))
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	kubermaticlog.Logger = log
	ctrlCtx.log = log

	selector, err := workerlabel.LabelSelector(runOpts.workerName)
	if err != nil {
		log.Fatalw("failed to create the label selector for the given worker", "workerName", runOpts.workerName, zap.Error(err))
	}

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

	// load kubeconfig and create API client
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

	cfg, err := config.ClientConfig()
	if err != nil {
		log.Fatalw("failed to create client", zap.Error(err))
	}

	ctrlCtx.labelSelectorFunc = func(listOpts *metav1.ListOptions) {
		listOpts.LabelSelector = selector.String()
	}

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: ""})
	if err != nil {
		log.Fatalw("failed to create Controller Manager instance", zap.Error(err))
	}
	ctrlCtx.mgr = mgr
	ctrlCtx.seedsGetter, err = provider.SeedsGetterFactory(ctx, ctrlCtx.mgr.GetClient(), runOpts.dcFile, ctrlCtx.namespace, runOpts.workerName, runOpts.dynamicDatacenters)
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
			runOpts.workerName,
			ctrlCtx.seedsGetter,
			provider.SeedClientGetterFactory(ctrlCtx.seedKubeconfigGetter),
			migrationOptions.SeedMigrationEnabled())
		if err != nil {
			log.Fatalw("failed to create validatingAdmissionWebhook server for seeds", zap.Error(err))
		}

		// This group starts the validation webhook server; it's not using the
		// mgr because we want the webhook to run so that migrations can perform
		// operations on seeds
		{
			g.Add(func() error {
				return seedValidationWebhookServer.Start(ctx.Done())
			}, func(err error) {
				ctxCancel()
			})
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

			return leaderelection.RunAsLeader(leaderCtx, log, cfg, mgr.GetRecorder(controllerName), electionName, func(ctx context.Context) error {
				// create a dedicated client because the one from the manager is not yet
				// initialized and returns 404's for everything
				client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
				if err != nil {
					return fmt.Errorf("failed to create kube client: %v", err)
				}

				// wait for the webhook to be ready
				if migrationOptions.SeedMigrationEnabled() {
					timeout := 30 * time.Second
					endpoint := types.NamespacedName{Namespace: ctrlCtx.namespace, Name: "seed-webhook"}

					log.Infow("waiting for webhook to be ready...", "webhook", endpoint, "timeout", timeout)
					if err := wait.Poll(500*time.Millisecond, timeout, func() (bool, error) {
						endpoints := &corev1.Endpoints{}
						if err := client.Get(ctx, endpoint, endpoints); err != nil {
							return false, err
						}
						return len(endpoints.Subsets) > 0, nil
					}); err != nil {
						return fmt.Errorf("failed to wait for webhook: %v", err)
					}
					log.Info("webhook is ready")
				}

				log.Info("executing migrations...")
				if err := mastermigrations.RunAll(ctx, log, client, ctrlCtx.namespace, migrationOptions); err != nil {
					return fmt.Errorf("failed to run migrations: %v", err)
				}
				log.Info("migrations executed successfully")

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
