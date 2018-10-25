package handler

import (
	"fmt"
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

	//
	// if enabled exposes defines an endpoint for generating kubeconfig for a cluster that will contain OIDC tokens
	if oidcKubeConfEndpoint {
		// GET or POST ?? !!
		mux.Methods(http.MethodGet).
			Path("/kubeconfig").
			Handler(r.createOIDCKubeconfig(oidcCfg))

		// Remove this EP
		mainMux.Methods(http.MethodGet).
			Path("/clusters").
			Handler(r.redirectTo("/api/v1/kubeconfig"))
	}
}

// swagger:route GET /api/v1/kubeconfig
//
// Lists sizes from digitalocean
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: TODO ???!!!!!!!!!!!!!!!!!!!!!!!!!!!?????!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
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

// Remove this EP
func (r Routing) redirectTo(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery := r.URL.RawQuery
		newPath := fmt.Sprintf("%s?%s", path, rawQuery)
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	})
}
