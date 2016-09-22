package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/kubermatic/api/controller/cluster"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/wait"
)

func main() {
	// parse flags
	homeDir := os.Getenv("HOME")
	kubeconfig := flag.String("kubeconfig", path.Join(homeDir, ".kube/config"), "The kubeconfig file with a current context.")
	masterResources := flag.String("master-resources", "", "The master resources path (required).")
	hostPattern := flag.String("host-pattern", "%s.%s.i.kubermatic.io", "The fmt.Sprintf pattern for the etcd server host, interpolated with the cluster name.")
	externalURL := flag.String("external-url", "%s.%s.kubermatic.io", "The fmt.Sprintf pattern for the apiserver host and the the dc.")
	dcFile := flag.String("datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	dev := flag.Bool("dev", false, "Create dev-mode clusters only processed by dev-mode cluster controller")

	flag.Parse()

	if *masterResources == "" {
		print("master-resources path is undefined\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// load list of datacenters
	dcs := map[string]provider.DatacenterMeta{}
	if *dcFile != "" {
		var err error
		dcs, err = provider.DatacentersMeta(*dcFile)
		if err != nil {
			log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", *dcFile, err))
		}
	}

	// create controller for each context
	clientcmdConfig, err := clientcmd.LoadFromFile(*kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	for ctx := range clientcmdConfig.Contexts {
		// create kube client
		clientcmdConfig, err := clientcmd.LoadFromFile(*kubeconfig)
		if err != nil {
			log.Fatal(err)
		}
		clientConfig := clientcmd.NewNonInteractiveClientConfig(
			*clientcmdConfig,
			ctx,
			&clientcmd.ConfigOverrides{},
			nil,
		)
		cfg, err := clientConfig.ClientConfig()
		if err != nil {
			log.Fatal(err)
		}
		client, err := client.New(cfg)
		if err != nil {
			log.Fatal(err)
		}

		// start controller
		cps := cloud.Providers(dcs)
		ctrl, err := cluster.NewController(
			ctx, client, cps, *masterResources, *hostPattern, *externalURL, *dev,
		)
		if err != nil {
			log.Fatal(err)
		}
		go ctrl.Run(wait.NeverStop)
	}

	<-wait.NeverStop
}
