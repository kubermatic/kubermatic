// @APIVersion 1.4.0
// @APITitle Kubermatic REST API
// @APIDescription Kubermatic REST API
// @BasePath http://loodse.com/
// @Contact info@loodse.com
// @License TBD
// @LicenseUrl https://github.com/kubermatic/api
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
	"github.com/kubermatic/kubermatic/api/controller/version"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/handler"
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/provider/cloud"
	"github.com/kubermatic/kubermatic/api/provider/kubernetes"

	"github.com/kubermatic/kubermatic/api/metrics"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	prometheusAddr   = flag.String("prometheus-address", "127.0.0.1:8085", "The Address on which the prometheus handler should be exposed")
	prometheusPath   = flag.String("prometheus-path", "/metrics", "The path on the host, on which the handler is available")
	workerName       = flag.String("worker-name", "", "Create clusters only processed by worker-name cluster controller")
	kubeConfig       = flag.String("kubeconfig", "", "The kubeconfig file path with one context per Kubernetes provider")
	dcFile           = flag.String("datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	address          = flag.String("address", ":8080", "The address to listen on")
	masterKubeconfig = flag.String("master-kubeconfig", "", "When set it will overwrite the usage of the InClusterConfig")
	tokenIssuer      = flag.String("token-issuer", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	versionsFile     = flag.String("versions", "versions.yaml", "The versions.yaml file path")
	updatesFile      = flag.String("updates", "updates.yaml", "The updates.yaml file path")
	clientID         = flag.String("client-id", "", "OpenID client ID")
	_                = flag.String("swagger-path", "./swagger/api/index.json", "OpenID client ID")
)

func main() {
	flag.Parse()

	dcs, err := provider.LoadDatacentersMeta(*dcFile)
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

	authenticator := handler.NewOpenIDAuthenticator(
		*tokenIssuer,
		*clientID,
		handler.NewCombinedExtractor(
			handler.NewHeaderBearerTokenExtractor("Authorization"),
			handler.NewQueryParamBearerTokenExtractor("token"),
		),
	)

	// start server
	ctx := context.Background()

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

	r := handler.NewRouting(ctx, dcs, kps, cps, authenticator, masterTPRClient, versions, updates)
	router := mux.NewRouter()
	r.Register(router)
	go metrics.ServeForever(*prometheusAddr, *prometheusPath)
	glog.Info(fmt.Sprintf("Listening on %s", *address))
	glog.Fatal(http.ListenAndServe(*address, handlers.CombinedLoggingHandler(os.Stdout, router)))
}
