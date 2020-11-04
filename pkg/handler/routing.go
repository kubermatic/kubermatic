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
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	prometheusapi "github.com/prometheus/client_golang/api"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/handler/auth"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/watcher"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	log                                   *zap.SugaredLogger
	presetsProvider                       provider.PresetProvider
	seedsGetter                           provider.SeedsGetter
	seedsClientGetter                     provider.SeedClientGetter
	sshKeyProvider                        provider.SSHKeyProvider
	privilegedSSHKeyProvider              provider.PrivilegedSSHKeyProvider
	userProvider                          provider.UserProvider
	serviceAccountProvider                provider.ServiceAccountProvider
	privilegedServiceAccountProvider      provider.PrivilegedServiceAccountProvider
	serviceAccountTokenProvider           provider.ServiceAccountTokenProvider
	privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider
	projectProvider                       provider.ProjectProvider
	privilegedProjectProvider             provider.PrivilegedProjectProvider
	logger                                log.Logger
	oidcIssuerVerifier                    auth.OIDCIssuerVerifier
	tokenVerifiers                        auth.TokenVerifier
	tokenExtractors                       auth.TokenExtractor
	clusterProviderGetter                 provider.ClusterProviderGetter
	addonProviderGetter                   provider.AddonProviderGetter
	addonConfigProvider                   provider.AddonConfigProvider
	updateManager                         common.UpdateManager
	prometheusClient                      prometheusapi.Client
	projectMemberProvider                 provider.ProjectMemberProvider
	privilegedProjectMemberProvider       provider.PrivilegedProjectMemberProvider
	userProjectMapper                     provider.ProjectMemberMapper
	saTokenAuthenticator                  serviceaccount.TokenAuthenticator
	saTokenGenerator                      serviceaccount.TokenGenerator
	eventRecorderProvider                 provider.EventRecorderProvider
	exposeStrategy                        corev1.ServiceType
	accessibleAddons                      sets.String
	userInfoGetter                        provider.UserInfoGetter
	settingsProvider                      provider.SettingsProvider
	adminProvider                         provider.AdminProvider
	admissionPluginProvider               provider.AdmissionPluginsProvider
	settingsWatcher                       watcher.SettingsWatcher
	userWatcher                           watcher.UserWatcher
}

// NewRouting creates a new Routing.
func NewRouting(routingParams RoutingParams) Routing {
	return Routing{
		log:                                   routingParams.Log,
		presetsProvider:                       routingParams.PresetsProvider,
		seedsGetter:                           routingParams.SeedsGetter,
		seedsClientGetter:                     routingParams.SeedsClientGetter,
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
		logger:                                log.NewLogfmtLogger(os.Stderr),
		oidcIssuerVerifier:                    routingParams.OIDCIssuerVerifier,
		tokenVerifiers:                        routingParams.TokenVerifiers,
		tokenExtractors:                       routingParams.TokenExtractors,
		updateManager:                         routingParams.UpdateManager,
		prometheusClient:                      routingParams.PrometheusClient,
		projectMemberProvider:                 routingParams.ProjectMemberProvider,
		privilegedProjectMemberProvider:       routingParams.PrivilegedProjectMemberProvider,
		userProjectMapper:                     routingParams.UserProjectMapper,
		saTokenAuthenticator:                  routingParams.SATokenAuthenticator,
		saTokenGenerator:                      routingParams.SATokenGenerator,
		eventRecorderProvider:                 routingParams.EventRecorderProvider,
		exposeStrategy:                        routingParams.ExposeStrategy,
		accessibleAddons:                      routingParams.AccessibleAddons,
		userInfoGetter:                        routingParams.UserInfoGetter,
		settingsProvider:                      routingParams.SettingsProvider,
		adminProvider:                         routingParams.AdminProvider,
		admissionPluginProvider:               routingParams.AdmissionPluginProvider,
		settingsWatcher:                       routingParams.SettingsWatcher,
		userWatcher:                           routingParams.UserWatcher,
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	return []httptransport.ServerOption{
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(ErrorEncoder),
		httptransport.ServerBefore(middleware.TokenExtractor(r.tokenExtractors)),
	}
}

type RoutingParams struct {
	Log                                   *zap.SugaredLogger
	PresetsProvider                       provider.PresetProvider
	SeedsGetter                           provider.SeedsGetter
	SeedsClientGetter                     provider.SeedClientGetter
	SSHKeyProvider                        provider.SSHKeyProvider
	PrivilegedSSHKeyProvider              provider.PrivilegedSSHKeyProvider
	UserProvider                          provider.UserProvider
	ServiceAccountProvider                provider.ServiceAccountProvider
	PrivilegedServiceAccountProvider      provider.PrivilegedServiceAccountProvider
	ServiceAccountTokenProvider           provider.ServiceAccountTokenProvider
	PrivilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider
	ProjectProvider                       provider.ProjectProvider
	PrivilegedProjectProvider             provider.PrivilegedProjectProvider
	OIDCIssuerVerifier                    auth.OIDCIssuerVerifier
	TokenVerifiers                        auth.TokenVerifier
	TokenExtractors                       auth.TokenExtractor
	ClusterProviderGetter                 provider.ClusterProviderGetter
	AddonProviderGetter                   provider.AddonProviderGetter
	AddonConfigProvider                   provider.AddonConfigProvider
	UpdateManager                         common.UpdateManager
	PrometheusClient                      prometheusapi.Client
	ProjectMemberProvider                 provider.ProjectMemberProvider
	PrivilegedProjectMemberProvider       provider.PrivilegedProjectMemberProvider
	UserProjectMapper                     provider.ProjectMemberMapper
	SATokenAuthenticator                  serviceaccount.TokenAuthenticator
	SATokenGenerator                      serviceaccount.TokenGenerator
	EventRecorderProvider                 provider.EventRecorderProvider
	ExposeStrategy                        corev1.ServiceType
	AccessibleAddons                      sets.String
	UserInfoGetter                        provider.UserInfoGetter
	SettingsProvider                      provider.SettingsProvider
	AdminProvider                         provider.AdminProvider
	AdmissionPluginProvider               provider.AdmissionPluginsProvider
	SettingsWatcher                       watcher.SettingsWatcher
	UserWatcher                           watcher.UserWatcher
	ExternalClusterProvider               provider.ExternalClusterProvider
	PrivilegedExternalClusterProvider     provider.PrivilegedExternalClusterProvider
	ConstraintTemplateProvider            provider.ConstraintTemplateProvider
	ConstraintProvider                    provider.ConstraintProvider
	PrivilegedConstraintProvider          provider.PrivilegedConstraintProvider
}
