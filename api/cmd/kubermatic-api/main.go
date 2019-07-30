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
//     Version: 2.11
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
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	metricspkg "github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/presets"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"github.com/kubermatic/kubermatic/api/pkg/util/informer"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

	if err := kubermaticv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to add kubermaticv1 scheme to scheme.Scheme", "error", err)
	}

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
	presetsManager, err := presets.NewFromFile(options.presetsFile)
	if err != nil {
		log.Fatalw("failed to create presets manager", "error", err)
	}
	apiHandler, err := createAPIHandler(options, providers, oidcIssuerVerifier, tokenVerifiers, tokenExtractors, updateManager, presetsManager)
	if err != nil {
		log.Fatalw("failed to create API Handler", "error", err)
	}

	go metricspkg.ServeForever(options.internalAddr, "/metrics")
	log.Infow("the API server listening", "listenAddress", options.listenAddress)
	log.Fatalw("failed to start API server", "error", http.ListenAndServe(options.listenAddress, handlers.CombinedLoggingHandler(os.Stdout, apiHandler)))
}

func createInitProviders(options serverRunOptions) (providers, error) {
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

	// We use the manager only to get a lister-backed ctrlruntimeclient.Client. We can not use it for most
	// other actions, because it doesn't support impersonation (and cant be changed to do that as that would mean it has to replicate the apiservers RBAC for the lister)
	mgr, err := manager.New(masterCfg, manager.Options{MetricsBindAddress: "0", LeaderElection: false})
	if err != nil {
		return providers{}, fmt.Errorf("failed to construct manager: %v", err)
	}
	seedsGetter, err := provider.SeedsGetterFactory(context.Background(), mgr.GetClient(), options.dcFile, options.workerName, options.dynamicDatacenters)
	if err != nil {
		return providers{}, err
	}
	seedKubeconfigGetter, err := provider.SeedKubeconfigGetterFactory(context.Background(), mgr.GetClient(), options.kubeconfig, options.dynamicDatacenters)
	if err != nil {
		return providers{}, err
	}
	clusterProviderGetter := clusterProviderFactory(seedKubeconfigGetter, options.workerName, options.featureGates.Enabled(OIDCKubeCfgEndpoint))

	userMasterLister := kubermaticMasterInformerFactory.Kubermatic().V1().Users().Lister()
	sshKeyProvider := kubernetesprovider.NewSSHKeyProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserSSHKeys().Lister())
	userProvider := kubernetesprovider.NewUserProvider(kubermaticMasterClient, userMasterLister, kubernetesprovider.IsServiceAccount)

	serviceAccountTokenProvider, err := kubernetesprovider.NewServiceAccountTokenProvider(defaultKubernetesImpersonationClient.CreateImpersonatedKubernetesClientSet, kubeMasterInformerFactory.Core().V1().Secrets().Lister())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create service account token provider due to %v", err)
	}
	serviceAccountProvider := kubernetesprovider.NewServiceAccountProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, userMasterLister, options.domain)

	credentialsProvider, err := kubernetesprovider.NewCredentialsProvider(defaultKubernetesImpersonationClient.CreateImpersonatedKubernetesClientSet, kubeMasterInformerFactory.Core().V1().Secrets().Lister())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create credentials provider due to %v", err)
	}

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
		eventRecorderProvider:                 eventRecorderProvider,
		clusterProviderGetter:                 clusterProviderGetter,
		seedsGetter:                           seedsGetter,
		credentialsProvider:                   credentialsProvider}, nil
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

func createAPIHandler(options serverRunOptions, prov providers, oidcIssuerVerifier auth.OIDCIssuerVerifier, tokenVerifiers auth.TokenVerifier, tokenExtractors auth.TokenExtractor, updateManager common.UpdateManager, presetsManager common.PresetsManager) (http.HandlerFunc, error) {
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
		prov.seedsGetter,
		prov.clusterProviderGetter,
		prov.sshKey,
		prov.user,
		prov.serviceAccountProvider,
		prov.serviceAccountTokenProvider,
		prov.project,
		prov.privilegedProject,
		prov.credentialsProvider,
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
		presetsManager,
		options.exposeStrategy,
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

	mainRouter.Methods(http.MethodGet).
		Path("/api/swagger.json").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, options.swaggerFile)
		})

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

func clusterProviderFactory(seedKubeconfigGetter provider.SeedKubeconfigGetter, workerName string, oidcKubeCfgEndpointEnabled bool) provider.ClusterProviderGetter {
	// Discovery can take very long when the cluster is not physically close, so cache the discovery info
	mapperLock := &sync.Mutex{}
	restMapperCache := map[string]meta.RESTMapper{}

	return func(seedName string) (provider.ClusterProvider, error) {
		cfg, err := seedKubeconfigGetter(seedName)
		if err != nil {
			return nil, err
		}
		kubeClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("faild to create kubeClient: %v", err)
		}
		defaultImpersonationClientForSeed := kubernetesprovider.NewKubermaticImpersonationClient(cfg)

		// Make sure this changes if someone uses the same name for a differt seed
		mapperKey := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s/%s",
			seedName, cfg.Host, cfg.APIPath, cfg.Username, cfg.Password, cfg.BearerToken, cfg.BearerTokenFile,
			string(cfg.CertData), string(cfg.KeyData), string(cfg.CAData))

		// We must aquire the lock here to not get panics for concurrent map access
		mapperLock.Lock()
		defer mapperLock.Unlock()
		if _, exists := restMapperCache[mapperKey]; !exists {
			restMapperCache[mapperKey], err = restmapper.NewDynamicRESTMapper(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create restMapper for seed %q: %v", seedName, err)
			}
			kubermaticlog.Logger.With("seed", seedName).Info("Created restMapper")
		}
		mapper := restMapperCache[mapperKey]
		seedCtrlruntimeClient, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamic seed client: %v", err)
		}

		userClusterConnectionProvider, err := client.NewExternal(seedCtrlruntimeClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get userClusterConnectionProvider: %v", err)
		}

		return kubernetesprovider.NewClusterProvider(
			defaultImpersonationClientForSeed.CreateImpersonatedKubermaticClientSet,
			userClusterConnectionProvider,
			workerName,
			rbac.ExtractGroupPrefix,
			seedCtrlruntimeClient,
			kubeClient,
			oidcKubeCfgEndpointEnabled,
		), nil
	}
}
