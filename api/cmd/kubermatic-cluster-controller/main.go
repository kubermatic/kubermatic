package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	mastercrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	initscheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned/scheme"
	masterinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/master"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/seed"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
)

var (
	prometheusAddr        = flag.String("prometheus-address", "127.0.0.1:8085", "The Address on which the prometheus handler should be exposed")
	prometheusPath        = flag.String("prometheus-path", "/metrics", "The path on the host, on which the handler is available")
	kubeConfig            = flag.String("kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	masterKubeconfig      = flag.String("master-kubeconfig", "", "When set it will overwrite the usage of the InClusterConfig")
	masterResources       = flag.String("master-resources", "", "The master resources path (Required).")
	externalURL           = flag.String("external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	dcFile                = flag.String("datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	workerName            = flag.String("worker-name", "", "Create clusters only processed by worker-name cluster controller")
	versionsFile          = flag.String("versions", "versions.yaml", "The versions.yaml file path")
	updatesFile           = flag.String("updates", "updates.yaml", "The updates.yaml file path")
	apiserverExternalPort = flag.Int("apiserver-external-port", 8443, "Port on which the apiserver of a client cluster should be reachable")
	workerCount           = flag.Int("worker-count", 4, "Number of workers which process the clusters in parallel.")
)

const (
	controllerName = "kubermatic-cluster-controller"
)

func main() {
	flag.Parse()

	if *masterResources == "" {
		glog.Fatal("master-resources path is undefined\n\n")
	}

	if *externalURL == "" {
		glog.Fatal("external-url is undefined\n\n")
	}

	dcs, err := provider.LoadDatacentersMeta(*dcFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load datacenter yaml %q: %v", *dcFile, err))
	}

	// load versions
	versions, err := version.LoadVersions(*versionsFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load version yaml %q: %v", *versionsFile, err))
	}

	// load updates
	updates, err := version.LoadUpdates(*updatesFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load version yaml %q: %v", *versionsFile, err))
	}

	seedCmdConfig, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		glog.Fatal(err)
	}

	var masterConfig *restclient.Config
	masterConfig, err = clientcmd.BuildConfigFromFlags("", *masterKubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	var g run.Group

	// Create a Kubernetes api client
	masterKubeClient := kubernetes.NewForConfigOrDie(masterConfig)
	// Create event recorder
	recorder := getEventRecorder(masterKubeClient)

	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("user requested to stop the application")
			case <-ctx.Done():
				return errors.New("parent context has been closed - propagating the request")
			}
		}, func(err error) {
			ctxDone()
		})
	}

	// This group is running an internal http server with metrics and other debug information
	{
		m := http.NewServeMux()
		m.Handle(*prometheusPath, promhttp.Handler())

		s := http.Server{
			Addr:         *prometheusAddr,
			Handler:      m,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		g.Add(func() error {
			glog.Infof("Starting the internal http server: %s\n", *prometheusAddr)
			err := s.ListenAndServe()
			if err != nil {
				return fmt.Errorf("internal http server failed: %v", err)
			}
			return nil
		}, func(err error) {
			glog.Errorf("Stopping internal http server: %v", err)
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			glog.Info("Shutting down the internal http server")
			if err := s.Shutdown(ctx); err != nil {
				glog.Error("failed to shutdown the internal http server gracefully:", err)
			}
		})
	}

	// This group is running the actual controller logic
	{
		g.Add(func() error {
			leaderElectionClient, err := kubernetes.NewForConfig(restclient.AddUserAgent(masterConfig, "kubermatic-cluster-controller-leader-election"))
			if err != nil {
				return err
			}
			callbacks := kubeleaderelection.LeaderCallbacks{
				OnStartedLeading: func(stop <-chan struct{}) {
					err := startController(ctx.Done(), dcs, masterConfig, seedCmdConfig, versions, updates)
					if err != nil {
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
			if *workerName != "" {
				leaderName = *workerName + "-" + leaderName
			}
			leader, err := leaderelection.New(leaderName, leaderElectionClient, recorder, callbacks)
			if err != nil {
				return fmt.Errorf("failed to create a leaderelection: %v", err)
			}

			go leader.Run()
			<-ctx.Done()
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
func getEventRecorder(masterKubeClient *kubernetes.Clientset) record.EventRecorder {
	// Create event broadcaster
	// Add kubermatic types to the default Kubernetes Scheme so Events can be
	// logged properly
	initscheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.V(4).Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: masterKubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: controllerName})
	return recorder
}

func startController(stop <-chan struct{}, dcs map[string]provider.DatacenterMeta, masterConfig *restclient.Config, seedConfig *clientcmdapi.Config, versions map[string]*apiv1.MasterVersion, updates []apiv1.MasterUpdate) error {
	metrics := NewClusterControllerMetrics()
	clusterMetrics := cluster.ControllerMetrics{
		Clusters: metrics.Clusters,
		Workers:  metrics.Workers,
	}

	masterCrdClient := mastercrdclient.NewForConfigOrDie(restclient.AddUserAgent(masterConfig, controllerName))
	masterInformerGroup := masterinformer.New(masterCrdClient)

	seedProvider, err := seed.NewFromConfig(seedConfig)
	if err != nil {
		return err
	}

	cps := cloud.Providers(dcs)
	ctrl, err := cluster.NewController(
		seedProvider,
		masterCrdClient,
		cps,
		versions,
		updates,
		*masterResources,
		*externalURL,
		*workerName,
		*apiserverExternalPort,
		dcs,
		masterInformerGroup,
		clusterMetrics,
	)
	if err != nil {
		return err
	}

	masterInformerGroup.Run(stop)
	cache.WaitForCacheSync(stop, masterInformerGroup.HasSynced)

	glog.Info("Starting controller")
	ctrl.Run(*workerCount, stop)

	return nil
}
