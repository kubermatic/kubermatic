// Package classification Kubermatic API.
//
// Kubermatic API
//
// This describes possible operations which can be made against the Kubermatic API.
//
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     Host: cloud.kubermatic.io
//     Version: 2.2.3
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
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/crd"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	listenAddress   string
	kubeconfig      string
	prometheusAddr  string
	masterResources string
	dcFile          string
	workerName      string
	versionsFile    string
	updatesFile     string
	tokenIssuer     string
	clientID        string

	tokenIssuerSkipTLSVerify bool
)

const (
	informerResyncPeriod = 5 * time.Minute
)

func main() {
	flag.StringVar(&listenAddress, "address", ":8080", "The address to listen on")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig.")
	flag.StringVar(&prometheusAddr, "prometheus-address", "127.0.0.1:8085", "The Address on which the prometheus handler should be exposed")
	flag.StringVar(&masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	flag.StringVar(&versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.StringVar(&tokenIssuer, "token-issuer", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.BoolVar(&tokenIssuerSkipTLSVerify, "token-issuer-skip-tls-verify", false, "SKip TLS verification for the token issuer")
	flag.StringVar(&clientID, "client-id", "", "OpenID client ID")
	flag.Parse()

	datacenters, err := provider.LoadDatacentersMeta(dcFile)
	if err != nil {
		glog.Fatalf("failed to load datacenter yaml %q: %v", dcFile, err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	// Create crd's
	extclient := apiextclient.NewForConfigOrDie(config)
	err = crd.EnsureCustomResourceDefinitions(extclient)
	if err != nil {
		glog.Fatal(err)
	}

	kubermaticMasterClient := kubermaticclientset.NewForConfigOrDie(config)
	kubermaticMasterInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticMasterClient, informerResyncPeriod)

	sshKeyProvider := kubernetesprovider.NewSSHKeyProvider(kubermaticMasterClient, kubermaticMasterInformerFactory.Kubermatic().V1().UserSSHKeies().Lister(), handler.IsAdmin)
	userProvider := kubernetesprovider.NewUserProvider(kubermaticMasterClient, kubermaticMasterInformerFactory.Kubermatic().V1().Users().Lister())
	defaultImpersonationClient := kubernetesprovider.NewKubermaticImpersonationClient(config)
	projectProvider, err := kubernetesprovider.NewProjectProvider(defaultImpersonationClient.CreateImpersonatedClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().Projects().Lister())
	if err != nil {
		glog.Fatalf("failed to create project provider due to %v", err)
	}

	// create a cluster provider for each context
	clientcmdConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	clusterProviders := map[string]provider.ClusterProvider{}
	for ctx := range clientcmdConfig.Contexts {
		clientConfig := clientcmd.NewNonInteractiveClientConfig(
			*clientcmdConfig,
			ctx,
			&clientcmd.ConfigOverrides{CurrentContext: ctx},
			nil,
		)
		cfg, err := clientConfig.ClientConfig()
		if err != nil {
			glog.Fatal(err)
		}

		glog.V(2).Infof("adding %s as seed", ctx)

		kubeClient := kubernetes.NewForConfigOrDie(cfg)
		kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, informerResyncPeriod)

		kubermaticSeedClient := kubermaticclientset.NewForConfigOrDie(cfg)
		kubermaticSeedInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticSeedClient, informerResyncPeriod)
		clusterProviders[ctx] = kubernetesprovider.NewClusterProvider(kubermaticSeedClient, client.New(kubeInformerFactory.Core().V1().Secrets().Lister()), kubermaticSeedInformerFactory.Kubermatic().V1().Clusters().Lister(), workerName, handler.IsAdmin)

		kubeInformerFactory.Start(wait.NeverStop)
		kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

		kubermaticSeedInformerFactory.Start(wait.NeverStop)
		kubermaticSeedInformerFactory.WaitForCacheSync(wait.NeverStop)
	}
	kubermaticMasterInformerFactory.Start(wait.NeverStop)
	kubermaticMasterInformerFactory.WaitForCacheSync(wait.NeverStop)

	authenticator, err := handler.NewOpenIDAuthenticator(
		tokenIssuer,
		clientID,
		handler.NewCombinedExtractor(
			handler.NewHeaderBearerTokenExtractor("Authorization"),
			handler.NewQueryParamBearerTokenExtractor("token"),
		),
		tokenIssuerSkipTLSVerify,
	)
	if err != nil {
		glog.Fatalf("failed to create a openid authenticator for issuer %s (clientID=%s): %v", tokenIssuer, clientID, err)
	}

	// start server
	ctx := context.Background()

	// load versions
	versions, err := version.LoadVersions(versionsFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load version yaml %q: %v", versionsFile, err))
	}

	// load updates
	updates, err := version.LoadUpdates(updatesFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load version yaml %q: %v", versionsFile, err))
	}

	cloudProviders := cloud.Providers(datacenters)

	r := handler.NewRouting(
		ctx,
		datacenters,
		clusterProviders,
		cloudProviders,
		sshKeyProvider,
		userProvider,
		projectProvider,
		authenticator,
		versions,
		updates,
		masterResources,
	)

	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	v2Router := mainRouter.PathPrefix("/api/v2").Subrouter()
	v3Router := mainRouter.PathPrefix("/api/v3").Subrouter()
	r.RegisterV1(v1Router)
	r.RegisterV2(v2Router)
	r.RegisterV3(v3Router)

	go metrics.ServeForever(prometheusAddr, "/metrics")
	glog.Info(fmt.Sprintf("Listening on %s", listenAddress))
	glog.Fatal(http.ListenAndServe(listenAddress, handlers.CombinedLoggingHandler(os.Stdout, mainRouter)))
}
