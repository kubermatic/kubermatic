package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/cluster"
	"github.com/kubermatic/api/controller/version"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/extensions/etcd"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
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

	dcs, err := provider.DatacentersMeta(*dcFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load datacenter yaml %q: %v", *dcFile, err))
	}

	// load versions
	versions, err := version.LoadVersions(*versionsFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load version yaml %q: %v", *versionsFile, err))
	}

	// load updates
	updates := []api.MasterUpdate{}
	if *updatesFile != "" {
		var err error
		updates, err = version.LoadUpdates(*updatesFile)
		if err != nil {
			glog.Fatal(fmt.Sprintf("failed to load updates yaml %q: %v", *updatesFile, err))
		}
	}

	// create controller for each context
	clientcmdConfig, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		glog.Fatal(err)
	}

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
		client, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			glog.Fatal(err)
		}
		tprClient, err := extensions.WrapClientsetWithExtensions(cfg)
		if err != nil {
			glog.Fatal(err)
		}

		etcdClusterClient, err := etcd.WrapClientsetWithExtensions(cfg)
		if err != nil {
			glog.Fatal(err)
		}

		// start controller
		cps := cloud.Providers(dcs)
		ctrl, err := cluster.NewController(
			ctx,
			client,
			tprClient,
			etcdClusterClient,
			cps,
			versions,
			updates,
			*masterResources,
			*externalURL,
			*workerName,
			*apiserverExternalPort,
		)
		if err != nil {
			glog.Fatal(err)
		}
		go ctrl.Run(wait.NeverStop)
	}

	<-wait.NeverStop
}
