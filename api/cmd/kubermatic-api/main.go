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

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/credentials"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
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
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	options, err := newServerRunOptions()
	if err != nil {
		fmt.Printf("failed to create server run options due to = %v\n", err)
		os.Exit(1)
	}
	if err := options.validate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rawLog := kubermaticlog.New(options.log.Debug, kubermaticlog.Format(options.log.Format))
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	kubermaticlog.Logger = log

	providers, err := createInitProviders(options)
	if err != nil {
		log.Fatalw("failed to create and initialize providers", "error", err)
	}
	oidcIssuerVerifier, err := createOIDCClients(options)
	if err != nil {
		log.Fatalw("failed to create an openid authenticator", "issuer", options.oidcURL, "oidcClientID", options.oidcAuthenticatorClientID, "error", err)
	}
	tokenVerifiers, tokenExtractors, err := createAuthClients(options, providers)
	if err != nil {
		log.Fatalw("failed to create auth clients", "error", err)
	}
	updateManager, err := version.NewFromFiles(options.versionsFile, options.updatesFile)
	if err != nil {
		log.Fatalw("failed to create update manager", "error", err)
	}
	credentialManager, err := credentials.NewFromFiles(options.credentialFile)
	if err != nil {
		log.Fatalw("failed to create credential manager", "error", err)
	}
	apiHandler, err := createAPIHandler(options, providers, oidcIssuerVerifier, tokenVerifiers, tokenExtractors, updateManager, credentialManager)
	if err != nil {
		log.Fatalw("failed to create API Handler", "error", err)
	}

	go metricspkg.ServeForever(options.internalAddr, "/metrics")
	log.Infow("the API server listening", "listenAddress", options.listenAddress)
	log.Fatalw("failed to start API server", "error", http.ListenAndServe(options.listenAddress, handlers.CombinedLoggingHandler(os.Stdout, apiHandler)))
}

func createInitProviders(options serverRunOptions) (providers, error) {
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
			kubermaticlog.Logger.Infow("adding seed", "seed", ctx)
			kubeClient := kubernetes.NewForConfigOrDie(cfg)
			kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, informer.DefaultInformerResyncPeriod)
			kubermaticSeedClient := kubermaticclientset.NewForConfigOrDie(cfg)
			kubermaticSeedInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticSeedClient, informer.DefaultInformerResyncPeriod)
			defaultImpersonationClientForSeed := kubernetesprovider.NewKubermaticImpersonationClient(cfg)
			seedCtrlruntimeClient, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
			if err != nil {
				return providers{}, fmt.Errorf("failed to create dynamic seed client: %v", err)
			}

			userClusterConnectionProvider, err := client.NewExternal(seedCtrlruntimeClient)
			if err != nil {
				return providers{}, fmt.Errorf("failed to get userClusterConnectionProvider: %v", err)
			}

			clusterProviders[ctx] = kubernetesprovider.NewClusterProvider(
				defaultImpersonationClientForSeed.CreateImpersonatedKubermaticClientSet,
				userClusterConnectionProvider,
				kubermaticSeedInformerFactory.Kubermatic().V1().Clusters().Lister(),
				options.workerName,
				rbac.ExtractGroupPrefix,
				seedCtrlruntimeClient,
				kubeClient,
			)

			kubeInformerFactory.Start(wait.NeverStop)
			kubeInformerFactory.WaitForCacheSync(wait.NeverStop)
			kubermaticSeedInformerFactory.Start(wait.NeverStop)
			kubermaticSeedInformerFactory.WaitForCacheSync(wait.NeverStop)
		}
	}

	masterCfg, err := clientcmd.BuildConfigFromFlags("", options.kubeconfig)
	if err != nil {
		return providers{}, fmt.Errorf("unable to build client configuration from kubeconfig due to %v", err)
	}

	// create other providers
	kubeMasterClient := kubernetes.NewForConfigOrDie(masterCfg)
	kubeMasterInformerFactory := informers.NewSharedInformerFactory(kubeMasterClient, informer.DefaultInformerResyncPeriod)
	kubermaticMasterClient := kubermaticclientset.NewForConfigOrDie(masterCfg)
	kubermaticMasterInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticMasterClient, informer.DefaultInformerResyncPeriod)
	defaultKubermaticImpersonationClient := kubernetesprovider.NewKubermaticImpersonationClient(masterCfg)
	defaultKubernetesImpersonationClient := kubernetesprovider.NewKubernetesImpersonationClient(masterCfg)

	datacenters, err := provider.LoadDatacentersMeta(options.dcFile)
	if err != nil {
		return providers{}, fmt.Errorf("failed to load datacenter yaml %q: %v", options.dcFile, err)
	}
	cloudProviders := cloud.Providers(datacenters)
	userMasterLister := kubermaticMasterInformerFactory.Kubermatic().V1().Users().Lister()
	sshKeyProvider := kubernetesprovider.NewSSHKeyProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserSSHKeys().Lister())
	userProvider := kubernetesprovider.NewUserProvider(kubermaticMasterClient, userMasterLister, kubernetesprovider.IsServiceAccount)

	serviceAccountTokenProvider, err := kubernetesprovider.NewServiceAccountTokenProvider(defaultKubernetesImpersonationClient.CreateImpersonatedKubernetesClientSet, kubeMasterInformerFactory.Core().V1().Secrets().Lister())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create service account token provider due to %v", err)
	}
	serviceAccountProvider := kubernetesprovider.NewServiceAccountProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, userMasterLister, options.domain)

	projectMemberProvider := kubernetesprovider.NewProjectMemberProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserProjectBindings().Lister(), userMasterLister, kubernetesprovider.IsServiceAccount)
	projectProvider, err := kubernetesprovider.NewProjectProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().Projects().Lister())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create project provider due to %v", err)
	}

	privilegedProjectProvider, err := kubernetesprovider.NewPrivilegedProjectProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create privileged project provider due to %v", err)
	}

	kubeMasterInformerFactory.Start(wait.NeverStop)
	kubeMasterInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubermaticMasterInformerFactory.Start(wait.NeverStop)
	kubermaticMasterInformerFactory.WaitForCacheSync(wait.NeverStop)

	eventRecorderProvider := kubernetesprovider.NewEventRecorder()

	return providers{
			sshKey:                                sshKeyProvider,
			user:                                  userProvider,
			serviceAccountProvider:                serviceAccountProvider,
			serviceAccountTokenProvider:           serviceAccountTokenProvider,
			privilegedServiceAccountTokenProvider: serviceAccountTokenProvider,
			project:                               projectProvider,
			privilegedProject:                     privilegedProjectProvider,
			projectMember:                         projectMemberProvider,
			memberMapper:                          projectMemberProvider,
			cloud:                                 cloudProviders,
			eventRecorderProvider:                 eventRecorderProvider,
			clusters:                              clusterProviders,
			datacenters:                           datacenters},
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

