package handler

import (
	"context"
	"net/http"
	"os"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

const (
	rawToken = iota
	apiUserContextKey
	userCRContextKey
	datacenterContextKey
	clusterProviderContextKey
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	ctx                       context.Context
	datacenters               map[string]provider.DatacenterMeta
	optimisticClusterProvider provider.ClusterProvider
	cloudProviders            map[string]provider.CloudProvider
	sshKeyProvider            provider.SSHKeyProvider
	userProvider              provider.UserProvider
	logger                    log.Logger
	authenticator             Authenticator
	versions                  map[string]*apiv1.MasterVersion
	updates                   []apiv1.MasterUpdate
	clusterProviders          map[string]provider.ClusterProvider
}

// NewRouting creates a new Routing.
func NewRouting(
	ctx context.Context,
	datacenters map[string]provider.DatacenterMeta,
	clusterProviders map[string]provider.ClusterProvider,
	optimisticClusterProvider provider.ClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	sshKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	authenticator Authenticator,
	versions map[string]*apiv1.MasterVersion,
	updates []apiv1.MasterUpdate,
) Routing {
	return Routing{
		ctx:                       ctx,
		datacenters:               datacenters,
		clusterProviders:          clusterProviders,
		optimisticClusterProvider: optimisticClusterProvider,
		sshKeyProvider:            sshKeyProvider,
		userProvider:              userProvider,
		cloudProviders:            cloudProviders,
		logger:                    log.NewLogfmtLogger(os.Stderr),
		authenticator:             authenticator,
		versions:                  versions,
		updates:                   updates,
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
