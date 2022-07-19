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

package handler

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"os"

	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	prometheusapi "github.com/prometheus/client_golang/api"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/kubermatic/v2/pkg/watcher"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	log                                   *zap.SugaredLogger
	logger                                log.Logger
	versions                              kubermatic.Versions
	presetProvider                        provider.PresetProvider
	masterClient                          ctrlruntimeclient.Client
	seedsGetter                           provider.SeedsGetter
	seedsClientGetter                     provider.SeedClientGetter
	kubermaticConfigGetter                provider.KubermaticConfigurationGetter
	sshKeyProvider                        provider.SSHKeyProvider
	privilegedSSHKeyProvider              provider.PrivilegedSSHKeyProvider
	userProvider                          provider.UserProvider
	serviceAccountProvider                provider.ServiceAccountProvider
	privilegedServiceAccountProvider      provider.PrivilegedServiceAccountProvider
	serviceAccountTokenProvider           provider.ServiceAccountTokenProvider
	privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider
	projectProvider                       provider.ProjectProvider
	privilegedProjectProvider             provider.PrivilegedProjectProvider
	oidcIssuerVerifier                    auth.OIDCIssuerVerifier
	tokenVerifiers                        auth.TokenVerifier
	tokenExtractors                       auth.TokenExtractor
	clusterProviderGetter                 provider.ClusterProviderGetter
	addonProviderGetter                   provider.AddonProviderGetter
	addonConfigProvider                   provider.AddonConfigProvider
	prometheusClient                      prometheusapi.Client
	projectMemberProvider                 provider.ProjectMemberProvider
	privilegedProjectMemberProvider       provider.PrivilegedProjectMemberProvider
	featureGatesProvider                  provider.FeatureGatesProvider
	userProjectMapper                     provider.ProjectMemberMapper
	saTokenAuthenticator                  serviceaccount.TokenAuthenticator
	saTokenGenerator                      serviceaccount.TokenGenerator
	eventRecorderProvider                 provider.EventRecorderProvider
	exposeStrategy                        kubermaticv1.ExposeStrategy
	userInfoGetter                        provider.UserInfoGetter
	settingsProvider                      provider.SettingsProvider
	adminProvider                         provider.AdminProvider
	admissionPluginProvider               provider.AdmissionPluginsProvider
	settingsWatcher                       watcher.SettingsWatcher
	userWatcher                           watcher.UserWatcher
	caBundle                              *x509.CertPool
	features                              features.FeatureGate
	seedProvider                          provider.SeedProvider
	resourceQuotaProvider                 provider.ResourceQuotaProvider
}

// NewRouting creates a new Routing.
func NewRouting(routingParams RoutingParams, masterClient ctrlruntimeclient.Client) Routing {
	return Routing{
		log:                                   routingParams.Log,
		logger:                                log.NewLogfmtLogger(os.Stderr),
		presetProvider:                        routingParams.PresetProvider,
		masterClient:                          masterClient,
		seedsGetter:                           routingParams.SeedsGetter,
		seedsClientGetter:                     routingParams.SeedsClientGetter,
		kubermaticConfigGetter:                routingParams.KubermaticConfigurationGetter,
		clusterProviderGetter:                 routingParams.ClusterProviderGetter,
		addonProviderGetter:                   routingParams.AddonProviderGetter,
		addonConfigProvider:                   routingParams.AddonConfigProvider,
		sshKeyProvider:                        routingParams.SSHKeyProvider,
		privilegedSSHKeyProvider:              routingParams.PrivilegedSSHKeyProvider,
		userProvider:                          routingParams.UserProvider,
		serviceAccountProvider:                routingParams.ServiceAccountProvider,
		privilegedServiceAccountProvider:      routingParams.PrivilegedServiceAccountProvider,
		serviceAccountTokenProvider:           routingParams.ServiceAccountTokenProvider,
		privilegedServiceAccountTokenProvider: routingParams.PrivilegedServiceAccountTokenProvider,
		projectProvider:                       routingParams.ProjectProvider,
		privilegedProjectProvider:             routingParams.PrivilegedProjectProvider,
		oidcIssuerVerifier:                    routingParams.OIDCIssuerVerifier,
		tokenVerifiers:                        routingParams.TokenVerifiers,
		tokenExtractors:                       routingParams.TokenExtractors,
		prometheusClient:                      routingParams.PrometheusClient,
		projectMemberProvider:                 routingParams.ProjectMemberProvider,
		privilegedProjectMemberProvider:       routingParams.PrivilegedProjectMemberProvider,
		featureGatesProvider:                  routingParams.FeatureGatesProvider,
		userProjectMapper:                     routingParams.UserProjectMapper,
		saTokenAuthenticator:                  routingParams.SATokenAuthenticator,
		saTokenGenerator:                      routingParams.SATokenGenerator,
		eventRecorderProvider:                 routingParams.EventRecorderProvider,
		exposeStrategy:                        routingParams.ExposeStrategy,
		userInfoGetter:                        routingParams.UserInfoGetter,
		settingsProvider:                      routingParams.SettingsProvider,
		adminProvider:                         routingParams.AdminProvider,
		admissionPluginProvider:               routingParams.AdmissionPluginProvider,
		settingsWatcher:                       routingParams.SettingsWatcher,
		userWatcher:                           routingParams.UserWatcher,
		versions:                              routingParams.Versions,
		caBundle:                              routingParams.CABundle,
		features:                              routingParams.Features,
		seedProvider:                          routingParams.SeedProvider,
		resourceQuotaProvider:                 routingParams.ResourceQuotaProvider,
	}
}

