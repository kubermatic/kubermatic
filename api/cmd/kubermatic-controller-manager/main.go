package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/leaderelection"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/net"
	kuberinformers "k8s.io/client-go/informers"
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

	masterResources      string
	externalURL          string
	dc                   string
	dcFile               string
	workerName           string
	versionsFile         string
	updatesFile          string
	workerCount          int
	overwriteRegistry    string
	nodePortRange        string
	addons               string
	backupContainerFile  string
	backupContainerImage string
	backupInterval       string
}

type controllerContext struct {
	runOptions                controllerRunOptions
	stopCh                    <-chan struct{}
	kubeClient                kubernetes.Interface
	kubermaticClient          kubermaticclientset.Interface
	kubermaticInformerFactory externalversions.SharedInformerFactory
	kubeInformerFactory       kuberinformers.SharedInformerFactory
	seedClustersRESTClient    []kubernetes.Interface
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
	flag.StringVar(&runOp.addons, "addons", "/opt/addons", "Path to addon manifests. Should contain sub-folders for each addon")
	flag.StringVar(&runOp.backupContainerFile, "backup-container", "", fmt.Sprintf("[Required] Filepath of a backupContainer yaml. It must mount a volume named %s from which it reads the etcd backups", backupcontroller.SharedVolumeName))
	flag.StringVar(&runOp.backupContainerImage, "backup-container-init-image", backupcontroller.DefaultBackupContainerImage, "Docker image to use for the init container in the backup job, must be an etcd v3 image. Only set this if your cluster can not use the public quay.io registry")
	flag.StringVar(&runOp.backupInterval, "backup-interval", backupcontroller.DefaultBackupInterval, "Interval in which the etcd gets backed up")
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

	// create a REST Client for each seed cluster we find in kubeconfig
	seedClustersRESTClient := []kubernetes.Interface{}
	{
		clientcmdConfig, err := clientcmd.LoadFromFile(runOp.kubeconfig)
		if err != nil {
			glog.Fatal(err)
		}

		for ctx := range clientcmdConfig.Contexts {
			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*clientcmdConfig,
				ctx,
				&clientcmd.ConfigOverrides{CurrentContext: ctx},
				nil,
			)
			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				glog.Fatal(err)
			}
			kubeClient, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				glog.Fatal(err)
			}
			if cfg.Host == config.Host && cfg.Username == config.Username && cfg.Password == config.Password {
				glog.V(2).Infof("Skipping adding %s as a seed cluster. It is exactly the same as existing kubeClient", ctx)
				continue
			}
			glog.V(2).Infof("Adding %s as seed cluster", ctx)
			seedClustersRESTClient = append(seedClustersRESTClient, kubeClient)
		}
	}

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
			leaderElectionClient, err := kubernetes.NewForConfig(restclient.AddUserAgent(config, "kubermatic-controller-manager-leader-election"))
			if err != nil {
				return err
			}
			callbacks := kubeleaderelection.LeaderCallbacks{
				OnStartedLeading: func(stop <-chan struct{}) {
					ctrlCtx := controllerContext{runOptions: runOp, stopCh: ctx.Done(), kubeClient: kubeClient, kubermaticClient: kubermaticClient, seedClustersRESTClient: seedClustersRESTClient}
					err := runAllControllers(ctrlCtx)
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
