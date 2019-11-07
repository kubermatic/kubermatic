package handler

import (
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/admin"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
)

//RegisterV1Admin declares all router paths for the admin users
func (r Routing) RegisterV1Admin(mux *mux.Router) {
	//
	// Defines a set of HTTP endpoints for the admin users
	mux.Methods(http.MethodGet).
		Path("/admin/settings").
		Handler(r.getKubermaticSettings())
}

// swagger:route GET /api/v1/admin/settings admin
//
//     Gets a kubermatic settings.
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
