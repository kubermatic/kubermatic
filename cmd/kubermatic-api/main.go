/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	gatekeeperconfigv1alpha1 "github.com/open-policy-agent/gatekeeper/apis/config/v1alpha1"
	prometheusapi "github.com/prometheus/client_golang/api"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticclientset "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned"
	kubermaticinformers "k8c.io/kubermatic/v2/pkg/crd/client/informers/externalversions"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/handler"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	v2 "k8c.io/kubermatic/v2/pkg/handler/v2"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	metricspkg "k8c.io/kubermatic/v2/pkg/metrics"
	"k8c.io/kubermatic/v2/pkg/pprof"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/util/cli"
	"k8c.io/kubermatic/v2/pkg/version"
	kuberneteswatcher "k8c.io/kubermatic/v2/pkg/watcher/kubernetes"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntime "sigs.k8s.io/controller-runtime"
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

	ctx := context.Background()
	cli.Hello(log, "API", options.log.Debug, &options.versions)

	if err := clusterv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme", zap.Stringer("api", clusterv1alpha1.SchemeGroupVersion), zap.Error(err))
	}
	if err := v1beta1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme", zap.Stringer("api", v1beta1.SchemeGroupVersion), zap.Error(err))
	}
	if err := gatekeeperconfigv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme", zap.Stringer("api", gatekeeperconfigv1alpha1.GroupVersion), zap.Error(err))
	}

	providers, err := createInitProviders(ctx, options)
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
		if err := pprofOpts.Start(ctx); err != nil {
			log.Fatalw("Failed to start pprof handler", zap.Error(err))
		}
	}()

	go metricspkg.ServeForever(options.internalAddr, "/metrics")
	log.Infow("the API server listening", "listenAddress", options.listenAddress)
	log.Fatalw("failed to start API server", "error", http.ListenAndServe(options.listenAddress, handlers.CombinedLoggingHandler(os.Stdout, apiHandler)))
}

