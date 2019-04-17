package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

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
		glog.Fatalf("failed to create controller run options due to = %v", err)
	}

	if err := options.validate(); err != nil {
		glog.Fatal(err)
	}
	log := kubermaticlog.New(options.log.debug, kubermaticlog.Format(options.log.format))

	config, err := clientcmd.BuildConfigFromFlags(options.masterURL, options.kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	// Set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(log)

	// Create a manager, disable metrics as we have our own handler that exposes
	// the metrics of both the ctrltuntime registry and the default registry
	mgr, err := manager.New(config, manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		glog.Fatalf("failed to create mgr: %v", err)
	}
	// Add all custom type schemes to our scheme. Otherwise we won't get a informer
	if err := autoscalingv1beta2.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("failed to add the autoscaling.k8s.io scheme to mgr: %v", err)
	}
	if err := kubermaticv1.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("failed to add kubermatic scheme to mgr: %v", err)
	}

	recorder := mgr.GetRecorder(controllerName)

	// Check if the CRD for the VerticalPodAutoscaler is registered by allocating an informer
	if _, err := informer.GetSyncedStoreFromDynamicFactory(mgr.GetCache(), &autoscalingv1beta2.VerticalPodAutoscaler{}); err != nil {
		if _, crdNotRegistered := err.(*meta.NoKindMatchError); crdNotRegistered {
			glog.Fatal(`
The VerticalPodAutoscaler is not installed in this seed cluster.
Please install the VerticalPodAutoscaler according to the documentation: https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler#installation`)
		}
	}

	//Register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_controller_manager", prometheus.DefaultRegisterer)

	dockerPullConfigJSON, err := ioutil.ReadFile(options.dockerPullConfigJSONFile)
	if err != nil {
		glog.Fatalf("Failed to read dockerPullConfigJSON file %q: %v", options.dockerPullConfigJSONFile, err)
	}

	ctrlCtx, err := newControllerContext(options, mgr, log)
	if err != nil {
		glog.Fatal(err)
	}
	ctrlCtx.dockerPullConfigJSON = dockerPullConfigJSON

	if err := createAllControllers(ctrlCtx); err != nil {
		glog.Fatalf("could not create all controllers: %v", err)
	}

	glog.V(4).Info("Starting clusters collector")
	collectors.MustRegisterClusterCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetClient())

	var g run.Group
	// This group is forever waiting in a goroutine for signals to stop
	{
		signalCtx, stopWaitingForSignal := context.WithCancel(context.Background())
		defer stopWaitingForSignal()
		signalChan := signals.SetupSignalHandler()
		g.Add(func() error {
			select {
			case <-signalChan:
				glog.Info("Received a signal to stop")
				return nil
			case <-signalCtx.Done():
				return nil
			}
		}, func(err error) {
			stopWaitingForSignal()
		})
	}

	// This group is running an internal http server with metrics and other debug information
	{
		metricsServerCtx, stopMetricsServer := context.WithCancel(context.Background())
		defer stopMetricsServer()
		m := &metricsServer{
			gatherers: []prometheus.Gatherer{
				prometheus.DefaultGatherer, ctrlruntimemetrics.Registry},
			listenAddress: options.internalAddr,
		}

		g.Add(func() error {
			glog.Infof("Starting the internal HTTP server: %s\n", options.internalAddr)
			return m.Start(metricsServerCtx.Done())
		}, func(err error) {
			stopMetricsServer()
		})
	}

	// This group is running the actual controller logic
	{
		leaderCtx, stopLeaderElection := context.WithCancel(context.Background())
		defer stopLeaderElection()
		g.Add(func() error {
			leaderElectionClient, err := kubernetes.NewForConfig(rest.AddUserAgent(config, "kubermatic-controller-manager-leader-election"))
			if err != nil {
				return err
			}
			callbacks := kubeleaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					glog.Info("Acquired the leader lease")

					glog.Info("Executing migrations...")
					if err := migrations.RunAll(ctrlCtx.mgr.GetConfig(), ctrlCtx.runOptions.workerName); err != nil {
						glog.Errorf("failed to run migrations: %v", err)
						stopLeaderElection()
						return
					}
					glog.Info("Migrations executed successfully")

					glog.Info("Starting the controller-manager...")
					if err := mgr.Start(ctx.Done()); err != nil {
						glog.Errorf("The controller-manager stopped with an error: %v", err)
						stopLeaderElection()
					}
				},
				OnStoppedLeading: func() {
					// Gets called when we could not renew the lease or the parent context was closed
					glog.Info("Shutting down the controller-manager...")
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
		glog.Fatal(err)
	}
}

func newControllerContext(
	runOp controllerRunOptions,
	mgr manager.Manager,
	log logr.Logger,
) (*controllerContext, error) {
	ctrlCtx := &controllerContext{
		mgr:        mgr,
		runOptions: runOp,
		log:        log,
	}

	var err error
	ctrlCtx.dcs, err = provider.LoadDatacentersMeta(ctrlCtx.runOptions.dcFile)
	if err != nil {
		return nil, err
	}

	var clientProvider client.UserClusterConnectionProvider
	if ctrlCtx.runOptions.kubeconfig != "" {
		clientProvider, err = client.NewExternal(mgr.GetClient())
		if err != nil {
			return nil, fmt.Errorf("failed to get clientProvider: %v", err)
		}
	} else {
		clientProvider, err = client.NewInternal(mgr.GetClient())
		if err != nil {
			return nil, fmt.Errorf("failed to get clientProvider: %v", err)
		}
	}
	ctrlCtx.clientProvider = clientProvider

	return ctrlCtx, nil
}
