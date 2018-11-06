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
//     Version: 2.8
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
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	options, err := newServerRunOptions()
	if err != nil {
		glog.Fatalf("failed to create server run options due to = %v", err)
	}
	if err := options.validate(); err != nil {
		glog.Fatalf("incorrect flags were passed to the server, err  = %v", err)
	}

	providers, err := createInitProviders(options)
	if err != nil {
		glog.Fatalf("failed to create and initialize providers due to %v", err)
	}
	authenticator, issuerVerifier, err := createOIDCAuthenticatorIssuer(options)
	if err != nil {
		glog.Fatalf("failed to create a openid authenticator for issuer %s (oidcClientID=%s) due to %v", options.oidcIssuerURL, options.oidcClientID, err)
	}
	updateManager, err := version.NewFromFiles(options.versionsFile, options.updatesFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to create update manager due to %v", err))
	}
	apiHandler, err := createAPIHandler(options, providers, authenticator, issuerVerifier, updateManager)
	if err != nil {
		glog.Fatalf(fmt.Sprintf("failed to create API Handler due to %v", err))
	}

	go metrics.ServeForever(options.internalAddr, "/metrics")
	glog.Info(fmt.Sprintf("Listening on %s", options.listenAddress))
	glog.Fatal(http.ListenAndServe(options.listenAddress, handlers.CombinedLoggingHandler(os.Stdout, apiHandler)))
}

func createInitProviders(options serverRunOptions) (providers, error) {
	config, err := clientcmd.BuildConfigFromFlags("", options.kubeconfig)
	if err != nil {
		return providers{}, fmt.Errorf("unable to build client configuration from kubeconfig due to %v", err)
	}

	// create cluster providers - one foreach context
	clusterProviders := map[string]provider.ClusterProvider{}
	{
		clientcmdConfig, err := clientcmd.LoadFromFile(options.kubeconfig)
		if err != nil {
			return providers{}, fmt.Errorf("unable to create client config for due to %v", err)
		}

		for ctx := range clientcmdConfig.Contexts {
			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*clientcmdConfig,
				ctx,
				&clientcmd.ConfigOverrides{CurrentContext: ctx},
				nil,
			)
			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				return providers{}, fmt.Errorf("unable to create client config for %s due to %v", ctx, err)
			}
			glog.V(2).Infof("adding %s as seed", ctx)

			kubeClient := kubernetes.NewForConfigOrDie(cfg)
			kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, informer.DefaultInformerResyncPeriod)
			kubermaticSeedClient := kubermaticclientset.NewForConfigOrDie(cfg)
			kubermaticSeedInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticSeedClient, informer.DefaultInformerResyncPeriod)
			defaultImpersonationClientForSeed := kubernetesprovider.NewKubermaticImpersonationClient(cfg)

			clusterProviders[ctx] = kubernetesprovider.NewClusterProvider(
				defaultImpersonationClientForSeed.CreateImpersonatedClientSet,
				client.New(kubeInformerFactory.Core().V1().Secrets().Lister()),
				kubermaticSeedInformerFactory.Kubermatic().V1().Clusters().Lister(),
				options.workerName,
			)

			kubeInformerFactory.Start(wait.NeverStop)
			kubeInformerFactory.WaitForCacheSync(wait.NeverStop)
			kubermaticSeedInformerFactory.Start(wait.NeverStop)
			kubermaticSeedInformerFactory.WaitForCacheSync(wait.NeverStop)
		}
	}

	// create other providers
	kubermaticMasterClient := kubermaticclientset.NewForConfigOrDie(config)
	kubermaticMasterInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticMasterClient, informer.DefaultInformerResyncPeriod)
	defaultImpersonationClient := kubernetesprovider.NewKubermaticImpersonationClient(config)

	datacenters, err := provider.LoadDatacentersMeta(options.dcFile)
	if err != nil {
		return providers{}, fmt.Errorf("failed to load datacenter yaml %q: %v", options.dcFile, err)
	}
	cloudProviders := cloud.Providers(datacenters)
	sshKeyProvider := kubernetesprovider.NewSSHKeyProvider(defaultImpersonationClient.CreateImpersonatedClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserSSHKeies().Lister())
	userProvider := kubernetesprovider.NewUserProvider(kubermaticMasterClient, kubermaticMasterInformerFactory.Kubermatic().V1().Users().Lister())
	projectMemberProvider := kubernetesprovider.NewProjectMemberProvider(defaultImpersonationClient.CreateImpersonatedClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserProjectBindings().Lister())
	projectProvider, err := kubernetesprovider.NewProjectProvider(defaultImpersonationClient.CreateImpersonatedClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().Projects().Lister())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create project provider due to %v", err)
	}

	kubermaticMasterInformerFactory.Start(wait.NeverStop)
	kubermaticMasterInformerFactory.WaitForCacheSync(wait.NeverStop)

	return providers{sshKey: sshKeyProvider, user: userProvider, project: projectProvider, projectMember: projectMemberProvider, memberMapper: projectMemberProvider, cloud: cloudProviders, clusters: clusterProviders, datacenters: datacenters}, nil
}

func createOIDCAuthenticatorIssuer(options serverRunOptions) (handler.OIDCAuthenticator, handler.OIDCIssuerVerifier, error) {
	authenticator, err := handler.NewOpenIDAuthenticator(
		options.oidcIssuerURL,
		options.oidcClientID,
		options.oidcClientSecret,
		options.oidcRedirectURI,
		handler.NewCombinedExtractor(
			handler.NewHeaderBearerTokenExtractor("Authorization"),
			handler.NewQueryParamBearerTokenExtractor("token"),
		),
		options.oidcSkipTLSVerify,
	)

	return authenticator, authenticator, err
}

func createAPIHandler(options serverRunOptions, prov providers, oidcAuthenticator handler.OIDCAuthenticator, oidcIssuerVerifier handler.OIDCIssuerVerifier, updateManager *version.Manager) (http.HandlerFunc, error) {
	var prometheusClient prometheusapi.Client
	if options.featureGates.Enabled(PrometheusEndpoint) {
		var err error
		if prometheusClient, err = prometheusapi.NewClient(prometheusapi.Config{
			Address: options.prometheusURL,
		}); err != nil {
			return nil, err
		}
	}

	r := handler.NewRouting(
		prov.datacenters,
		prov.clusters,
		prov.cloud,
		prov.sshKey,
		prov.user,
		prov.project,
		oidcAuthenticator,
		oidcIssuerVerifier,
		updateManager,
		prometheusClient,
		prov.projectMember,
		prov.memberMapper,
	)

	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	v1AlphaRouter := mainRouter.PathPrefix("/api/v1alpha").Subrouter()
	r.RegisterV1(v1Router)
	r.RegisterV1Optional(v1Router,
		options.featureGates.Enabled(OIDCKubeCfgEndpoint),
		handler.OIDCConfiguration{
			URL:                  options.oidcIssuerURL,
			ClientID:             options.oidcClientID,
			ClientSecret:         options.oidcClientSecret,
			OfflineAccessAsScope: options.oidcOfflineAccessAsScope,
		},
		mainRouter)
	r.RegisterV1Alpha(v1AlphaRouter)

	metrics.RegisterHTTPVecs()

	lookupRoute := func(r *http.Request) string {
		var match mux.RouteMatch
		ok := mainRouter.Match(r, &match)
		if !ok {
			return ""
		}

		name := match.Route.GetName()
		if name != "" {
			return name
		}

		name, err := match.Route.GetPathTemplate()
		if err != nil {
			return ""
		}

		return name
	}

	return metrics.InstrumentHandler(mainRouter, lookupRoute), nil
}
