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
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/seed"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/bringyourown"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/digitalocean"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/fake"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/api/core/v1"
	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
)

var (
	kubeconfig     string
	masterURL      string
	prometheusAddr string

	masterResources string
	externalURL     string
	dc              string
	dcFile          string
	workerName      string
	versionsFile    string
	updatesFile     string
	workerCount     int
)

const (
	controllerName = "kubermatic-cluster-controller"
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&prometheusAddr, "prometheus-address", "127.0.0.1:8085", "The Address on which the prometheus handler should be exposed")
	flag.StringVar(&masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&externalURL, "external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	flag.StringVar(&dc, "datacenter-name", "", "The name of the seed datacenter, the controller is running in. It will be used to build the absolute url for a customer cluster.")
	flag.StringVar(&dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	flag.StringVar(&versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.IntVar(&workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.Parse()

	if masterResources == "" {
		glog.Fatal("master-resources path is undefined\n\n")
	}

	if externalURL == "" {
		glog.Fatal("external-url is undefined\n\n")
	}

	if dc == "" {
		glog.Fatal("datacenter-name is undefined")
	}

	dcs, err := provider.LoadDatacentersMeta(dcFile)
	if err != nil {
		glog.Fatalf("failed to load datacenter yaml %q: %v", dcFile, err)
	}

	// load versions
	versions, err := version.LoadVersions(versionsFile)
	if err != nil {
		glog.Fatalf("failed to load version yaml %q: %v", versionsFile, err)
	}

	// load updates
	updates, err := version.LoadUpdates(updatesFile)
	if err != nil {
		glog.Fatalf("failed to load version yaml %q: %v", versionsFile, err)
	}

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	var g run.Group

	kubeClient := kubernetes.NewForConfigOrDie(config)
	kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)
	recorder := getEventRecorder(kubeClient)

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
		m.Handle("/metrics", promhttp.Handler())

		s := http.Server{
			Addr:         prometheusAddr,
			Handler:      m,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		g.Add(func() error {
			glog.Infof("Starting the internal http server: %s\n", prometheusAddr)
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
			if workerName != "" {
				leaderName = workerName + "-" + leaderName
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
	kubermaticv1.AddToScheme(scheme.Scheme)
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
		Clusters:      metrics.Clusters,
		ClusterPhases: metrics.ClusterPhases,
		Workers:       metrics.Workers,
	}

	kubermaticInformerFactory := externalversions.NewSharedInformerFactory(kubermaticClient, time.Minute*5)
	kubeInformerFactory := kuberinformers.NewSharedInformerFactory(kubeClient, time.Minute*5)

	cps := map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(dcs),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(dcs),
		provider.OpenstackCloudProvider:    openstack.NewCloudProvider(dcs),
	}

	ctrl, err := cluster.NewController(
		kubeClient,
		kubermaticClient,
		versions,
		updates,
		masterResources,
		externalURL,
		workerName,
		dc,
		dcs,
		cps,
		clusterMetrics,

		kubermaticInformerFactory.Kubermatic().V1().Clusters(),
		kubermaticInformerFactory.Etcd().V1beta2().EtcdClusters(),
		kubeInformerFactory.Core().V1().Namespaces(),
		kubeInformerFactory.Core().V1().Secrets(),
		kubeInformerFactory.Core().V1().Services(),
		kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
		kubeInformerFactory.Core().V1().ConfigMaps(),
		kubeInformerFactory.Core().V1().ServiceAccounts(),
		kubeInformerFactory.Extensions().V1beta1().Deployments(),
		kubeInformerFactory.Extensions().V1beta1().Ingresses(),
		kubeInformerFactory.Rbac().V1beta1().ClusterRoleBindings(),
	)
	if err != nil {
		return err
	}

	kubermaticInformerFactory.Start(stop)
	kubeInformerFactory.Start(stop)

	kubermaticInformerFactory.WaitForCacheSync(stop)
	kubeInformerFactory.WaitForCacheSync(stop)

	glog.Info("Starting controller")
	ctrl.Run(workerCount, stop)

	return nil
}
