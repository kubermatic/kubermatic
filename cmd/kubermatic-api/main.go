package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/kubermatic/api/handler"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/kubermatic/api/provider/kubernetes"
	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"

	ghandlers "github.com/gorilla/handlers"
)

func main() {
	// parse flags
	kubeconfig := flag.String("kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	auth := flag.Bool("auth", true, "Activate authentication with JSON Web Tokens")
	jwtKey := flag.String("jwt-key", "", "The JSON Web Token validation key, encoded in base64")
	flag.Parse()

	// create CloudProviders
	cps := cloud.Providers()

	// create KubernetesProvider for each context in the kubeconfig
	kps := map[string]provider.KubernetesProvider{
		"fake-1": kubernetes.NewKubernetesFakeProvider("fake-1", cps),
		"fake-2": kubernetes.NewKubernetesFakeProvider("fake-2", cps),
	}
	clientcmdConfig, err := clientcmd.LoadFromFile(*kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	for ctx := range clientcmdConfig.Contexts {
		clientconfig := clientcmd.NewNonInteractiveClientConfig(
			*clientcmdConfig,
			ctx,
			&clientcmd.ConfigOverrides{},
		)
		cfg, err := clientconfig.ClientConfig()
		if err != nil {
			log.Fatal(err)
		}

		kps[ctx] = kubernetes.NewKubernetesProvider(cfg, cps, "Frankfurt", "de", "gce")
	}

	// start server
	ctx := context.Background()
	b := handler.NewBinding(ctx, kps, cps, *auth, *jwtKey)
	mux := mux.NewRouter()
	b.Register(mux)

	log.Fatal(http.ListenAndServe(":8080", ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