func createAuthClients(options serverRunOptions, prov providers) (auth.TokenVerifier, auth.TokenExtractor, error) {
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

	jwtExtractorVerifier := auth.NewServiceAccountAuthClient(
		auth.NewHeaderBearerTokenExtractor("Authorization"),
		serviceaccount.JWTTokenAuthenticator([]byte(options.serviceAccountSigningKey)),
		prov.privilegedServiceAccountTokenProvider,
	)

	tokenVerifiers := auth.NewTokenVerifierPlugins([]auth.TokenVerifier{oidcExtractorVerifier, jwtExtractorVerifier})
	tokenExtractors := auth.NewTokenExtractorPlugins([]auth.TokenExtractor{oidcExtractorVerifier, jwtExtractorVerifier})
	return tokenVerifiers, tokenExtractors, nil
}

func createAPIHandler(options serverRunOptions, prov providers, oidcIssuerVerifier auth.OIDCIssuerVerifier, tokenVerifiers auth.TokenVerifier, tokenExtractors auth.TokenExtractor, updateManager common.UpdateManager, credentialManager common.CredentialManager) (http.HandlerFunc, error) {
	var prometheusClient prometheusapi.Client
	if options.featureGates.Enabled(PrometheusEndpoint) {
		var err error
		if prometheusClient, err = prometheusapi.NewClient(prometheusapi.Config{
			Address: options.prometheusURL,
		}); err != nil {
			return nil, err
		}
	}

	serviceAccountTokenGenerator, err := serviceaccount.JWTTokenGenerator([]byte(options.serviceAccountSigningKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create service account token generator due to %v", err)
	}
	serviceAccountTokenAuth := serviceaccount.JWTTokenAuthenticator([]byte(options.serviceAccountSigningKey))

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
		serviceAccountTokenAuth,
		serviceAccountTokenGenerator,
		prov.eventRecorderProvider,
		credentialManager,
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
