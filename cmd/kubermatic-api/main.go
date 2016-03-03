package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	ghandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/handler"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/kubermatic/api/provider/kubernetes"
	"golang.org/x/net/context"
)

func main() {
	// parse flags
	kubeconfig := flag.String("kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	auth := flag.Bool("auth", true, "Activate authentication with JSON Web Tokens")
	dcFile := flag.String("datacenters", "", "The datacenters.yaml file path")
	jwtKey := flag.String("jwt-key", "", "The JSON Web Token validation key, encoded in base64")
	address := flag.String("address", ":8080", "The address to listen on")
	flag.Parse()

	// load list of datacenters
	dcs := map[string]provider.DatacenterMeta{}
	if *dcFile != "" {
		var err error
		dcs, err = provider.DatacentersMeta(*dcFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	// create CloudProviders
	cps := cloud.Providers()

	// create KubernetesProvider for each context in the kubeconfig
	kps, err := kubernetes.Providers(*kubeconfig, cps)
	if err != nil {
		log.Fatal(err)
	}

	// start server
	ctx := context.Background()
	r := handler.NewRouting(ctx, dcs, kps, cps, *auth, *jwtKey)
	mux := mux.NewRouter()
	r.Register(mux)
	log.Println(fmt.Sprintf("Listening on %s", *address))
	log.Fatal(http.ListenAndServe(*address, ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
