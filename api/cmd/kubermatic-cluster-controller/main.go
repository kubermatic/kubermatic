package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/controller/cluster"
	"github.com/kubermatic/kubermatic/api/controller/version"
	"github.com/kubermatic/kubermatic/api/metrics"
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/provider/cloud"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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

	// Create crd's

	for ctx := range clientcmdConfig.Contexts {
		// create kube client
		clientcmdConfig, err := clientcmd.LoadFromFile(*kubeConfig)
		if err != nil {
			glog.Fatal(err)
		}
		clientConfig := clientcmd.NewNonInteractiveClientConfig(
			*clientcmdConfig,
			ctx,
			&clientcmd.ConfigOverrides{},
			nil,
		)

		cfg, err := clientConfig.ClientConfig()
		if err != nil {
			glog.Fatal(err)
		}
		client := kubernetes.NewForConfigOrDie(cfg)
		crdClient := crdclient.NewForConfigOrDie(cfg)

		// start controller
		cps := cloud.Providers(dcs)
		ctrl, err := cluster.NewController(
			ctx,
			client,
			crdClient,
			cps,
			versions,
			updates,
			*masterResources,
			*externalURL,
			*workerName,
			*apiserverExternalPort,
			dcs,
		)
		if err != nil {
			glog.Fatal(err)
		}
		go ctrl.Run(wait.NeverStop)
	}

	go metrics.ServeForever(*prometheusAddr, *prometheusPath)
	<-wait.NeverStop
}
