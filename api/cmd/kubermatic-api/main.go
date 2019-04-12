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
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	metricspkg "github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
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
	oidcIssuerVerifier, err := createOIDCClients(options)
	if err != nil {
		glog.Fatalf("failed to create an openid authenticator for issuer %s (oidcClientID=%s) due to %v", options.oidcURL, options.oidcAuthenticatorClientID, err)
	}
	tokenVerifiers, tokenExtractors, err := createAuthClients(options)
	if err != nil {
		glog.Fatalf("failed to create auth clients due to %v", err)
	}
	updateManager, err := version.NewFromFiles(options.versionsFile, options.updatesFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to create update manager due to %v", err))
	}
	apiHandler, err := createAPIHandler(options, providers, oidcIssuerVerifier, tokenVerifiers, tokenExtractors, updateManager)
	if err != nil {
		glog.Fatalf(fmt.Sprintf("failed to create API Handler due to %v", err))
	}

	go metricspkg.ServeForever(options.internalAddr, "/metrics")
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

			userClusterConnectionProvider, err := client.NewExternal(kubeInformerFactory.Core().V1().Secrets().Lister())
			if err != nil {
				return providers{}, fmt.Errorf("failed to get userClusterConnectionProvider: %v", err)
			}

			clusterProviders[ctx] = kubernetesprovider.NewClusterProvider(
				defaultImpersonationClientForSeed.CreateImpersonatedKubermaticClientSet,
				userClusterConnectionProvider,
				kubermaticSeedInformerFactory.Kubermatic().V1().Clusters().Lister(),
				options.workerName,
				rbac.ExtractGroupPrefix,
			)

			kubeInformerFactory.Start(wait.NeverStop)
			kubeInformerFactory.WaitForCacheSync(wait.NeverStop)
			kubermaticSeedInformerFactory.Start(wait.NeverStop)
			kubermaticSeedInformerFactory.WaitForCacheSync(wait.NeverStop)
		}
	}

	// create other providers
	kubeClient := kubernetes.NewForConfigOrDie(config)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, informer.DefaultInformerResyncPeriod)
	kubermaticMasterClient := kubermaticclientset.NewForConfigOrDie(config)
	kubermaticMasterInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticMasterClient, informer.DefaultInformerResyncPeriod)
	defaultKubermaticImpersonationClient := kubernetesprovider.NewKubermaticImpersonationClient(config)
	defaultKubernetesImpersonationClient := kubernetesprovider.NewKubernetesImpersonationClient(config)

	datacenters, err := provider.LoadDatacentersMeta(options.dcFile)
	if err != nil {
		return providers{}, fmt.Errorf("failed to load datacenter yaml %q: %v", options.dcFile, err)
	}
	cloudProviders := cloud.Providers(datacenters)
	userLister := kubermaticMasterInformerFactory.Kubermatic().V1().Users().Lister()
	sshKeyProvider := kubernetesprovider.NewSSHKeyProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserSSHKeys().Lister())
	userProvider := kubernetesprovider.NewUserProvider(kubermaticMasterClient, userLister)

	serviceAccountTokenGenerator, err := serviceaccount.JWTTokenGenerator([]byte(options.serviceAccountSigningKey))
	if err != nil {
		return providers{}, fmt.Errorf("failed to create service account token generator due to %v", err)
	}
	serviceAccountTokenAuth := serviceaccount.JWTTokenAuthenticator([]byte(options.serviceAccountSigningKey))
	serviceAccountTokenProvider := kubernetesprovider.NewServiceAccountTokenProvider(defaultKubernetesImpersonationClient.CreateImpersonatedKubernetesClientSet, serviceAccountTokenGenerator, serviceAccountTokenAuth, kubeInformerFactory.Core().V1().Secrets().Lister())
	serviceAccountProvider := kubernetesprovider.NewServiceAccountProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, userLister, options.domain)
	projectMemberProvider := kubernetesprovider.NewProjectMemberProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserProjectBindings().Lister(), userLister)
	projectProvider, err := kubernetesprovider.NewProjectProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().Projects().Lister())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create project provider due to %v", err)
	}

	privilegedProjectProvider, err := kubernetesprovider.NewPrivilegedProjectProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create privileged project provider due to %v", err)
	}

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubermaticMasterInformerFactory.Start(wait.NeverStop)
	kubermaticMasterInformerFactory.WaitForCacheSync(wait.NeverStop)

	return providers{
			sshKey:                      sshKeyProvider,
			user:                        userProvider,
			serviceAccountProvider:      serviceAccountProvider,
			serviceAccountTokenProvider: serviceAccountTokenProvider,
			project:                     projectProvider,
			privilegedProject:           privilegedProjectProvider,
			projectMember:               projectMemberProvider,
			memberMapper:                projectMemberProvider,
			cloud:                       cloudProviders,
			clusters:                    clusterProviders,
			datacenters:                 datacenters},
		nil
}