func createInitProviders(ctx context.Context, options serverRunOptions) (providers, error) {
	masterCfg, err := ctrlruntime.GetConfig()
	if err != nil {
		return providers{}, fmt.Errorf("unable to build client configuration from kubeconfig due to %v", err)
	}

	// create other providers
	kubeMasterClient := kubernetes.NewForConfigOrDie(masterCfg)
	kubeMasterInformerFactory := informers.NewSharedInformerFactory(kubeMasterClient, 30*time.Minute)
	kubermaticMasterClient := kubermaticclientset.NewForConfigOrDie(masterCfg)
	kubermaticMasterInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticMasterClient, 30*time.Minute)

	// We use the manager only to get a lister-backed ctrlruntimeclient.Client. We can not use it for most
	// other actions, because it doesn't support impersonation (and can't be changed to do that as that would mean it has to replicate the apiservers RBAC for the lister)
	mgr, err := manager.New(masterCfg, manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		return providers{}, fmt.Errorf("failed to construct manager: %v", err)
	}

	client := mgr.GetClient()

	defaultImpersonationClient := kubernetesprovider.NewImpersonationClient(masterCfg, mgr.GetRESTMapper())

	seedsGetter, err := seedsGetterFactory(ctx, client, options)
	if err != nil {
		return providers{}, err
	}
	seedKubeconfigGetter, err := seedKubeconfigGetterFactory(ctx, client, options)
	if err != nil {
		return providers{}, err
	}

	// Make sure the manager creates a cache for Seeds by requesting an informer
	if _, err := mgr.GetCache().GetInformer(ctx, &kubermaticv1.Seed{}); err != nil {
		kubermaticlog.Logger.Fatalw("failed to get seed informer", zap.Error(err))
	}
	// mgr.Start() is blocking
	go func() {
		if err := mgr.Start(ctx); err != nil {
			kubermaticlog.Logger.Fatalw("failed to start the mgr", zap.Error(err))
		}
	}()
	mgrSyncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if synced := mgr.GetCache().WaitForCacheSync(mgrSyncCtx); !synced {
		kubermaticlog.Logger.Fatal("failed to sync mgr cache")
	}

	seedClientGetter := provider.SeedClientGetterFactory(seedKubeconfigGetter)
	clusterProviderGetter := clusterProviderFactory(mgr.GetRESTMapper(), seedKubeconfigGetter, seedClientGetter, options)

	presetsProvider, err := kubernetesprovider.NewPresetsProvider(ctx, client, options.presetsFile, options.dynamicPresets)
	if err != nil {
		return providers{}, err
	}
	admissionPluginProvider := kubernetesprovider.NewAdmissionPluginsProvider(ctx, client)
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

	sshKeyProvider := kubernetesprovider.NewSSHKeyProvider(defaultImpersonationClient.CreateImpersonatedClient, client)
	privilegedSSHKeyProvider, err := kubernetesprovider.NewPrivilegedSSHKeyProvider(client)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create privileged SSH key provider due to %v", err)
	}
	userProvider := kubernetesprovider.NewUserProvider(client, kubernetesprovider.IsProjectServiceAccount, kubermaticMasterClient)
	settingsProvider := kubernetesprovider.NewSettingsProvider(ctx, kubermaticMasterClient, client)
	addonConfigProvider := kubernetesprovider.NewAddonConfigProvider(client)
	adminProvider := kubernetesprovider.NewAdminProvider(client)

	serviceAccountTokenProvider, err := kubernetesprovider.NewServiceAccountTokenProvider(defaultImpersonationClient.CreateImpersonatedClient, client)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create service account token provider due to %v", err)
	}

	serviceAccountProvider := kubernetesprovider.NewServiceAccountProvider(defaultImpersonationClient.CreateImpersonatedClient, client, options.domain)
	projectMemberProvider := kubernetesprovider.NewProjectMemberProvider(defaultImpersonationClient.CreateImpersonatedClient, client, kubernetesprovider.IsProjectServiceAccount)
	projectProvider, err := kubernetesprovider.NewProjectProvider(defaultImpersonationClient.CreateImpersonatedClient, client)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create project provider due to %v", err)
	}

	privilegedProjectProvider, err := kubernetesprovider.NewPrivilegedProjectProvider(client)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create privileged project provider due to %v", err)
	}

	userInfoGetter, err := provider.UserInfoGetterFactory(projectMemberProvider)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create user info getter due to %v", err)
	}

	externalClusterProvider, err := kubernetesprovider.NewExternalClusterProvider(defaultImpersonationClient.CreateImpersonatedClient, mgr.GetClient())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create external cluster provider due to %v", err)
	}

	defaultConstraintProvider, err := kubernetesprovider.NewDefaultConstraintProvider(defaultImpersonationClient.CreateImpersonatedClient, mgr.GetClient(), options.namespace)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create default constraint provider due to %w", err)
	}

	constraintTemplateProvider, err := kubernetesprovider.NewConstraintTemplateProvider(defaultImpersonationClient.CreateImpersonatedClient, mgr.GetClient())
	if err != nil {
		return providers{}, fmt.Errorf("failed to create constraint template provider due to %v", err)
	}

	clusterTemplateProvider, err := kubernetesprovider.NewClusterTemplateProvider(defaultImpersonationClient.CreateImpersonatedClient, client)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create cluster template provider due to %v", err)
	}

	constraintProviderGetter := kubernetesprovider.ConstraintProviderFactory(mgr.GetRESTMapper(), seedKubeconfigGetter)

	kubeMasterInformerFactory.Start(wait.NeverStop)
	kubeMasterInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubermaticMasterInformerFactory.Start(wait.NeverStop)
	kubermaticMasterInformerFactory.WaitForCacheSync(wait.NeverStop)

	eventRecorderProvider := kubernetesprovider.NewEventRecorder()

	addonProviderGetter := kubernetesprovider.AddonProviderFactory(mgr.GetRESTMapper(), seedKubeconfigGetter, options.accessibleAddons)

	alertmanagerProviderGetter := kubernetesprovider.AlertmanagerProviderFactory(mgr.GetRESTMapper(), seedKubeconfigGetter)

	ruleGroupProviderGetter := kubernetesprovider.RuleGroupProviderFactory(mgr.GetRESTMapper(), seedKubeconfigGetter)

	settingsWatcher, err := kuberneteswatcher.NewSettingsWatcher(settingsProvider)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create settings watcher due to %v", err)
	}

	userWatcher, err := kuberneteswatcher.NewUserWatcher(userProvider)
	if err != nil {
		return providers{}, fmt.Errorf("failed to create user watcher due to %v", err)
	}

	return providers{
		sshKey:                                sshKeyProvider,
		privilegedSSHKeyProvider:              privilegedSSHKeyProvider,
		user:                                  userProvider,
		serviceAccountProvider:                serviceAccountProvider,
		privilegedServiceAccountProvider:      serviceAccountProvider,
		serviceAccountTokenProvider:           serviceAccountTokenProvider,
		privilegedServiceAccountTokenProvider: serviceAccountTokenProvider,
		project:                               projectProvider,
		privilegedProject:                     privilegedProjectProvider,
		projectMember:                         projectMemberProvider,
		privilegedProjectMemberProvider:       projectMemberProvider,
		memberMapper:                          projectMemberProvider,
		eventRecorderProvider:                 eventRecorderProvider,
		clusterProviderGetter:                 clusterProviderGetter,
		seedsGetter:                           seedsGetter,
		seedClientGetter:                      seedClientGetter,
		addons:                                addonProviderGetter,
		addonConfigProvider:                   addonConfigProvider,
		userInfoGetter:                        userInfoGetter,
		settingsProvider:                      settingsProvider,
		adminProvider:                         adminProvider,
		presetProvider:                        presetsProvider,
		admissionPluginProvider:               admissionPluginProvider,
		settingsWatcher:                       settingsWatcher,
		userWatcher:                           userWatcher,
		externalClusterProvider:               externalClusterProvider,
		privilegedExternalClusterProvider:     externalClusterProvider,
		constraintTemplateProvider:            constraintTemplateProvider,
		defaultConstraintProvider:             defaultConstraintProvider,
		constraintProviderGetter:              constraintProviderGetter,
		alertmanagerProviderGetter:            alertmanagerProviderGetter,
		clusterTemplateProvider:               clusterTemplateProvider,
		ruleGroupProviderGetter:               ruleGroupProviderGetter,
	}, nil
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
		options.caBundle.CertPool(),
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
		options.caBundle.CertPool(),
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

	routingParams := handler.RoutingParams{
		Log:                                   kubermaticlog.New(options.log.Debug, options.log.Format).Sugar(),
		PresetsProvider:                       prov.presetProvider,
		SeedsGetter:                           prov.seedsGetter,
		SeedsClientGetter:                     prov.seedClientGetter,
		SSHKeyProvider:                        prov.sshKey,
		PrivilegedSSHKeyProvider:              prov.privilegedSSHKeyProvider,
		UserProvider:                          prov.user,
		ServiceAccountProvider:                prov.serviceAccountProvider,
		PrivilegedServiceAccountProvider:      prov.privilegedServiceAccountProvider,
		ServiceAccountTokenProvider:           prov.serviceAccountTokenProvider,
		PrivilegedServiceAccountTokenProvider: prov.privilegedServiceAccountTokenProvider,
		ProjectProvider:                       prov.project,
		PrivilegedProjectProvider:             prov.privilegedProject,
		OIDCIssuerVerifier:                    oidcIssuerVerifier,
		TokenVerifiers:                        tokenVerifiers,
		TokenExtractors:                       tokenExtractors,
		ClusterProviderGetter:                 prov.clusterProviderGetter,
		AddonProviderGetter:                   prov.addons,
		AddonConfigProvider:                   prov.addonConfigProvider,
		UpdateManager:                         updateManager,
		PrometheusClient:                      prometheusClient,
		ProjectMemberProvider:                 prov.projectMember,
		PrivilegedProjectMemberProvider:       prov.privilegedProjectMemberProvider,
		UserProjectMapper:                     prov.memberMapper,
		SATokenAuthenticator:                  serviceAccountTokenAuth,
		SATokenGenerator:                      serviceAccountTokenGenerator,
		EventRecorderProvider:                 prov.eventRecorderProvider,
		ExposeStrategy:                        options.exposeStrategy,
		AccessibleAddons:                      options.accessibleAddons,
		UserInfoGetter:                        prov.userInfoGetter,
		SettingsProvider:                      prov.settingsProvider,
		AdminProvider:                         prov.adminProvider,
		AdmissionPluginProvider:               prov.admissionPluginProvider,
		SettingsWatcher:                       prov.settingsWatcher,
		UserWatcher:                           prov.userWatcher,
		ExternalClusterProvider:               prov.externalClusterProvider,
		PrivilegedExternalClusterProvider:     prov.privilegedExternalClusterProvider,
		ConstraintTemplateProvider:            prov.constraintTemplateProvider,
		DefaultConstraintProvider:             prov.defaultConstraintProvider,
		ConstraintProviderGetter:              prov.constraintProviderGetter,
		AlertmanagerProviderGetter:            prov.alertmanagerProviderGetter,
		ClusterTemplateProvider:               prov.clusterTemplateProvider,
		RuleGroupProviderGetter:               prov.ruleGroupProviderGetter,
		Versions:                              options.versions,
		CABundle:                              options.caBundle.CertPool(),
	}

	r := handler.NewRouting(routingParams)
	rv2 := v2.NewV2Routing(routingParams)

	registerMetrics()

	mainRouter := mux.NewRouter()
	mainRouter.Use(setSecureHeaders)
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	v2Router := mainRouter.PathPrefix("/api/v2").Subrouter()
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
	r.RegisterV1Websocket(v1Router)
	rv2.RegisterV2(v2Router, metrics)

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

func clusterProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter, seedClientGetter provider.SeedClientGetter, options serverRunOptions) provider.ClusterProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.ClusterProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		kubeClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubeClient: %v", err)
		}
		defaultImpersonationClientForSeed := kubernetesprovider.NewImpersonationClient(cfg, mapper)

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
			defaultImpersonationClientForSeed.CreateImpersonatedClient,
			userClusterConnectionProvider,
			options.workerName,
			rbac.ExtractGroupPrefix,
			seedCtrlruntimeClient,
			kubeClient,
			options.featureGates.Enabled(features.OIDCKubeCfgEndpoint),
			options.versions,
		), nil
	}
}
