package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// OIDCConfiguration is a struct that holds
// OIDC provider configuration data, read from command line arguments
type OIDCConfiguration struct {
	// URL holds OIDC Issuer URL address
	URL string
	// ClientID holds OIDC ClientID
	ClientID string
	// ClientSecret holds OIDC ClientSecret
	ClientSecret string
}

// RegisterV1Optional declares all router paths for v1
func (r Routing) RegisterV1Optional(mux *mux.Router, oidcKubeConfEndpoint bool, oidcCfg OIDCConfiguration, mainMux *mux.Router) {
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
func (r Routing) createOIDCKubeconfig(oidcCfg OIDCConfiguration) http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.datacenterMiddleware(),
			r.userInfoMiddlewareUnauthorized(),
		)(createOIDCKubeconfig(r.projectProvider, r.oidcIssuer, oidcCfg)),
		decodeCreateOIDCKubeconfig,
		encodeKubeconfigDoINeddAcditional,
		r.defaultServerOptions()...,
	)
}
