package handler

import (
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	prometheusapi "github.com/prometheus/client_golang/api"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/version"
)

// ContextKey defines a dedicated type for keys to use on contexts
type ContextKey string

const (
	rawToken                  ContextKey = "raw-auth-token"
	apiUserContextKey         ContextKey = "api-user"
	userCRContextKey          ContextKey = "user-cr"
	userInfoContextKey        ContextKey = "user-info"
	datacenterContextKey      ContextKey = "datacenter"
	clusterProviderContextKey ContextKey = "cluster-provider"
)

// UpdateManager specifies a set of methods to handle cluster versions & updates
type UpdateManager interface {
	GetVersion(string) (*version.MasterVersion, error)
	GetMasterVersions() ([]*version.MasterVersion, error)
	GetDefault() (*version.MasterVersion, error)
	AutomaticUpdate(from string) (*version.MasterVersion, error)
	GetPossibleUpdates(from string) ([]*version.MasterVersion, error)
}

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	datacenters           map[string]provider.DatacenterMeta
	cloudProviders        provider.CloudRegistry
	sshKeyProvider        provider.SSHKeyProvider
	userProvider          provider.UserProvider
	projectProvider       provider.ProjectProvider
	logger                log.Logger
	oidcAuthenticator     OIDCAuthenticator
	oidcIssuer            OIDCIssuerVerifier
	clusterProviders      map[string]provider.ClusterProvider
	updateManager         UpdateManager
	prometheusClient      prometheusapi.Client
	projectMemberProvider provider.ProjectMemberProvider
	userProjectMapper     provider.ProjectMemberMapper
}

// NewRouting creates a new Routing.
func NewRouting(
	datacenters map[string]provider.DatacenterMeta,
	newClusterProviders map[string]provider.ClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	newSSHKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	projectProvider provider.ProjectProvider,
	oidcAuthenticator OIDCAuthenticator,
	oidcIssuerVerifier OIDCIssuerVerifier,
	updateManager UpdateManager,
	prometheusClient prometheusapi.Client,
	projectMemberProvider provider.ProjectMemberProvider,
	userProjectMapper provider.ProjectMemberMapper,
) Routing {
	return Routing{
		datacenters:           datacenters,
		clusterProviders:      newClusterProviders,
		sshKeyProvider:        newSSHKeyProvider,
		userProvider:          userProvider,
		projectProvider:       projectProvider,
		cloudProviders:        cloudProviders,
		logger:                log.NewLogfmtLogger(os.Stderr),
		oidcAuthenticator:     oidcAuthenticator,
		oidcIssuer:            oidcIssuerVerifier,
		updateManager:         updateManager,
		prometheusClient:      prometheusClient,
		projectMemberProvider: projectMemberProvider,
		userProjectMapper:     userProjectMapper,
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	return []httptransport.ServerOption{
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.oidcAuthenticator.Extractor()),
	}
}
