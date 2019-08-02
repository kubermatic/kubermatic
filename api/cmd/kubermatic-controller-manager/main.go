package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-logr/zapr"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/collectors"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/migrations"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	"k8s.io/apimachinery/pkg/api/meta"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	controllerName = "kubermatic-controller-manager"
)

func main() {
	options, err := newControllerRunOptions()
	if err != nil {
		fmt.Printf("Failed to create controller run options due to = %v\n", err)
		os.Exit(1)
	}

	if err := options.validate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rawLog := kubermaticlog.New(options.log.Debug, kubermaticlog.Format(options.log.Format))
	log := rawLog.Sugar().With(
		"worker-name", options.workerName,
	)
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	config, err := clientcmd.BuildConfigFromFlags(options.masterURL, options.kubeconfig)
	if err != nil {
		log.Fatalw("Failed to create a kubernetes config", zap.Error(err))
	}

	// Set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.Log = ctrlruntimelog.NewDelegatingLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	// Create a manager, disable metrics as we have our own handler that exposes
	// the metrics of both the ctrltuntime registry and the default registry
	mgr, err := manager.New(config, manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		log.Fatalw("Failed to create the manager", zap.Error(err))
	}
	// Add all custom type schemes to our scheme. Otherwise we won't get a informer
	if err := autoscalingv1beta2.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", autoscalingv1beta2.SchemeGroupVersion), zap.Error(err))
	}
	if err := kubermaticv1.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalw("Failed to register scheme", zap.Stringer("api", kubermaticv1.SchemeGroupVersion), zap.Error(err))
	}

	recorder := mgr.GetRecorder(controllerName)

	// Check if the CRD for the VerticalPodAutoscaler is registered by allocating an informer
	if _, err := informer.GetSyncedStoreFromDynamicFactory(mgr.GetCache(), &autoscalingv1beta2.VerticalPodAutoscaler{}); err != nil {
		if _, crdNotRegistered := err.(*meta.NoKindMatchError); crdNotRegistered {
			log.Fatal(`
The VerticalPodAutoscaler is not installed in this seed cluster.
Please install the VerticalPodAutoscaler according to the documentation: https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler#installation`)
		}
	}

	//Register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_controller_manager", prometheus.DefaultRegisterer)

	dockerPullConfigJSON, err := ioutil.ReadFile(options.dockerPullConfigJSONFile)
	if err != nil {
		log.Fatalw(
			"Failed to read docker pull config file",
			zap.String("file", options.dockerPullConfigJSONFile),
			zap.Error(err),
		)
	}

	rootCtx, rootCancel := context.WithCancel(context.Background())
	seedGetter, err := provider.SeedGetterFactory(rootCtx, mgr.GetClient(), options.dc, options.dcFile, options.dynamicDatacenters)
	if err != nil {
		log.Fatalw("Unable to create the seed factory", zap.Error(err))
	}

	var clientProvider client.UserClusterConnectionProvider
	if options.kubeconfig != "" {
		clientProvider, err = client.NewExternal(mgr.GetClient())
	} else {
		clientProvider, err = client.NewInternal(mgr.GetClient())
	}
	if err != nil {
		log.Fatalw("Failed to get clientProvider", zap.Error(err))
	}

	ctrlCtx := &controllerContext{
		runOptions:           options,
		mgr:                  mgr,
		clientProvider:       clientProvider,
		seedGetter:           seedGetter,
		dockerPullConfigJSON: dockerPullConfigJSON,
		log:                  log,
	}

	if err := createAllControllers(ctrlCtx); err != nil {
		log.Fatalw("Could not create all controllers", zap.Error(err))
	}

	log.Debug("Starting clusters collector")
	collectors.MustRegisterClusterCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetClient())

	var g run.Group
	// This group is forever waiting in a goroutine for signals to stop
	{
		signalChan := signals.SetupSignalHandler()
		g.Add(func() error {
			select {
			case <-signalChan:
				log.Info("Received a signal to stop")
				return nil
			case <-rootCtx.Done():
				return nil
			}
		}, func(err error) {
			rootCancel()
		})
	}

	// This group is running an internal http server with metrics and other debug information
	{
		metricsServerCtx, stopMetricsServer := context.WithCancel(rootCtx)
		defer stopMetricsServer()
		m := &metricsServer{
			gatherers: []prometheus.Gatherer{
				prometheus.DefaultGatherer, ctrlruntimemetrics.Registry},
			listenAddress: options.internalAddr,
		}

		g.Add(func() error {
			log.Infof("Starting the internal HTTP server: %s\n", options.internalAddr)
			return m.Start(metricsServerCtx.Done())
		}, func(err error) {
			stopMetricsServer()
		})
	}

	// This group is running the actual controller logic
	{
		leaderCtx, stopLeaderElection := context.WithCancel(rootCtx)
		defer stopLeaderElection()
		g.Add(func() error {
			leaderElectionClient, err := kubernetes.NewForConfig(rest.AddUserAgent(config, "kubermatic-controller-manager-leader-election"))
			if err != nil {
				return err
			}
			callbacks := kubeleaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					log.Info("Acquired the leader lease")

					log.Info("Executing migrations...")
					if err := migrations.RunAll(ctrlCtx.mgr.GetConfig(), ctrlCtx.runOptions.workerName); err != nil {
						log.Errorf("failed to run migrations: %v", err)
						stopLeaderElection()
						return
					}
					log.Info("Migrations executed successfully")

					log.Info("Starting the controller-manager...")
					if err := mgr.Start(ctx.Done()); err != nil {
						log.Errorf("The controller-manager stopped with an error: %v", err)
						stopLeaderElection()
					}
				},
				OnStoppedLeading: func() {
					// Gets called when we could not renew the lease or the parent context was closed
					log.Info("Shutting down the controller-manager...")
					stopLeaderElection()
				},
			}

			leaderName := controllerName
			if options.workerName != "" {
				leaderName = options.workerName + "-" + leaderName
			}
			leader, err := leaderelection.New(leaderName, leaderElectionClient, recorder, callbacks)
			if err != nil {
				return fmt.Errorf("failed to create a leaderelection: %v", err)
			}

			leader.Run(leaderCtx)
			return nil
		}, func(err error) {
			stopLeaderElection()
		})
	}

	if err := g.Run(); err != nil {
		// Set the error as field so we have a consistent way of logging errors
		log.Fatalw("Shutting down with error", zap.Error(err))
	}
}
