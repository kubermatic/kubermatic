package hack

import (
	"net/http"

	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"

	"github.com/kubermatic/kubermatic/api/pkg/handler"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
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
	projectProvider provider.ProjectProvider,
	authenticator auth.OIDCAuthenticator,
	issuerVerifier auth.OIDCIssuerVerifier,
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
		projectProvider,
		authenticator,
		issuerVerifier,
		updateManager,
		prometheusClient,
		projectMemberProvider,
		projectMemberProvider, /*satisfies also a different interface*/
	)

	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	r.RegisterV1(v1Router)
	r.RegisterV1Optional(v1Router,
		true,
		*generateDefaultOicdCfg(),
		mainRouter,
	)
	return mainRouter
}

// generateDefaultOicdCfg creates test configuration for OpenID clients
func generateDefaultOicdCfg() *handler.OIDCConfiguration {
	return &handler.OIDCConfiguration{
		URL:                  test.IssuerURL,
		ClientID:             test.IssuerClientID,
		ClientSecret:         test.IssuerClientSecret,
		OfflineAccessAsScope: true,
	}
}
