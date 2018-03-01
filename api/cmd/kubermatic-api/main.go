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
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/crd"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/bringyourown"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/digitalocean"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/fake"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"k8s.io/apimachinery/pkg/util/wait"

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
)

const (
	informerResyncPeriod = 5 * time.Minute
)

func main() {
	flag.StringVar(&listenAddress, "address", ":8080", "The listenAddress to listen on")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig.")
	flag.StringVar(&prometheusAddr, "prometheus-address", "127.0.0.1:8085", "The Address on which the prometheus handler should be exposed")
	flag.StringVar(&masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	flag.StringVar(&workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	flag.StringVar(&versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.StringVar(&tokenIssuer, "token-issuer", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
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
	kubermaticMasterInformerFactory := externalversions.NewSharedInformerFactory(kubermaticMasterClient, informerResyncPeriod)

	sshKeyProvider := kubernetes.NewSSHKeyProvider(kubermaticMasterClient, kubermaticMasterInformerFactory.Kubermatic().V1().UserSSHKeies().Lister())
	userProvider := kubernetes.NewUserProvider(kubermaticMasterClient, kubermaticMasterInformerFactory.Kubermatic().V1().Users().Lister())

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

		kubermaticSeedClient := kubermaticclientset.NewForConfigOrDie(cfg)
		kubermaticSeedInformerFactory := externalversions.NewSharedInformerFactory(kubermaticSeedClient, informerResyncPeriod)
		clusterProviders[ctx] = kubernetes.NewClusterProvider(kubermaticSeedClient, kubermaticSeedInformerFactory.Kubermatic().V1().Clusters().Lister(), workerName)

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

	cloudProviders := map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(datacenters),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(datacenters),
		provider.OpenstackCloudProvider:    openstack.NewCloudProvider(datacenters),
	}

	r := handler.NewRouting(ctx, datacenters, clusterProviders, cloudProviders, sshKeyProvider, userProvider, authenticator, versions, updates)
	router := mux.NewRouter()
	r.Register(router)
	go metrics.ServeForever(prometheusAddr, "/metrics")
	glog.Info(fmt.Sprintf("Listening on %s", listenAddress))
	glog.Fatal(http.ListenAndServe(listenAddress, handlers.CombinedLoggingHandler(os.Stdout, router)))
}
