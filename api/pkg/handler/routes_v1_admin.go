package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/admin"
)

//RegisterV1Admin declares all router paths for the admin users
func (r Routing) RegisterV1Admin(mux *mux.Router) {
	//
	// Defines a set of HTTP endpoints for the admin users
	mux.Methods(http.MethodGet).
		Path("/admin/settings").
		Handler(r.getKubermaticSettings())

	mux.Methods(http.MethodPatch).
		Path("/admin/settings").
		Handler(r.patchKubermaticSettings())
}

// swagger:route GET /api/v1/admin/settings admin getKubermaticSettings
//
//     Gets the global settings.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GlobalSettings
//       401: empty
//       403: empty
func (r Routing) getKubermaticSettings() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.KubermaticSettingsEndpoint(r.userInfoGetter, r.settingsProvider)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v1/admin/settings admin patchKubermaticSettings
//
//     Patches the global settings.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GlobalSettings
//       401: empty
//       403: empty
func (r Routing) patchKubermaticSettings() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateKubermaticSettingsEndpoint(r.userInfoGetter, r.settingsProvider)),
		admin.DecodePatchKubermaticSettingsReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
