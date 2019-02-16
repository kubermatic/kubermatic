package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kubermatic/kubermatic/api/pkg/collectors"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	autoscalingv1beta1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1"
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

	kubeClient := kubernetes.NewForConfigOrDie(config)
	kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)
	recorder, err := getEventRecorder(kubeClient)
	if err != nil {
		glog.Fatalf("failed to get event recorder: %v", err)
	}

	// Create a manager
	mgr, err := manager.New(config, manager.Options{})
	if err != nil {
		glog.Fatalf("failed to create mgr: %v", err)
	}
	// Add all custom type schemes to our scheme. Otherwise we won't get a informer
	if err := autoscalingv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("failed to add the autoscaling.k8s.io scheme to mgr: %v", err)
	}
	if err := kubermaticv1.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("failed to add kubermatic scheme to mgr: %v", err)
	}

	dynamicClient := mgr.GetClient()
	dynamicCache := mgr.GetCache()

	// Check if the CRD for the VerticalPodAutoscaler is registered by allocating an informer
	if _, err := informer.GetSyncedStoreFromDynamicFactory(dynamicCache, &autoscalingv1beta1.VerticalPodAutoscaler{}); err != nil {
		if _, crdNotRegistered := err.(*meta.NoKindMatchError); crdNotRegistered {
			glog.Fatal(`
The VerticalPodAutoscaler is not installed in this seed cluster.
Please install the VerticalPodAutoscaler according to the documentation: https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler#installation`)
		}
	}

	//Register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_controller_manager", prometheus.DefaultRegisterer)

	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	done := ctx.Done()
	go func() {
		if err := dynamicCache.Start(done); err != nil {
			glog.Fatal("failed to start the dynamic lister")
		}
	}()

	ctrlCtx, err := newControllerContext(options, mgr, done, kubeClient, kubermaticClient, dynamicClient, dynamicCache)
	if err != nil {
		glog.Fatal(err)
	}

	controllers, err := createAllControllers(ctrlCtx)
	if err != nil {
		glog.Fatalf("could not create all controllers: %v", err)
	}

	for name, register := range collectors.AvailableCollectors {
		glog.V(6).Infof("Starting %s collector", name)
		register(prometheus.DefaultRegisterer, ctrlCtx.kubeInformerFactory, ctrlCtx.kubermaticInformerFactory)
	}

	// Start context (Informers)
	ctrlCtx.Start()

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
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.Handler())

		s := http.Server{
			Addr:         options.internalAddr,
			Handler:      m,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		g.Add(func() error {
			glog.Infof("Starting the internal http server: %s\n", options.internalAddr)
			err := s.ListenAndServe()
			if err != nil {
				return fmt.Errorf("internal http server failed: %v", err)
			}
			return nil
		}, func(err error) {
			glog.Errorf("Stopping internal http server: %v", err)
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			glog.Info("Shutting down the internal http server")
			if err := s.Shutdown(timeoutCtx); err != nil {
				glog.Error("failed to shutdown the internal http server gracefully:", err)
			}
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
				OnStartedLeading: func(stop <-chan struct{}) {
					if err = runAllControllers(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh, ctxDone, ctrlCtx.mgr, controllers); err != nil {
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

			go leader.Run()
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

func getEventRecorder(masterKubeClient *kubernetes.Clientset) (record.EventRecorder, error) {
	// Create event broadcaster
	// Add kubermatic types to the default Kubernetes Scheme so Events can be
	// logged properly
	if err := kubermaticv1.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.V(4).Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: masterKubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName})
	return recorder, nil
}

func newControllerContext(
	runOp controllerRunOptions,
	mgr manager.Manager,
	done <-chan struct{},
	kubeClient kubernetes.Interface,
	kubermaticClient kubermaticclientset.Interface,
	dynamicClient ctrlruntimeclient.Client,
	dynamicCache cache.Cache) (*controllerContext, error) {
	ctrlCtx := &controllerContext{
		mgr:              mgr,
		runOptions:       runOp,
		stopCh:           done,
		kubeClient:       kubeClient,
		kubermaticClient: kubermaticClient,
		dynamicClient:    dynamicClient,
		dynamicCache:     dynamicCache,
	}

	selector, err := workerlabel.LabelSelector(runOp.workerName)
	if err != nil {
		return nil, err
	}

	ctrlCtx.kubermaticInformerFactory = kubermaticinformers.NewFilteredSharedInformerFactory(ctrlCtx.kubermaticClient, informer.DefaultInformerResyncPeriod, metav1.NamespaceAll, selector)
	ctrlCtx.kubeInformerFactory = kubeinformers.NewSharedInformerFactory(ctrlCtx.kubeClient, informer.DefaultInformerResyncPeriod)

	return ctrlCtx, nil
}

func (ctx *controllerContext) Start() {
	ctx.kubermaticInformerFactory.Start(ctx.stopCh)
	ctx.kubeInformerFactory.Start(ctx.stopCh)

	ctx.kubermaticInformerFactory.WaitForCacheSync(ctx.stopCh)
	ctx.kubeInformerFactory.WaitForCacheSync(ctx.stopCh)
}
