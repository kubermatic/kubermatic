package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/collectors"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
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

	// Enable logging
	log.SetLogger(log.ZapLogger(false))

	mgrOptions := manager.Options{
		// Disable metrics as we have our own handler that exposes
		// the metrics of both the ctrltuntime registry and the default registry
		MetricsBindAddress: "0",
		LeaderElection:     true,
		LeaderElectionID:   controllerName,
	}
	if options.kubeconfig != "" {
		// Use default if we run outside of a cluster.
		// If running in-cluster, the leaderelection takes the namespace from /var/run/secrets/kubernetes.io/serviceaccount/namespace
		// That exists in every pod
		mgrOptions.LeaderElectionNamespace = metav1.NamespaceDefault
	}
	if options.workerName != "" {
		mgrOptions.LeaderElectionID = controllerName + "-" + options.workerName
	}

	mgr, err := manager.New(config, mgrOptions)
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

	signalChan := signals.SetupSignalHandler()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	ctrlCtx, err := newControllerContext(options, mgr, ctx.Done())
	if err != nil {
		glog.Fatal(err)
	}
	ctrlCtx.dockerPullConfigJSON = dockerPullConfigJSON

	if err := registerAllControllers(ctrlCtx); err != nil {
		glog.Fatalf("could not create all controllers: %v", err)
	}

	glog.V(4).Info("Starting clusters collector")
	collectors.MustRegisterClusterCollector(prometheus.DefaultRegisterer, ctrlCtx.mgr.GetClient())

	var g run.Group

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-signalChan:
				glog.Info("Received signal to stop the application")
				// Cancel main context
				cancelCtx()
			case <-ctx.Done():
			}
			return nil
		}, func(err error) {
			// We don't need to do anything here as the routine gets stopped either by a signal or by the ctx
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
			return m.Start(ctx.Done())
		}, func(err error) {
			// No-Op because the metrics server gets the stopchannel and stops
			// once it is closed
		})
	}

	// This group starts the controller manager
	{
		g.Add(func() error {
			return mgr.Start(ctx.Done())
		}, func(err error) {
			// We don't need to do anything here as the routine gets stopped by the ctx
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
