// Package classification Kubermatic API.
//
// Kubermatic API
//
// This describes possible operations which can be made against the Kubermatic API.
//
// Terms Of Service:
//
// There are no TOS at this moment, use at your own risk we take no responsibility
//
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
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"
	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/features"
	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	metricspkg "github.com/kubermatic/kubermatic/api/pkg/metrics"
	"github.com/kubermatic/kubermatic/api/pkg/pprof"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	klog.InitFlags(nil)
	pprofOpts := &pprof.Opts{}
	pprofOpts.AddFlags(flag.CommandLine)
	options, err := newServerRunOptions()
	if err != nil {
		fmt.Printf("failed to create server run options due to = %v\n", err)
		os.Exit(1)
	}
	if err := options.validate(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rawLog := kubermaticlog.New(options.log.Debug, options.log.Format)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()
	kubermaticlog.Logger = log

	if err := clusterv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := v1beta1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme", zap.Stringer("api", v1beta1.SchemeGroupVersion), zap.Error(err))
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
	apiHandler, err := createAPIHandler(options, providers, oidcIssuerVerifier, tokenVerifiers, tokenExtractors, updateManager)
	if err != nil {
		log.Fatalw("failed to create API Handler", "error", err)
	}

	go func() {
		if err := pprofOpts.Start(make(chan struct{})); err != nil {
			log.Fatalw("Failed to start pprof handler", zap.Error(err))
		}
	}()

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
	kubeMasterInformerFactory := informers.NewSharedInformerFactory(kubeMasterClient, 30*time.Minute)
	kubermaticMasterClient := kubermaticclientset.NewForConfigOrDie(masterCfg)
	kubermaticMasterInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticMasterClient, 30*time.Minute)
	defaultKubermaticImpersonationClient := kubernetesprovider.NewKubermaticImpersonationClient(masterCfg)
	defaultKubernetesImpersonationClient := kubernetesprovider.NewKubernetesImpersonationClient(masterCfg)

	// We use the manager only to get a lister-backed ctrlruntimeclient.Client. We can not use it for most
	// other actions, because it doesn't support impersonation (and cant be changed to do that as that would mean it has to replicate the apiservers RBAC for the lister)
	mgr, err := manager.New(masterCfg, manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		return providers{}, fmt.Errorf("failed to construct manager: %v", err)
	}
	seedsGetter, err := provider.SeedsGetterFactory(context.Background(), mgr.GetClient(), options.dcFile, options.namespace, options.dynamicDatacenters)
	if err != nil {
		return providers{}, err
	}
	seedKubeconfigGetter, err := provider.SeedKubeconfigGetterFactory(context.Background(), mgr.GetClient(), options.kubeconfig, options.namespace, options.dynamicDatacenters)
	if err != nil {
		return providers{}, err
	}

	// Make sure the manager creates a cache for Seeds by requesting an informer
	if _, err := mgr.GetCache().GetInformer(&kubermaticv1.Seed{}); err != nil {
		kubermaticlog.Logger.Fatalw("failed to get seed informer", zap.Error(err))
	}
	// mgr.Start() is blocking
	go func() {
		if err := mgr.Start(wait.NeverStop); err != nil {
			kubermaticlog.Logger.Fatalw("failed to start the mgr", zap.Error(err))
		}
	}()
	mgrSyncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if synced := mgr.GetCache().WaitForCacheSync(mgrSyncCtx.Done()); !synced {
		kubermaticlog.Logger.Fatal("failed to sync mgr cache")
	}

	seedClientGetter := provider.SeedClientGetterFactory(seedKubeconfigGetter)
	clusterProviderGetter := clusterProviderFactory(seedKubeconfigGetter, seedClientGetter, options.workerName, options.featureGates.Enabled(features.OIDCKubeCfgEndpoint))

	presetsProvider, err := kubernetesprovider.NewPresetsProvider(context.Background(), mgr.GetClient(), options.presetsFile, options.dynamicPresets)
	if err != nil {
		return providers{}, err
	}
	// Warm up the restMapper cache. Log but ignore errors encountered here, maybe there are stale seeds
	go func() {
		seeds, err := seedsGetter()
		if err != nil {
			kubermaticlog.Logger.Infow("failed to get seeds when trying to warm up restMapper cache", zap.Error(err))
			return
		}
		for _, seed := range seeds {
			if _, err := clusterProviderGetter(seed); err != nil {
				kubermaticlog.Logger.Infow("failed to get clusterProvider when trying to warm up restMapper cache", zap.Error(err), "seed", seed.Name)
			}
		}
	}()

	userMasterLister := kubermaticMasterInformerFactory.Kubermatic().V1().Users().Lister()
	sshKeyProvider := kubernetesprovider.NewSSHKeyProvider(defaultKubermaticImpersonationClient.CreateImpersonatedKubermaticClientSet, kubermaticMasterInformerFactory.Kubermatic().V1().UserSSHKeys().Lister())
	userProvider := kubernetesprovider.NewUserProvider(kubermaticMasterClient, userMasterLister, kubernetesprovider.IsServiceAccount)
	settingsProvider := kubernetesprovider.NewSettingsProvider(kubermaticMasterClient, kubermaticMasterInformerFactory.Kubermatic().V1().KubermaticSettings().Lister())
	addonConfigProvider := kubernetesprovider.NewAddonConfigProvider(kubermaticMasterClient, kubermaticMasterInformerFactory.Kubermatic().V1().AddonConfigs().Lister())
	adminProvider := kubernetesprovider.NewAdminProvider(kubermaticMasterClient, userMasterLister)

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

	userInfoGetter, err := provider.UserInfoGetterFactory(projectMemberProvider)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create user info getter due to %v", err)
	}

	kubeMasterInformerFactory.Start(wait.NeverStop)
	kubeMasterInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubermaticMasterInformerFactory.Start(wait.NeverStop)
	kubermaticMasterInformerFactory.WaitForCacheSync(wait.NeverStop)

	eventRecorderProvider := kubernetesprovider.NewEventRecorder()

	addonProviderGetter := kubernetesprovider.AddonProviderFactory(seedKubeconfigGetter, options.accessibleAddons)

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
		addons:                                addonProviderGetter,
		addonConfigProvider:                   addonConfigProvider,
		userInfoGetter:                        userInfoGetter,
		settingsProvider:                      settingsProvider,
		adminProvider:                         adminProvider,
		presetProvider:                        presetsProvider}, nil
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
			auth.NewCookieHeaderBearerTokenExtractor("token"),
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

