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
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/version"
)

// NewTestRouting is a hack that helps us avoid circular imports
// for example handler package uses v1/dc and v1/dc needs handler for testing
func NewTestRouting(
	datacenters map[string]provider.DatacenterMeta,
	clusterProviders map[string]provider.ClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	sshKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	serviceAccountProvider provider.ServiceAccountProvider,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	issuerVerifier auth.OIDCIssuerVerifier,
	tokenVerifiers auth.TokenVerifier,
	tokenExtractors auth.TokenExtractor,
	prometheusClient prometheusapi.Client,
	projectMemberProvider *kubernetes.ProjectMemberProvider,
	versions []*version.MasterVersion,
	updates []*version.MasterUpdate) http.Handler {

	updateManager := version.New(versions, updates)
	r := handler.NewRouting(
		datacenters,
		clusterProviders,
		cloudProviders,
		sshKeyProvider,
		userProvider,
		serviceAccountProvider,
		projectProvider,
		privilegedProjectProvider,
		issuerVerifier,
		tokenVerifiers,
		tokenExtractors,
		updateManager,
		prometheusClient,
		projectMemberProvider,
		projectMemberProvider, /*satisfies also a different interface*/
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
			[]string{"cluster", "seed_dc"},
		),
	}
}
