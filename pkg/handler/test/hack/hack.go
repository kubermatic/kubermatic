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

package hack

import (
	"net/http"

	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/pkg/handler"
	"github.com/kubermatic/kubermatic/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/pkg/handler/test"
	"github.com/kubermatic/kubermatic/pkg/handler/v1/common"
	kubermaticlog "github.com/kubermatic/kubermatic/pkg/log"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/pkg/serviceaccount"
	"github.com/kubermatic/kubermatic/pkg/version"
	"github.com/kubermatic/kubermatic/pkg/watcher"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// NewTestRouting is a hack that helps us avoid circular imports
// for example handler package uses v1/dc and v1/dc needs handler for testing
func NewTestRouting(
	adminProvider provider.AdminProvider,
	settingsProvider provider.SettingsProvider,
	userInfoGetter provider.UserInfoGetter,
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
	clusterProvidersGetter provider.ClusterProviderGetter,
	addonProviderGetter provider.AddonProviderGetter,
	addonConfigProvider provider.AddonConfigProvider,
	sshKeyProvider provider.SSHKeyProvider,
	privilegedSSHKeyProvider provider.PrivilegedSSHKeyProvider,
	userProvider provider.UserProvider,
	serviceAccountProvider provider.ServiceAccountProvider,
	privilegedServiceAccountProvider provider.PrivilegedServiceAccountProvider,
	serviceAccountTokenProvider provider.ServiceAccountTokenProvider,
	privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	issuerVerifier auth.OIDCIssuerVerifier,
	tokenVerifiers auth.TokenVerifier,
	tokenExtractors auth.TokenExtractor,
	prometheusClient prometheusapi.Client,
	projectMemberProvider *kubernetes.ProjectMemberProvider,
	privilegedProjectMemberProvider provider.PrivilegedProjectMemberProvider,
	versions []*version.Version,
	updates []*version.Update,
	saTokenAuthenticator serviceaccount.TokenAuthenticator,
	saTokenGenerator serviceaccount.TokenGenerator,
	eventRecorderProvider provider.EventRecorderProvider,
	presetsProvider provider.PresetProvider,
	admissionPluginProvider provider.AdmissionPluginsProvider,
	settingsWatcher watcher.SettingsWatcher) http.Handler {

	updateManager := version.New(versions, updates)
	r := handler.NewRouting(
		kubermaticlog.Logger,
		presetsProvider,
		seedsGetter,
		seedClientGetter,
		clusterProvidersGetter,
		addonProviderGetter,
		addonConfigProvider,
		sshKeyProvider,
		privilegedSSHKeyProvider,
		userProvider,
		serviceAccountProvider,
		privilegedServiceAccountProvider,
		serviceAccountTokenProvider,
		privilegedServiceAccountTokenProvider,
		projectProvider,
		privilegedProjectProvider,
		issuerVerifier,
		tokenVerifiers,
		tokenExtractors,
		updateManager,
		prometheusClient,
		projectMemberProvider,
		privilegedProjectMemberProvider,
		projectMemberProvider, /*satisfies also a different interface*/
		saTokenAuthenticator,
		saTokenGenerator,
		eventRecorderProvider,
		corev1.ServiceTypeNodePort,
		sets.String{},
		userInfoGetter,
		settingsProvider,
		adminProvider,
		admissionPluginProvider,
		settingsWatcher,
	)

	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	r.RegisterV1(v1Router, generateDefaultMetrics())
	r.RegisterV1Legacy(v1Router)
	r.RegisterV1Optional(v1Router,
		true,
		*generateDefaultOicdCfg(),
		mainRouter,
	)
	r.RegisterV1Admin(v1Router)
	return mainRouter
}

// generateDefaultOicdCfg creates test configuration for OpenID clients
func generateDefaultOicdCfg() *common.OIDCConfiguration {
	return &common.OIDCConfiguration{
		URL:                  test.IssuerURL,
		ClientID:             test.IssuerClientID,
		ClientSecret:         test.IssuerClientSecret,
		OfflineAccessAsScope: true,
	}
}

func generateDefaultMetrics() common.ServerMetrics {
	return common.ServerMetrics{
		InitNodeDeploymentFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kubermatic_api_init_node_deployment_failures",
				Help: "The number of times initial node deployment couldn't be created within the timeout",
			},
			[]string{"cluster", "datacenter"},
		),
	}
}
