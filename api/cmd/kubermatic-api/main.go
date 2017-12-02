//go:generate swagger generate spec
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     Host: localhost
//     Version: 0.0.1
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
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
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/crd"
	mastercrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"

	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	prometheusAddr   = flag.String("prometheus-address", "127.0.0.1:8085", "The Address on which the prometheus handler should be exposed")
	prometheusPath   = flag.String("prometheus-path", "/metrics", "The path on the host, on which the handler is available")
	workerName       = flag.String("worker-name", "", "Create clusters only processed by worker-name cluster controller")
	dcFile           = flag.String("datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	address          = flag.String("address", ":8080", "The address to listen on")
	masterKubeconfig = flag.String("master-kubeconfig", "", "When set it will overwrite the usage of the InClusterConfig")
	tokenIssuer      = flag.String("token-issuer", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	versionsFile     = flag.String("versions", "versions.yaml", "The versions.yaml file path")
	updatesFile      = flag.String("updates", "updates.yaml", "The updates.yaml file path")
	clientID         = flag.String("client-id", "", "OpenID client ID")
)

func main() {
	flag.Parse()

	dcs, err := provider.LoadDatacentersMeta(*dcFile)
	if err != nil {
		glog.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", *dcFile, err))
	}

	// create CloudProviders
	cps := cloud.Providers(dcs)

	var config *rest.Config
	config, err = clientcmd.BuildConfigFromFlags("", *masterKubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	config.Impersonate = rest.ImpersonationConfig{}
	masterCrdClient := mastercrdclient.NewForConfigOrDie(config)
	kp := kubernetes.NewKubernetesProvider(masterCrdClient, cps, *workerName, dcs)

	// Create crd's
	extclient := apiextclient.NewForConfigOrDie(config)
	err = crd.EnsureCustomResourceDefinitions(extclient)
	if err != nil {
		glog.Error(err)
	}

	authenticator := auth.NewOpenIDAuthenticator(
		*tokenIssuer,
		*clientID,
		auth.NewCombinedExtractor(
			auth.NewHeaderBearerTokenExtractor("Authorization"),
			auth.NewQueryParamBearerTokenExtractor("token"),
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

	r := handler.NewRouting(ctx, dcs, kp, cps, authenticator, versions, updates)
	router := mux.NewRouter()
	r.Register(router)
	go metrics.ServeForever(*prometheusAddr, *prometheusPath)
	glog.Info(fmt.Sprintf("Listening on %s", *address))
	glog.Fatal(http.ListenAndServe(*address, handlers.CombinedLoggingHandler(os.Stdout, router)))
}
