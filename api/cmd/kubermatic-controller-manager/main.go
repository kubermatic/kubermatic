package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/util/informer"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kubermatic/kubermatic/api/pkg/collectors"
	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/net"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeleaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
)

type controllerRunOptions struct {
	kubeconfig   string
	masterURL    string
	internalAddr string

	masterResources                                  string
	externalURL                                      string
	dc                                               string
	dcFile                                           string
	workerName                                       string
	versionsFile                                     string
	updatesFile                                      string
	workerCount                                      int
	overwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	addonsPath                                       string
	addonsList                                       string
	backupContainerFile                              string
	cleanupContainerFile                             string
	backupContainerImage                             string
	backupInterval                                   string
	etcdDiskSize                                     string
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	monitoringScrapeAnnotationPrefix                 string
	dockerPullConfigJSONFile                         string
}

type controllerContext struct {
	runOptions                controllerRunOptions
	stopCh                    <-chan struct{}
	kubeClient                kubernetes.Interface
	kubermaticClient          kubermaticclientset.Interface
	kubermaticInformerFactory kubermaticinformers.SharedInformerFactory
	kubeInformerFactory       kubeinformers.SharedInformerFactory
}

const (
	controllerName = "kubermatic-controller-manager"
)

func main() {
	runOp := controllerRunOptions{}
	flag.StringVar(&runOp.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&runOp.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&runOp.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal server is running on")
	flag.StringVar(&runOp.masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&runOp.externalURL, "external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	flag.StringVar(&runOp.dc, "datacenter-name", "", "The name of the seed datacenter, the controller is running in. It will be used to build the absolute url for a customer cluster.")
	flag.StringVar(&runOp.dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&runOp.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.StringVar(&runOp.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&runOp.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.IntVar(&runOp.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&runOp.overwriteRegistry, "overwrite-registry", "", "registry to use for all images")
	flag.StringVar(&runOp.nodePortRange, "nodeport-range", "30000-32767", "NodePort range to use for new clusters. It must be within the NodePort range of the seed-cluster")
	flag.StringVar(&runOp.nodeAccessNetwork, "node-access-network", "10.254.0.0/16", "A network which allows direct access to nodes via VPN. Uses CIDR notation.")
	flag.StringVar(&runOp.addonsPath, "addons-path", "/opt/addons", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&runOp.addonsList, "addons-list", "canal,dashboard,dns,kube-proxy,openvpn,rbac,kubelet-configmap,default-storage-class,metrics-server", "Comma separated list of Addons to install into every user-cluster")
	flag.StringVar(&runOp.backupContainerFile, "backup-container", "", fmt.Sprintf("[Required] Filepath of a backup container yaml. It must mount a volume named %s from which it reads the etcd backups", backupcontroller.SharedVolumeName))
	flag.StringVar(&runOp.cleanupContainerFile, "cleanup-container", "", "[Required] Filepath of a cleanup container yaml. The container will be used to cleanup the backup directory for a cluster after it got deleted.")
	flag.StringVar(&runOp.backupContainerImage, "backup-container-init-image", backupcontroller.DefaultBackupContainerImage, "Docker image to use for the init container in the backup job, must be an etcd v3 image. Only set this if your cluster can not use the public quay.io registry")
	flag.StringVar(&runOp.backupInterval, "backup-interval", backupcontroller.DefaultBackupInterval, "Interval in which the etcd gets backed up")
	flag.StringVar(&runOp.etcdDiskSize, "etcd-disk-size", "5Gi", "Size for the etcd PV's. Only applies to new clusters.")
	flag.StringVar(&runOp.inClusterPrometheusRulesFile, "in-cluster-prometheus-rules-file", "", "The file containing the custom alerting rules for the prometheus running in the cluster-foo namespaces.")
	flag.BoolVar(&runOp.inClusterPrometheusDisableDefaultRules, "in-cluster-prometheus-disable-default-rules", false, "A flag indicating whether the default rules for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&runOp.dockerPullConfigJSONFile, "docker-pull-config-json-file", "config.json", "The file containing the docker auth config.")
	flag.BoolVar(&runOp.inClusterPrometheusDisableDefaultScrapingConfigs, "in-cluster-prometheus-disable-default-scraping-configs", false, "A flag indicating whether the default scraping configs for the prometheus running in the cluster-foo namespaces should be deployed.")
	flag.StringVar(&runOp.inClusterPrometheusScrapingConfigsFile, "in-cluster-prometheus-scraping-configs-file", "", "The file containing the custom scraping configs for the prometheus running in the cluster-foo namespaces.")
	flag.StringVar(&runOp.monitoringScrapeAnnotationPrefix, "monitoring-scrape-annotation-prefix", "monitoring.kubermatic.io", "The prefix for monitoring annotations in the user cluster. Default: monitoring.kubermatic.io -> monitoring.kubermatic.io/scrape, monitoring.kubermatic.io/path")
	flag.Parse()

	if runOp.masterResources == "" {
		glog.Fatal("master-resources path is undefined\n\n")
	}

	if runOp.externalURL == "" {
		glog.Fatal("external-url is undefined\n\n")
	}

	if runOp.dc == "" {
		glog.Fatal("datacenter-name is undefined")
	}

	if runOp.backupContainerFile == "" {
		glog.Fatal("backup-container is undefined")
	}

	if runOp.dockerPullConfigJSONFile == "" {
		glog.Fatal("docker-pull-config-json-file is undefined")
	}

	if runOp.monitoringScrapeAnnotationPrefix == "" {
		glog.Fatal("moniotring-scrape-annotation-prefix is undefined")
	}

	// Validate etcd disk size
	resource.MustParse(runOp.etcdDiskSize)

	// Validate node-port range
	net.ParsePortRangeOrDie(runOp.nodePortRange)

	// dcFile, versionFile, updatesFile are required by cluster controller
	// the following code ensures that the files are available and fails fast if not.
	_, err := provider.LoadDatacentersMeta(runOp.dcFile)
	if err != nil {
		glog.Fatalf("failed to load datacenter yaml %q: %v", runOp.dcFile, err)
	}

	config, err := clientcmd.BuildConfigFromFlags(runOp.masterURL, runOp.kubeconfig)
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

	//Register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_controller_manager", prometheus.DefaultRegisterer)

	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	ctrlCtx, err := newControllerContext(runOp, ctx.Done(), kubeClient, kubermaticClient)
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
			Addr:         runOp.internalAddr,
			Handler:      m,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		g.Add(func() error {
			glog.Infof("Starting the internal http server: %s\n", runOp.internalAddr)
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
			leaderElectionClient, err := kubernetes.NewForConfig(restclient.AddUserAgent(config, "kubermatic-controller-manager-leader-election"))
			if err != nil {
				return err
			}
			callbacks := kubeleaderelection.LeaderCallbacks{
				OnStartedLeading: func(stop <-chan struct{}) {
					if err = runAllControllers(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh, ctxDone, controllers); err != nil {
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
			if runOp.workerName != "" {
				leaderName = runOp.workerName + "-" + leaderName
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

func newControllerContext(runOp controllerRunOptions, done <-chan struct{}, kubeClient kubernetes.Interface, kubermaticClient kubermaticclientset.Interface) (*controllerContext, error) {
	ctrlCtx := &controllerContext{
		runOptions:       runOp,
		stopCh:           done,
		kubeClient:       kubeClient,
		kubermaticClient: kubermaticClient,
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