func NewRequestErrorHandler(log *zap.SugaredLogger, req *http.Request) transport.ErrorHandlerFunc {
	return func(ctx context.Context, err error) {
		// When the client cancels the request, the context is canceled.
		// In this case we do not want to log the error.
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}

		// client-side errors should also not be logged
		if httpErr := AsHTTPError(err); httpErr != nil && httpErr.StatusCode() < http.StatusInternalServerError {
			return
		}

		log.Errorw(err.Error(), "request", req.URL.String())
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	var req *http.Request

	return []httptransport.ServerOption{
		httptransport.ServerBefore(func(c context.Context, r *http.Request) context.Context {
			req = r
			return c
		}),
		httptransport.ServerErrorHandler(NewRequestErrorHandler(r.log, req)),
		httptransport.ServerErrorEncoder(ErrorEncoder),
		httptransport.ServerBefore(middleware.TokenExtractor(r.tokenExtractors)),
	}
}

type RoutingParams struct {
	Log                                     *zap.SugaredLogger
	PresetProvider                          provider.PresetProvider
	SeedsGetter                             provider.SeedsGetter
	SeedsClientGetter                       provider.SeedClientGetter
	KubermaticConfigurationGetter           provider.KubermaticConfigurationGetter
	SSHKeyProvider                          provider.SSHKeyProvider
	PrivilegedSSHKeyProvider                provider.PrivilegedSSHKeyProvider
	UserProvider                            provider.UserProvider
	ServiceAccountProvider                  provider.ServiceAccountProvider
	PrivilegedServiceAccountProvider        provider.PrivilegedServiceAccountProvider
	ServiceAccountTokenProvider             provider.ServiceAccountTokenProvider
	PrivilegedServiceAccountTokenProvider   provider.PrivilegedServiceAccountTokenProvider
	ProjectProvider                         provider.ProjectProvider
	PrivilegedProjectProvider               provider.PrivilegedProjectProvider
	OIDCIssuerVerifier                      auth.OIDCIssuerVerifier
	TokenVerifiers                          auth.TokenVerifier
	TokenExtractors                         auth.TokenExtractor
	ClusterProviderGetter                   provider.ClusterProviderGetter
	AddonProviderGetter                     provider.AddonProviderGetter
	AddonConfigProvider                     provider.AddonConfigProvider
	PrometheusClient                        prometheusapi.Client
	ProjectMemberProvider                   provider.ProjectMemberProvider
	PrivilegedProjectMemberProvider         provider.PrivilegedProjectMemberProvider
	UserProjectMapper                       provider.ProjectMemberMapper
	SATokenAuthenticator                    serviceaccount.TokenAuthenticator
	SATokenGenerator                        serviceaccount.TokenGenerator
	EventRecorderProvider                   provider.EventRecorderProvider
	ExposeStrategy                          kubermaticv1.ExposeStrategy
	UserInfoGetter                          provider.UserInfoGetter
	SettingsProvider                        provider.SettingsProvider
	AdminProvider                           provider.AdminProvider
	AdmissionPluginProvider                 provider.AdmissionPluginsProvider
	SettingsWatcher                         watcher.SettingsWatcher
	UserWatcher                             watcher.UserWatcher
	ExternalClusterProvider                 provider.ExternalClusterProvider
	PrivilegedExternalClusterProvider       provider.PrivilegedExternalClusterProvider
	FeatureGatesProvider                    provider.FeatureGatesProvider
	DefaultConstraintProvider               provider.DefaultConstraintProvider
	ConstraintTemplateProvider              provider.ConstraintTemplateProvider
	ConstraintProviderGetter                provider.ConstraintProviderGetter
	AlertmanagerProviderGetter              provider.AlertmanagerProviderGetter
	ClusterTemplateProvider                 provider.ClusterTemplateProvider
	ClusterTemplateInstanceProviderGetter   provider.ClusterTemplateInstanceProviderGetter
	RuleGroupProviderGetter                 provider.RuleGroupProviderGetter
	PrivilegedAllowedRegistryProvider       provider.PrivilegedAllowedRegistryProvider
	EtcdBackupConfigProviderGetter          provider.EtcdBackupConfigProviderGetter
	EtcdRestoreProviderGetter               provider.EtcdRestoreProviderGetter
	EtcdBackupConfigProjectProviderGetter   provider.EtcdBackupConfigProjectProviderGetter
	EtcdRestoreProjectProviderGetter        provider.EtcdRestoreProjectProviderGetter
	BackupCredentialsProviderGetter         provider.BackupCredentialsProviderGetter
	PrivilegedMLAAdminSettingProviderGetter provider.PrivilegedMLAAdminSettingProviderGetter
	SeedProvider                            provider.SeedProvider
	ResourceQuotaProvider                   provider.ResourceQuotaProvider
	GroupProjectBindingProvider             provider.GroupProjectBindingProvider
	PrivilegedIPAMPoolProviderGetter        provider.PrivilegedIPAMPoolProviderGetter
	ApplicationDefinitionProvider           provider.ApplicationDefinitionProvider
	Versions                                kubermatic.Versions
	CABundle                                *x509.CertPool
	Features                                features.FeatureGate
}
