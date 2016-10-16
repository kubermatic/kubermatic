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
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

func main() {

	initConfig()

	// parse flags
	kubeconfig := flag.String("kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	auth := flag.Bool("auth", true, "Activate authentication with JSON Web Tokens")
	dcFile := flag.String("datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	secretsFile := flag.String("secrets", "secrets.yaml", "The secrets.yaml file path")
	jwtKey := flag.String("jwt-key", "", "The JSON Web Token validation key, encoded in base64")
	address := flag.String("address", ":8080", "The address to listen on")
	dev := flag.Bool("dev", false, "Create dev-mode clusters only processed by dev-mode cluster controller")
	flag.Parse()

	// load list of datacenters
	dcs := map[string]provider.DatacenterMeta{}
	if *dcFile != "" {
		var err error
		dcs, err = provider.DatacentersMeta(*dcFile)
		if err != nil {
			log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", *dcFile, err))
		}
	}

	// create CloudProviders
	cps := cloud.Providers(dcs)

	// create KubernetesProvider for each context in the kubeconfig
	kps, err := kubernetes.Providers(*kubeconfig, dcs, cps, *secretsFile, *dev)
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

func initConfig() {
	viper.SetConfigName("config")         // name of config file (without extension)
	viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
	viper.ReadInConfig()                  // Find and read the config file
}