func createAPIHandler(options serverRunOptions, prov providers, oidcIssuerVerifier auth.OIDCIssuerVerifier, tokenVerifiers auth.TokenVerifier, tokenExtractors auth.TokenExtractor, updateManager common.UpdateManager) (http.HandlerFunc, error) {
	var prometheusClient prometheusapi.Client
	if options.featureGates.Enabled(features.PrometheusEndpoint) {
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
		kubermaticlog.New(options.log.Debug, options.log.Format).Sugar(),
		prov.presetProvider,
		prov.seedsGetter,
		prov.clusterProviderGetter,
		prov.addons,
		prov.addonConfigProvider,
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
		options.exposeStrategy,
		options.accessibleAddons,
		prov.userInfoGetter,
		prov.settingsProvider,
		prov.adminProvider,
	)

	registerMetrics()

	mainRouter := mux.NewRouter()
	mainRouter.Use(setSecureHeaders)
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	r.RegisterV1(v1Router, metrics)
	r.RegisterV1Legacy(v1Router)
	r.RegisterV1Optional(v1Router,
		options.featureGates.Enabled(features.OIDCKubeCfgEndpoint),
		common.OIDCConfiguration{
			URL:                  options.oidcURL,
			ClientID:             options.oidcIssuerClientID,
			ClientSecret:         options.oidcIssuerClientSecret,
			CookieHashKey:        options.oidcIssuerCookieHashKey,
			CookieSecureMode:     options.oidcIssuerCookieSecureMode,
			OfflineAccessAsScope: options.oidcIssuerOfflineAccessAsScope,
		},
		mainRouter)
	r.RegisterV1Admin(v1Router)

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

func setSecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ContentSecurityPolicy sets the `Content-Security-Policy` header providing
		// security against cross-site scripting (XSS), clickjacking and other code
		// injection attacks resulting from execution of malicious content in the
		// trusted web page context. Reference: https://w3c.github.io/webappsec-csp/
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'self'; style-src 'self'; img-src 'self'; media-src 'self'; frame-ancestors 'self'; frame-src 'self'; connect-src 'self'")
		// XFrameOptions can be used to indicate whether or not a browser should
		// be allowed to render a page in a <frame>, <iframe> or <object> .
		// Sites can use this to avoid clickjacking attacks, by ensuring that their
		// content is not embedded into other sites.provides protection against
		// clickjacking.
		// Optional. Default value "SAMEORIGIN".
		// Possible values:
		// - "SAMEORIGIN" - The page can only be displayed in a frame on the same origin as the page itself.
		// - "DENY" - The page cannot be displayed in a frame, regardless of the site attempting to do so.
		// - "ALLOW-FROM uri" - The page can only be displayed in a frame on the specified origin.
		w.Header().Set("X-Frame-Options", "DENY")
		// XSSProtection provides protection against cross-site scripting attack (XSS)
		// by setting the `X-XSS-Protection` header.
		// Optional. Default value "1; mode=block".
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// ContentTypeNosniff provides protection against overriding Content-Type
		// header by setting the `X-Content-Type-Options` header.
		// Optional. Default value "nosniff".
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func clusterProviderFactory(seedKubeconfigGetter provider.SeedKubeconfigGetter, seedClientGetter provider.SeedClientGetter, workerName string, oidcKubeCfgEndpointEnabled bool) provider.ClusterProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.ClusterProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		kubeClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("faild to create kubeClient: %v", err)
		}
		defaultImpersonationClientForSeed := kubernetesprovider.NewKubermaticImpersonationClient(cfg)

		seedCtrlruntimeClient, err := seedClientGetter(seed)
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamic seed client: %v", err)
		}

		userClusterConnectionProvider, err := client.NewExternal(seedCtrlruntimeClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get userClusterConnectionProvider: %v", err)
		}

		return kubernetesprovider.NewClusterProvider(
			cfg,
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
