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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/features"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/kubermatic/v2/pkg/watcher"

	"k8s.io/apimachinery/pkg/util/sets"
)

type serverRunOptions struct {
	listenAddress                 string
	internalAddr                  string
	prometheusURL                 string
	masterResources               string
	workerName                    string
	versionsFile                  string
	updatesFile                   string
	providerIncompatibilitiesFile string
	presetsFile                   string
	swaggerFile                   string
	domain                        string
	exposeStrategy                kubermaticv1.ExposeStrategy
	dynamicPresets                bool
	namespace                     string
	log                           kubermaticlog.Options
	accessibleAddons              sets.String
	caBundle                      *certificates.CABundle

	// for development purposes, a local configuration file
	// can be used to provide the KubermaticConfiguration
	kubermaticConfiguration *operatorv1alpha1.KubermaticConfiguration

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

	// service account configuration
	serviceAccountSigningKey string

	featureGates features.FeatureGate
	versions     kubermatic.Versions
}

func newServerRunOptions() (serverRunOptions, error) {
	s := serverRunOptions{featureGates: features.FeatureGate{}}
	var (
		rawExposeStrategy   string
		rawAccessibleAddons string
		caBundleFile        string
		configFile          string
	)

	s.log = kubermaticlog.NewDefaultOptions()
	s.log.AddFlags(flag.CommandLine)

	flag.StringVar(&s.listenAddress, "address", ":8080", "The address to listen on")
	flag.StringVar(&s.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal handler should be exposed")
	flag.StringVar(&s.prometheusURL, "prometheus-url", "http://prometheus.monitoring.svc.local:web", "The URL on which this API can talk to Prometheus")
	flag.StringVar(&s.masterResources, "master-resources", "", "The path to the master resources (Required).")
	flag.StringVar(&s.workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	flag.StringVar(&s.versionsFile, "versions", "versions.yaml", "The versions.yaml file path")
	flag.StringVar(&s.updatesFile, "updates", "updates.yaml", "The updates.yaml file path")
	flag.StringVar(&s.providerIncompatibilitiesFile, "provider-incompatibilities", "provider-incompatibilities.yaml", "The provider-incompatibilities.yaml file path")
	flag.StringVar(&s.presetsFile, "presets", "", "The optional file path for a file containing presets")
	flag.StringVar(&s.swaggerFile, "swagger", "./cmd/kubermatic-api/swagger.json", "The swagger.json file path")
	flag.StringVar(&rawAccessibleAddons, "accessible-addons", "", "Comma-separated list of user cluster addons to expose via the API")
	flag.StringVar(&caBundleFile, "ca-bundle", "", "The path to the certificate for the CA that signed your identity provider’s web certificate.")
	flag.StringVar(&s.oidcURL, "oidc-url", "", "URL of the OpenID token issuer. Example: http://auth.int.kubermatic.io")
	flag.BoolVar(&s.oidcSkipTLSVerify, "oidc-skip-tls-verify", false, "Skip TLS verification for the token issuer")
	flag.StringVar(&s.oidcAuthenticatorClientID, "oidc-authenticator-client-id", "", "Authenticator client ID")
	flag.StringVar(&s.oidcIssuerClientID, "oidc-issuer-client-id", "", "Issuer client ID")
	flag.StringVar(&s.oidcIssuerClientSecret, "oidc-issuer-client-secret", "", "OpenID client secret")
	flag.StringVar(&s.oidcIssuerRedirectURI, "oidc-issuer-redirect-uri", "", "Callback URL for OpenID responses.")
	flag.StringVar(&s.oidcIssuerCookieHashKey, "oidc-issuer-cookie-hash-key", "", "Hash key authenticates the cookie value using HMAC. It is recommended to use a key with 32 or 64 bytes.")
	flag.BoolVar(&s.oidcIssuerCookieSecureMode, "oidc-issuer-cookie-secure-mode", true, "When true cookie received only with HTTPS. Set false for local deployment with HTTP")
	flag.BoolVar(&s.oidcIssuerOfflineAccessAsScope, "oidc-issuer-offline-access-as-scope", true, "Set it to false if OIDC provider requires to set \"access_type=offline\" query param when accessing the refresh token")
	flag.Var(&s.featureGates, "feature-gates", "A set of key=value pairs that describe feature gates for various features.")
	flag.StringVar(&s.domain, "domain", "localhost", "A domain name on which the server is deployed")
	flag.StringVar(&s.serviceAccountSigningKey, "service-account-signing-key", "", "Signing key authenticates the service account's token value using HMAC. It is recommended to use a key with 32 bytes or longer.")
	flag.StringVar(&rawExposeStrategy, "expose-strategy", "NodePort", "The strategy to expose the controlplane with, either \"NodePort\" which creates NodePorts with a \"nodeport-proxy.k8s.io/expose: true\" annotation or \"LoadBalancer\", which creates a LoadBalancer")
	flag.BoolVar(&s.dynamicPresets, "dynamic-presets", false, "Whether to enable dynamic presets")
	flag.StringVar(&s.namespace, "namespace", "kubermatic", "The namespace kubermatic runs in, uses to determine where to look for datacenter custom resources")
	flag.StringVar(&configFile, "kubermatic-configuration-file", "", "(for development only) path to a KubermaticConfiguration YAML file")
	addFlags(flag.CommandLine)
	flag.Parse()

	var validExposeStrategy bool
	s.exposeStrategy, validExposeStrategy = kubermaticv1.ExposeStrategyFromString(rawExposeStrategy)
	if !validExposeStrategy {
		return s, fmt.Errorf("--expose-strategy must be one of: %s, got %q", kubermaticv1.AllExposeStrategies, rawExposeStrategy)
	}

	s.accessibleAddons = sets.NewString(strings.Split(rawAccessibleAddons, ",")...)
	s.accessibleAddons.Delete("")

	if len(caBundleFile) == 0 {
		return s, errors.New("no -ca-bundle configured")
	}

	cabundle, err := certificates.NewCABundleFromFile(caBundleFile)
	if err != nil {
		return s, fmt.Errorf("failed to read CA bundle file '%s': %v", caBundleFile, err)
	}

	s.caBundle = cabundle
	s.versions = kubermatic.NewDefaultVersions()

	return s, nil
}

func (o serverRunOptions) validate() error {
	if err := serviceaccount.ValidateKey([]byte(o.serviceAccountSigningKey)); err != nil {
		return fmt.Errorf("the service-account-signing-key is incorrect due to error: %v", err)
	}

	return nil
}

type providers struct {
	sshKey                                  provider.SSHKeyProvider
	privilegedSSHKeyProvider                provider.PrivilegedSSHKeyProvider
	user                                    provider.UserProvider
	serviceAccountProvider                  provider.ServiceAccountProvider
	privilegedServiceAccountProvider        provider.PrivilegedServiceAccountProvider
	serviceAccountTokenProvider             provider.ServiceAccountTokenProvider
	privilegedServiceAccountTokenProvider   provider.PrivilegedServiceAccountTokenProvider
	project                                 provider.ProjectProvider
	privilegedProject                       provider.PrivilegedProjectProvider
	projectMember                           provider.ProjectMemberProvider
	privilegedProjectMemberProvider         provider.PrivilegedProjectMemberProvider
	memberMapper                            provider.ProjectMemberMapper
	eventRecorderProvider                   provider.EventRecorderProvider
	clusterProviderGetter                   provider.ClusterProviderGetter
	seedsGetter                             provider.SeedsGetter
	seedClientGetter                        provider.SeedClientGetter
	configGetter                            provider.KubermaticConfigurationGetter
	addons                                  provider.AddonProviderGetter
	addonConfigProvider                     provider.AddonConfigProvider
	userInfoGetter                          provider.UserInfoGetter
	settingsProvider                        provider.SettingsProvider
	adminProvider                           provider.AdminProvider
	presetProvider                          provider.PresetProvider
	admissionPluginProvider                 provider.AdmissionPluginsProvider
	settingsWatcher                         watcher.SettingsWatcher
	userWatcher                             watcher.UserWatcher
	externalClusterProvider                 provider.ExternalClusterProvider
	privilegedExternalClusterProvider       provider.PrivilegedExternalClusterProvider
	constraintTemplateProvider              provider.ConstraintTemplateProvider
	defaultConstraintProvider               provider.DefaultConstraintProvider
	constraintProviderGetter                provider.ConstraintProviderGetter
	alertmanagerProviderGetter              provider.AlertmanagerProviderGetter
	clusterTemplateProvider                 provider.ClusterTemplateProvider
	clusterTemplateInstanceProviderGetter   provider.ClusterTemplateInstanceProviderGetter
	ruleGroupProviderGetter                 provider.RuleGroupProviderGetter
	privilegedAllowedRegistryProvider       provider.PrivilegedAllowedRegistryProvider
	etcdBackupConfigProviderGetter          provider.EtcdBackupConfigProviderGetter
	etcdRestoreProviderGetter               provider.EtcdRestoreProviderGetter
	etcdBackupConfigProjectProviderGetter   provider.EtcdBackupConfigProjectProviderGetter
	etcdRestoreProjectProviderGetter        provider.EtcdRestoreProjectProviderGetter
	backupCredentialsProviderGetter         provider.BackupCredentialsProviderGetter
	privilegedMLAAdminSettingProviderGetter provider.PrivilegedMLAAdminSettingProviderGetter
}

func loadKubermaticConfiguration(filename string) (*operatorv1alpha1.KubermaticConfiguration, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse file as YAML: %v", err)
	}

	defaulted, err := defaults.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to process: %v", err)
	}

	return defaulted, nil
}
