package hack

import (
	"net/http"

	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// NewTestRouting is a hack that helps us avoid circular imports
// for example handler package uses v1/dc and v1/dc needs handler for testing
func NewTestRouting(
	seedsGetter provider.SeedsGetter,
	clusterProvidersGetter provider.ClusterProviderGetter,
	addonProviderGetter provider.AddonProviderGetter,
	sshKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	serviceAccountProvider provider.ServiceAccountProvider,
	serviceAccountTokenProvider provider.ServiceAccountTokenProvider,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	issuerVerifier auth.OIDCIssuerVerifier,
	tokenVerifiers auth.TokenVerifier,
	tokenExtractors auth.TokenExtractor,
	prometheusClient prometheusapi.Client,
	projectMemberProvider *kubernetes.ProjectMemberProvider,
	versions []*version.Version,
	updates []*version.Update,
	saTokenAuthenticator serviceaccount.TokenAuthenticator,
	saTokenGenerator serviceaccount.TokenGenerator,
	eventRecorderProvider provider.EventRecorderProvider,
	credentialManager common.PresetsManager) http.Handler {

	updateManager := version.New(versions, updates)
	r := handler.NewRouting(
		kubermaticlog.Logger,
		seedsGetter,
		clusterProvidersGetter,
		addonProviderGetter,
		sshKeyProvider,
		userProvider,
		serviceAccountProvider,
		serviceAccountTokenProvider,
		projectProvider,
		privilegedProjectProvider,
		issuerVerifier,
		tokenVerifiers,
		tokenExtractors,
		updateManager,
		prometheusClient,
		projectMemberProvider,
		projectMemberProvider, /*satisfies also a different interface*/
		saTokenAuthenticator,
		saTokenGenerator,
		eventRecorderProvider,
		credentialManager,
		corev1.ServiceTypeNodePort,
	)

	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	r.RegisterV1(v1Router, generateDefaultMetrics(), sets.String{})
	r.RegisterV1Legacy(v1Router)
	r.RegisterV1Optional(v1Router,
		true,
		*generateDefaultOicdCfg(),
		mainRouter,
	)
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
