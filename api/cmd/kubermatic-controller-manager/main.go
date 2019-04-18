package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/collectors"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/migrations"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"

	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
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

	config, err := clientcmd.BuildConfigFromFlags(options.masterURL, options.kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	var g run.Group

	// Enable logging
	log.SetLogger(log.ZapLogger(false))

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

	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	done := ctx.Done()

	ctrlCtx, err := newControllerContext(options, mgr, done)
	if err != nil {
		glog.Fatal(err)
	}
	ctrlCtx.dockerPullConfigJSON = dockerPullConfigJSON

	if err := createAllControllers(ctrlCtx); err != nil {
		glog.Fatalf("could not create all controllers: %v", err)
	}

	glog.V(6).Info("Starting clusters collector")
	collectors.MustRegisterClusterCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetClient())

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("user requested to stop the application")
			case <-done:
				return errors.New("parent context has been closed - propagating the request")
			}
		}, func(err error) {
			ctxDone()
		})
	}

	// This group is running an internal http server with metrics and other debug information
	{
		m := &metricsServer{
			gatherers: []prometheus.Gatherer{
				prometheus.DefaultGatherer, ctrlruntimemetrics.Registry},
			listenAddress: options.internalAddr,
		}

		g.Add(func() error {
			glog.Infof("Starting the internal http server: %s\n", options.internalAddr)
			return m.Start(stopCh)
		}, func(err error) {
			// No-Op because the metrics server gets the stopchannel and stops
			// once it is closed
		})
	}

	// This group is running the actual controller logic
	{
		g.Add(func() error {
			leaderElectionClient, err := kubernetes.NewForConfig(rest.AddUserAgent(config, "kubermatic-controller-manager-leader-election"))
			if err != nil {
				return err
			}
			callbacks := kubeleaderelection.LeaderCallbacks{
				OnStartedLeading: func(_ context.Context) {
					if err := migrations.RunAll(ctrlCtx.mgr.GetConfig(), ctrlCtx.runOptions.workerName); err != nil {
						glog.Errorf("failed to run migrations: %v", err)
						ctxDone()
					}
					if err = runAllControllers(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh, ctxDone, ctrlCtx.mgr); err != nil {
						glog.Error(err)
						ctxDone()
					}
				},
				OnStoppedLeading: func() {
					glog.Error("==================== OnStoppedLeading ====================")
					ctxDone()
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

			go leader.Run(ctx)
			<-done
			return nil
		}, func(err error) {
			glog.Errorf("Stopping controller: %v", err)
			ctxDone()
		})
	}

	if err := g.Run(); err != nil {
		glog.Fatal(err)
	}
}

func newControllerContext(
	runOp controllerRunOptions,
	mgr manager.Manager,
	done <-chan struct{}) (*controllerContext, error) {
	ctrlCtx := &controllerContext{
		mgr:        mgr,
		runOptions: runOp,
		stopCh:     done,
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
