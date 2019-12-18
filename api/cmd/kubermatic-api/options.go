package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/features"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "k8s.io/api/core/v1"
)

type serverRunOptions struct {
	listenAddress      string
	kubeconfig         string
	internalAddr       string
	prometheusURL      string
	masterResources    string
	dcFile             string
	workerName         string
	versionsFile       string
	updatesFile        string
	presetsFile        string
	swaggerFile        string
	domain             string
	exposeStrategy     corev1.ServiceType
	dynamicDatacenters bool
	dynamicPresets     bool
	namespace          string
	log                kubermaticlog.Options
	accessibleAddons   sets.String

	// OIDC configuration
	oidcURL                        string
	oidcAuthenticatorClientID      string
	oidcIssuerClientID             string
	oidcIssuerClientSecret         string
	oidcIssuerRedirectURI          string
	oidcIssuerCookieHashKey        string
	oidcIssuerCookieSecureMode     bool
	oidcSkipTLSVerify              bool
	oidcIssuerOfflineAccessAsScope bool

	//service account configuration
	serviceAccountSigningKey string

	featureGates features.FeatureGate
}

func newServerRunOptions() (serverRunOptions, error) {
	s := serverRunOptions{}
	var (
		rawFeatureGates     string
		rawExposeStrategy   string
		rawAccessibleAddons string
	)

	s.log = kubermaticlog.NewDefaultOptions()
	s.log.AddFlags(flag.CommandLine)

	flag.StringVar(&s.listenAddress, "address", ":8080", "The address to listen on")
	flag.StringVar(&s.kubeconfig, "kubeconfig", "", "Path to the kubeconfig.")
	flag.StringVar(&s.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal handler should be exposed")
	flag.StringVar(&s.prometheusURL, "prometheus-url", "http://prometheus.monitoring.svc.local:web", "The URL on which this API can talk to Prometheus")
	flag.StringVar(&s.masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&s.dcFile, "datacenters", "", "The datacenters.yaml file path")
	flag.StringVar(&s.workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	flag.StringVar(&s.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&s.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.StringVar(&s.presetsFile, "presets", "", "The optional file path for a file containing presets")
	flag.StringVar(&s.swaggerFile, "swagger", "./cmd/kubermatic-api/swagger.json", "The swagger.json file path")
	flag.StringVar(&rawAccessibleAddons, "accessible-addons", "", "Comma-separated list of user cluster addons to expose via the API")
	flag.StringVar(&s.oidcURL, "oidc-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.BoolVar(&s.oidcSkipTLSVerify, "oidc-skip-tls-verify", false, "Skip TLS verification for the token issuer")
	flag.StringVar(&s.oidcAuthenticatorClientID, "oidc-authenticator-client-id", "", "Authenticator client ID")
	flag.StringVar(&s.oidcIssuerClientID, "oidc-issuer-client-id", "", "Issuer client ID")
	flag.StringVar(&s.oidcIssuerClientSecret, "oidc-issuer-client-secret", "", "OpenID client secret")
	flag.StringVar(&s.oidcIssuerRedirectURI, "oidc-issuer-redirect-uri", "", "Callback URL for OpenID responses.")
	flag.StringVar(&s.oidcIssuerCookieHashKey, "oidc-issuer-cookie-hash-key", "", "Hash key authenticates the cookie value using HMAC. It is recommended to use a key with 32 or 64 bytes.")
	flag.BoolVar(&s.oidcIssuerCookieSecureMode, "oidc-issuer-cookie-secure-mode", true, "When true cookie received only with HTTPS. Set false for local deployment with HTTP")
	flag.BoolVar(&s.oidcIssuerOfflineAccessAsScope, "oidc-issuer-offline-access-as-scope", true, "Set it to false if OIDC provider requires to set \"access_type=offline\" query param when accessing the refresh token")
	flag.StringVar(&rawFeatureGates, "feature-gates", "", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&s.domain, "domain", "localhost", "A domain name on which the server is deployed")
	flag.StringVar(&s.serviceAccountSigningKey, "service-account-signing-key", "", "Signing key authenticates the service account's token value using HMAC. It is recommended to use a key with 32 bytes or longer.")
	flag.StringVar(&rawExposeStrategy, "expose-strategy", "NodePort", "The strategy to expose the controlplane with, either \"NodePort\" which creates NodePorts with a \"nodeport-proxy.k8s.io/expose: true\" annotation or \"LoadBalancer\", which creates a LoadBalancer")
	flag.BoolVar(&s.dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters")
	flag.BoolVar(&s.dynamicPresets, "dynamic-presets", false, "Whether to enable dynamic presets")
	flag.StringVar(&s.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources")
	flag.Parse()

	featureGates, err := features.NewFeatures(rawFeatureGates)
	if err != nil {
		return s, err
	}
	s.featureGates = featureGates

	switch rawExposeStrategy {
	case "NodePort":
		s.exposeStrategy = corev1.ServiceTypeNodePort
	case "LoadBalancer":
		s.exposeStrategy = corev1.ServiceTypeLoadBalancer
	default:
		return s, fmt.Errorf("--expose-strategy must be either `NodePort` or `LoadBalancer`, got %q", rawExposeStrategy)
	}

	s.accessibleAddons = sets.NewString(strings.Split(rawAccessibleAddons, ",")...)
	s.accessibleAddons.Delete("")

	return s, nil
}

func (o serverRunOptions) validate() error {
	// OpenShift always requires those flags, but as long as OpenShift support is not stable/testable
	// we only validate them when the OIDCKubeCfgEndpoint feature flag is set (Kubernetes specific).
	// Otherwise we force users to set those flags without any result (for Kubernetes clusters)
	// TODO: Enforce validation as soon as OpenShift support is testable
	if o.featureGates.Enabled(features.OIDCKubeCfgEndpoint) {
		if len(o.oidcIssuerClientSecret) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-client-secret\" flag was not specified", features.OIDCKubeCfgEndpoint)
		}
		if len(o.oidcIssuerRedirectURI) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-redirect-uri\" flag was not specified", features.OIDCKubeCfgEndpoint)
		}
		if len(o.oidcIssuerCookieHashKey) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-cookie-hash-key\" flag was not specified", features.OIDCKubeCfgEndpoint)
		}
		if len(o.oidcIssuerClientID) == 0 {
			return fmt.Errorf("%s feature is enabled but \"oidc-issuer-client-id\" flag was not specified", features.OIDCKubeCfgEndpoint)
		}
	}

	if err := serviceaccount.ValidateKey([]byte(o.serviceAccountSigningKey)); err != nil {
		return fmt.Errorf("the service-account-signing-key is incorrect due to error: %v", err)
	}

	return nil
}

type providers struct {
	sshKey                                provider.SSHKeyProvider
	user                                  provider.UserProvider
	serviceAccountProvider                provider.ServiceAccountProvider
	serviceAccountTokenProvider           provider.ServiceAccountTokenProvider
	privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider
	project                               provider.ProjectProvider
	privilegedProject                     provider.PrivilegedProjectProvider
	projectMember                         provider.ProjectMemberProvider
	memberMapper                          provider.ProjectMemberMapper
	eventRecorderProvider                 provider.EventRecorderProvider
	clusterProviderGetter                 provider.ClusterProviderGetter
	seedsGetter                           provider.SeedsGetter
	addons                                provider.AddonProviderGetter
	addonConfigProvider                   provider.AddonConfigProvider
	userInfoGetter                        provider.UserInfoGetter
	settingsProvider                      provider.SettingsProvider
	adminProvider                         provider.AdminProvider
	presetProvider                        provider.PresetProvider
}
