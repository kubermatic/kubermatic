package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/handler"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/kubermatic/api/provider/kubernetes"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	workerName       = flag.String("worker-name", "", "Create clusters only processed by worker-name cluster controller")
	kubeConfig       = flag.String("kubeconfig", "", "The kubeconfig file path with one context per Kubernetes provider")
	auth             = flag.Bool("auth", true, "Activate authentication with JSON Web Tokens")
	dcFile           = flag.String("datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	jwtKey           = flag.String("jwt-key", "", "The JSON Web Token validation key, encoded in base64")
	address          = flag.String("address", ":8080", "The address to listen on")
	masterKubeconfig = flag.String("master-kubeconfig", "", "When set it will overwrite the usage of the InClusterConfig")
)

func main() {
	flag.Parse()

	dcs, err := provider.DatacentersMeta(*dcFile)
	if err != nil {
		glog.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", *dcFile, err))
	}

	// create CloudProviders
	cps := cloud.Providers(dcs)
	// create KubernetesProvider for each context in the kubeconfig
	kps, err := kubernetes.Providers(*kubeConfig, dcs, cps, *workerName)
	if err != nil {
		glog.Fatal(err)
	}

	var config *rest.Config
	config, err = clientcmd.BuildConfigFromFlags("", *masterKubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	masterTPRClient, err := extensions.WrapClientsetWithExtensions(config)
	if err != nil {
		glog.Fatal(err)
	}

	// start server
	ctx := context.Background()
	r := handler.NewRouting(ctx, dcs, kps, cps, *auth, *jwtKey, masterTPRClient)
	router := mux.NewRouter()
	r.Register(router)
	glog.Info(fmt.Sprintf("Listening on %s", *address))
	glog.Fatal(http.ListenAndServe(*address, handlers.CombinedLoggingHandler(os.Stdout, router)))
}
