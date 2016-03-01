package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	ghandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/handler"
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
	flag.Parse()

	// load meta data for datacenters
	metas := map[string]kubernetes.DatacenterMeta{}
	if *dcFile != "" {
		var err error
		metas, err = kubernetes.DatacentersMeta(*dcFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	// create CloudProviders
	cps := cloud.Providers()

	// create KubernetesProvider for each context in the kubeconfig
	kps, err := kubernetes.Providers(*kubeconfig, cps, metas)
	if err != nil {
		log.Fatal(err)
	}

	// start server
	ctx := context.Background()
	r := handler.NewRouting(ctx, kps, cps, *auth, *jwtKey)
	mux := mux.NewRouter()
	r.Register(mux)

	log.Fatal(http.ListenAndServe(":8080", ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