func createOIDCClients(options serverRunOptions) (auth.OIDCIssuerVerifier, error) {
	return auth.NewOpenIDClient(
		options.oidcURL,
		options.oidcIssuerClientID,
		options.oidcIssuerClientSecret,
		options.oidcIssuerRedirectURI,
		auth.NewCombinedExtractor(
			auth.NewHeaderBearerTokenExtractor("Authorization"),
			auth.NewQueryParamBearerTokenExtractor("token"),
		),
		options.oidcSkipTLSVerify,
	)
}

func createAuthClients(options serverRunOptions) (auth.TokenVerifier, auth.TokenExtractor, error) {
	oidcExtractorVerifier, err := auth.NewOpenIDClient(
		options.oidcURL,
		options.oidcAuthenticatorClientID,
		"",
		"",
		auth.NewCombinedExtractor(
			auth.NewHeaderBearerTokenExtractor("Authorization"),
			auth.NewQueryParamBearerTokenExtractor("token"),
		),
		options.oidcSkipTLSVerify,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OIDC Authenticator: %v", err)
	}

	tokenVerifiers := auth.NewTokenVerifierPlugins([]auth.TokenVerifier{oidcExtractorVerifier})
	tokenExtractors := auth.NewTokenExtractorPlugins([]auth.TokenExtractor{oidcExtractorVerifier})
	return tokenVerifiers, tokenExtractors, nil
}

func createAPIHandler(options serverRunOptions, prov providers, oidcIssuerVerifier auth.OIDCIssuerVerifier, tokenVerifiers auth.TokenVerifier, tokenExtractors auth.TokenExtractor, updateManager common.UpdateManager) (http.HandlerFunc, error) {
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
		prov.serviceAccountProvider,
		prov.serviceAccountTokenProvider,
		prov.project,
		prov.privilegedProject,
		oidcIssuerVerifier,
		tokenVerifiers,
		tokenExtractors,
		updateManager,
		prometheusClient,
		prov.projectMember,
		prov.memberMapper,
	)

	registerMetrics()

	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	v1AlphaRouter := mainRouter.PathPrefix("/api/v1alpha").Subrouter()
	r.RegisterV1(v1Router, metrics)
	r.RegisterV1Legacy(v1Router)
	r.RegisterV1Optional(v1Router,
		options.featureGates.Enabled(OIDCKubeCfgEndpoint),
		common.OIDCConfiguration{
			URL:                  options.oidcURL,
			ClientID:             options.oidcIssuerClientID,
			ClientSecret:         options.oidcIssuerClientSecret,
			CookieHashKey:        options.oidcIssuerCookieHashKey,
			CookieSecureMode:     options.oidcIssuerCookieSecureMode,
			OfflineAccessAsScope: options.oidcIssuerOfflineAccessAsScope,
		},
		mainRouter)
	r.RegisterV1Alpha(v1AlphaRouter)

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

	return instrumentHandler(mainRouter, lookupRoute), nil
}
