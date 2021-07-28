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

package v2

import (
	"crypto/x509"
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	prometheusapi "github.com/prometheus/client_golang/api"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/kubermatic/v2/pkg/watcher"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	log                                   *zap.SugaredLogger
	logger                                log.Logger
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
	exposeStrategy                        kubermaticv1.ExposeStrategy
	accessibleAddons                      sets.String
	userInfoGetter                        provider.UserInfoGetter
	settingsProvider                      provider.SettingsProvider
	adminProvider                         provider.AdminProvider
	admissionPluginProvider               provider.AdmissionPluginsProvider
	settingsWatcher                       watcher.SettingsWatcher
	userWatcher                           watcher.UserWatcher
	externalClusterProvider               provider.ExternalClusterProvider
	privilegedExternalClusterProvider     provider.PrivilegedExternalClusterProvider
	defaultConstraintProvider             provider.DefaultConstraintProvider
	constraintTemplateProvider            provider.ConstraintTemplateProvider
	constraintProviderGetter              provider.ConstraintProviderGetter
	alertmanagerProviderGetter            provider.AlertmanagerProviderGetter
	clusterTemplateProvider               provider.ClusterTemplateProvider
	clusterTemplateInstanceProviderGetter provider.ClusterTemplateInstanceProviderGetter
	ruleGroupProviderGetter               provider.RuleGroupProviderGetter
	privilegedAllowedRegistryProvider     provider.PrivilegedAllowedRegistryProvider
	etcdBackupConfigProviderGetter        provider.EtcdBackupConfigProviderGetter
	etcdRestoreProviderGetter             provider.EtcdRestoreProviderGetter
	versions                              kubermatic.Versions
	caBundle                              *x509.CertPool
}

// NewV2Routing creates a new Routing.
func NewV2Routing(routingParams handler.RoutingParams) Routing {
	return Routing{
		log:                                   routingParams.Log,
		logger:                                log.NewLogfmtLogger(os.Stderr),
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
		externalClusterProvider:               routingParams.ExternalClusterProvider,
		privilegedExternalClusterProvider:     routingParams.PrivilegedExternalClusterProvider,
		defaultConstraintProvider:             routingParams.DefaultConstraintProvider,
		constraintTemplateProvider:            routingParams.ConstraintTemplateProvider,
		constraintProviderGetter:              routingParams.ConstraintProviderGetter,
		alertmanagerProviderGetter:            routingParams.AlertmanagerProviderGetter,
		clusterTemplateProvider:               routingParams.ClusterTemplateProvider,
		clusterTemplateInstanceProviderGetter: routingParams.ClusterTemplateInstanceProviderGetter,
		ruleGroupProviderGetter:               routingParams.RuleGroupProviderGetter,
		privilegedAllowedRegistryProvider:     routingParams.PrivilegedAllowedRegistryProvider,
		etcdBackupConfigProviderGetter:        routingParams.EtcdBackupConfigProviderGetter,
		etcdRestoreProviderGetter:             routingParams.EtcdRestoreProviderGetter,
		versions:                              routingParams.Versions,
		caBundle:                              routingParams.CABundle,
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	return []httptransport.ServerOption{
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(handler.ErrorEncoder),
		httptransport.ServerBefore(middleware.TokenExtractor(r.tokenExtractors)),
		httptransport.ServerBefore(middleware.SetSeedsGetter(r.seedsGetter)),
	}
}
