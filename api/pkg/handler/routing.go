package handler

import (
	"context"
	"net/http"
	"os"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	prometheusapi "github.com/prometheus/client_golang/api"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/version"
)

// ContextKey defines a dedicated type for keys to use on contexts
type ContextKey string

const (
	rawToken                     ContextKey = "raw-auth-token"
	apiUserContextKey            ContextKey = "api-user"
	userCRContextKey             ContextKey = "user-cr"
	datacenterContextKey         ContextKey = "datacenter"
	clusterProviderContextKey    ContextKey = "cluster-provider"
	newClusterProviderContextKey ContextKey = "new-cluster-provider"
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
	newSSHKeyProvider     provider.NewSSHKeyProvider
	userProvider          provider.UserProvider
	projectProvider       provider.ProjectProvider
	logger                log.Logger
	authenticator         Authenticator
	clusterProviders      map[string]provider.ClusterProvider
	newClusterProviders   map[string]provider.NewClusterProvider
	updateManager         UpdateManager
	prometheusClient      prometheusapi.Client
	projectMemberProvider provider.ProjectMemberProvider
}

// NewRouting creates a new Routing.
func NewRouting(
	datacenters map[string]provider.DatacenterMeta,
	clusterProviders map[string]provider.ClusterProvider,
	newClusterProviders map[string]provider.NewClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	sshKeyProvider provider.SSHKeyProvider,
	newSSHKeyProvider provider.NewSSHKeyProvider,
	userProvider provider.UserProvider,
	projectProvider provider.ProjectProvider,
	authenticator Authenticator,
	updateManager UpdateManager,
	prometheusClient prometheusapi.Client,
	projectMemberProvider provider.ProjectMemberProvider,
) Routing {
	return Routing{
		datacenters:           datacenters,
		clusterProviders:      clusterProviders,
		newClusterProviders:   newClusterProviders,
		sshKeyProvider:        sshKeyProvider,
		newSSHKeyProvider:     newSSHKeyProvider,
		userProvider:          userProvider,
		projectProvider:       projectProvider,
		cloudProviders:        cloudProviders,
		logger:                log.NewLogfmtLogger(os.Stderr),
		authenticator:         authenticator,
		updateManager:         updateManager,
		prometheusClient:      prometheusClient,
		projectMemberProvider: projectMemberProvider,
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	return []httptransport.ServerOption{
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(r.authenticator.Extractor()),
	}
}

func newNotImplementedEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		return nil, errors.NewNotImplemented()
	}
}

// NotImplemented return a "Not Implemented" error.
func (r Routing) NotImplemented() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(newNotImplementedEndpoint()),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
