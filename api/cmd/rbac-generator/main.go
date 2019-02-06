package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/oklog/run"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	rbaccontroller "github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/signals"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type controllerRunOptions struct {
	kubeconfig   string
	masterURL    string
	internalAddr string

	workerName  string
	workerCount int
}

type controllerContext struct {
	runOptions                      controllerRunOptions
	stopCh                          <-chan struct{}
	kubeMasterClient                kubernetes.Interface
	kubermaticMasterClient          kubermaticclientset.Interface
	kubermaticMasterInformerFactory externalversions.SharedInformerFactory
	kubeMasterInformerFactory       kuberinformers.SharedInformerFactory
	allClusterProviders             []*rbaccontroller.ClusterProvider
}

func main() {
	var g run.Group
	ctrlCtx := controllerContext{}
	flag.StringVar(&ctrlCtx.runOptions.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&ctrlCtx.runOptions.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&ctrlCtx.runOptions.workerName, "worker-name", "", "The name of the worker that will only processes resources with label=worker-name.")
	flag.IntVar(&ctrlCtx.runOptions.workerCount, "worker-count", 4, "Number of workers which process the clusters in parallel.")
	flag.StringVar(&ctrlCtx.runOptions.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal server is running on")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(ctrlCtx.runOptions.masterURL, ctrlCtx.runOptions.kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	selector, err := workerlabel.LabelSelector(ctrlCtx.runOptions.workerName)
	if err != nil {
		glog.Fatal(err)
	}

	//Register the global error metric. Ensures that runtime.HandleError() increases the error metric
	metrics.RegisterRuntimErrorMetricCounter("kubermatic_rbac_generator", prometheus.DefaultRegisterer)

	ctrlCtx.stopCh = signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	ctrlCtx.kubeMasterClient = kubernetes.NewForConfigOrDie(config)
	ctrlCtx.kubermaticMasterClient = kubermaticclientset.NewForConfigOrDie(config)
	ctrlCtx.kubermaticMasterInformerFactory = externalversions.NewFilteredSharedInformerFactory(ctrlCtx.kubermaticMasterClient, informer.DefaultInformerResyncPeriod, metav1.NamespaceAll, selector)
	ctrlCtx.kubeMasterInformerFactory = kuberinformers.NewSharedInformerFactory(ctrlCtx.kubeMasterClient, informer.DefaultInformerResyncPeriod)

	ctrlCtx.allClusterProviders = []*rbaccontroller.ClusterProvider{}
	{
		clientcmdConfig, err := clientcmd.LoadFromFile(ctrlCtx.runOptions.kubeconfig)
		if err != nil {
			glog.Fatal(err)
		}

		for ctxName := range clientcmdConfig.Contexts {
			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*clientcmdConfig,
				ctxName,
				&clientcmd.ConfigOverrides{CurrentContext: ctxName},
				nil,
			)
			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				glog.Fatal(err)
			}

			var clusterPrefix string
			if ctxName == clientcmdConfig.CurrentContext {
				glog.V(2).Infof("Adding %s as master cluster", ctxName)
				clusterPrefix = rbaccontroller.MasterProviderPrefix
			} else {
				glog.V(2).Infof("Adding %s as seed cluster", ctxName)
				clusterPrefix = rbaccontroller.SeedProviderPrefix
			}
			kubeClient, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				glog.Fatal(err)
			}

			kubeInformerFactory := kuberinformers.NewSharedInformerFactory(kubeClient, time.Minute*5)
			kubermaticClient := kubermaticclientset.NewForConfigOrDie(cfg)
			kubermaticInformerFactory := externalversions.NewFilteredSharedInformerFactory(kubermaticClient, time.Minute*5, metav1.NamespaceAll, selector)
			ctrlCtx.allClusterProviders = append(ctrlCtx.allClusterProviders, rbaccontroller.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, ctxName), kubeClient, kubeInformerFactory, kubermaticClient, kubermaticInformerFactory))

			// special case the current context/master is also a seed cluster
			// we keep cluster resources also on master
			if ctxName == clientcmdConfig.CurrentContext {
				glog.V(2).Infof("Special case adding %s (current context) also as seed cluster", ctxName)
				clusterPrefix = rbaccontroller.SeedProviderPrefix
				ctrlCtx.allClusterProviders = append(ctrlCtx.allClusterProviders, rbaccontroller.NewClusterProvider(fmt.Sprintf("%s/%s", clusterPrefix, ctxName), kubeClient, kubeInformerFactory, kubermaticClient, kubermaticInformerFactory))
			}
		}
	}

	ctrl, err := rbaccontroller.New(
		rbaccontroller.NewMetrics(),
		ctrlCtx.allClusterProviders)
	if err != nil {
		glog.Fatal(err)
	}

	ctrlCtx.kubermaticMasterInformerFactory.Start(ctrlCtx.stopCh)
	ctrlCtx.kubeMasterInformerFactory.Start(ctrlCtx.stopCh)

	ctrlCtx.kubermaticMasterInformerFactory.WaitForCacheSync(ctrlCtx.stopCh)
	ctrlCtx.kubeMasterInformerFactory.WaitForCacheSync(ctrlCtx.stopCh)

	for _, seedClusterProvider := range ctrlCtx.allClusterProviders {
		seedClusterProvider.StartInformers(ctrlCtx.stopCh)
		if err := seedClusterProvider.WaitForCachesToSync(ctrlCtx.stopCh); err != nil {
			glog.Fatalf("failed to sync cache: %v", err)
		}
	}

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			<-ctrlCtx.stopCh
			return errors.New("user requested to stop the application")

		}, func(err error) {
			ctxDone()
		})
	}

	// This group is running an internal http server with metrics and other debug information
	{
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.Handler())

		s := http.Server{
			Addr:         ctrlCtx.runOptions.internalAddr,
			Handler:      m,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		g.Add(func() error {
			glog.Infof("Starting the internal http server: %s\n", ctrlCtx.runOptions.internalAddr)
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
			go ctrl.Run(ctrlCtx.runOptions.workerCount, ctrlCtx.stopCh)
			<-ctrlCtx.stopCh
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
