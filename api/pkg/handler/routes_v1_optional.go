package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
)

// RegisterV1Optional declares all router paths for v1
func (r Routing) RegisterV1Optional(mux *mux.Router, oidcKubeConfEndpoint bool, oidcCfg common.OIDCConfiguration, mainMux *mux.Router) {
	// if enabled exposes defines an endpoint for generating kubeconfig for a cluster that will contain OIDC tokens
	if oidcKubeConfEndpoint {
		mux.Methods(http.MethodGet).
			Path("/kubeconfig").
			Handler(r.createOIDCKubeconfig(oidcCfg))
	}
}

// swagger:route GET /api/v1/kubeconfig createOIDCKubeconfig
//
//     Starts OIDC flow and generates kubeconfig, the generated config
//     contains OIDC provider authentication info
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Kubeconfig
func (r Routing) createOIDCKubeconfig(oidcCfg common.OIDCConfiguration) http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.SetClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.SetPrivilegedClusterProvider(r.clusterProviderGetter, r.seedsGetter),
			middleware.UserInfoUnauthorized(r.userProjectMapper, r.userProvider),
		)(cluster.CreateOIDCKubeconfigEndpoint(r.projectProvider, r.privilegedProjectProvider, r.oidcIssuerVerifier, oidcCfg)),
		cluster.DecodeCreateOIDCKubeconfig,
		cluster.EncodeOIDCKubeconfig,
		r.defaultServerOptions()...,
	)
}
