package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/crd"
	mastercrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	seedcrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned"
	masterinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/master"
	seedinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"

	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	prometheusAddr        = flag.String("prometheus-address", "127.0.0.1:8085", "The Address on which the prometheus handler should be exposed")
	prometheusPath        = flag.String("prometheus-path", "/metrics", "The path on the host, on which the handler is available")
	kubeConfig            = flag.String("kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	masterResources       = flag.String("master-resources", "", "The master resources path (Required).")
	externalURL           = flag.String("external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	dcFile                = flag.String("datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	workerName            = flag.String("worker-name", "", "Create clusters only processed by worker-name cluster controller")
	versionsFile          = flag.String("versions", "versions.yaml", "The versions.yaml file path")
	updatesFile           = flag.String("updates", "updates.yaml", "The updates.yaml file path")
	apiserverExternalPort = flag.Int("apiserver-external-port", 8443, "Port on which the apiserver of a client cluster should be reachable")
	workerCount           = flag.Int("worker-count", 4, "Number of workers which process the clusters in parallel.")
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

	// create controller for each context
	clientcmdConfig, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		glog.Fatal(err)
	}

	stop := make(chan struct{})
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		close(stop)
	}()

	for dc := range clientcmdConfig.Contexts {
		// create kube kubeclient
		clientcmdConfig, err := clientcmd.LoadFromFile(*kubeConfig)
		if err != nil {
			glog.Fatal(err)
		}
		clientConfig := clientcmd.NewNonInteractiveClientConfig(
			*clientcmdConfig,
			dc,
			&clientcmd.ConfigOverrides{},
			nil,
		)

		cfg, err := clientConfig.ClientConfig()
		if err != nil {
			glog.Fatal(err)
		}
		kubeclient := kubernetes.NewForConfigOrDie(cfg)
		seedCrdClient := seedcrdclient.NewForConfigOrDie(cfg)
		masterCrdClient := mastercrdclient.NewForConfigOrDie(cfg)

		// Create crd's
		extclient := apiextclient.NewForConfigOrDie(cfg)
		err = crd.EnsureCustomResourceDefinitions(extclient)
		if err != nil {
			glog.Error(err)
		}

		seedInformerGroup := seedinformer.New(kubeclient, seedCrdClient)
		masterInformerGroup := masterinformer.New(masterCrdClient)

		// start controller
		cps := cloud.Providers(dcs)
		ctrl, err := cluster.NewController(
			dc,
			kubeclient,
			seedCrdClient,
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
			seedInformerGroup,
		)
		if err != nil {
			glog.Fatal(err)
		}

		seedInformerGroup.Run(stop)
		masterInformerGroup.Run(stop)
		cache.WaitForCacheSync(stop, seedInformerGroup.HasSynced)

		go ctrl.Run(*workerCount, stop)

	}

	go metrics.ServeForever(*prometheusAddr, *prometheusPath)
	<-stop
}
