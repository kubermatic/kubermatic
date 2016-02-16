package main

import (
	"bufio"
	"flag"
	"io/ioutil"
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
	yaml "gopkg.in/yaml.v2"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

// Datacenter describes a Kubermatic datacenter.
type Datacenter struct {
	Location string `yaml:"location"`
	Country  string `yaml:"country"`
	Provider string `yaml:"provider"`
}

// DatacentersMetadata describes a number of Kubermatic datacenters.
type DatacentersMetadata struct {
	Datacenters map[string]Datacenter `yaml:"datacenters"`
}

func datacenters(path string) (*DatacentersMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	datacenters := DatacentersMetadata{}
	err = yaml.Unmarshal(bytes, &datacenters)
	if err != nil {
		return nil, err
	}

	return &datacenters, nil
}

func main() {
	// parse flags
	kubeconfig := flag.String("kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	auth := flag.Bool("auth", true, "Activate authentication with JSON Web Tokens")
	dcFile := flag.String("datacenters", ".datacenters.yaml", "The datacenters.yaml file path")
	jwtKey := flag.String("jwt-key", "", "The JSON Web Token validation key, encoded in base64")
	flag.Parse()

	// load meta data for datacenters
	dcs, err := datacenters(*dcFile)
	if err != nil {
		log.Fatal(err)
	}

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

		meta := Datacenter{}
		if dc, found := dcs.Datacenters[ctx]; found {
			meta = dc
		}
		kps[ctx] = kubernetes.NewKubernetesProvider(
			cfg,
			cps,
			meta.Location,
			meta.Country,
			meta.Provider,
		)
	}

	// start server
	ctx := context.Background()
	b := handler.NewBinding(ctx, kps, cps, *auth, *jwtKey)
	mux := mux.NewRouter()
	b.Register(mux)

	log.Fatal(http.ListenAndServe(":8080", ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
