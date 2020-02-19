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
		Path("/admin").
		Handler(r.getAdmins())

	mux.Methods(http.MethodPut).
		Path("/admin").
		Handler(r.setAdmin())

	mux.Methods(http.MethodGet).
		Path("/admin/settings").
		Handler(r.getKubermaticSettings())

	mux.Methods(http.MethodPatch).
		Path("/admin/settings").
		Handler(r.patchKubermaticSettings())

	// Defines a set of HTTP endpoints for the admission plugins
	mux.Methods(http.MethodGet).
		Path("/admin/admission/plugins").
		Handler(r.listAdmissionPlugins())

	mux.Methods(http.MethodGet).
		Path("/admin/admission/plugins/{name}").
		Handler(r.getAdmissionPlugin())

	mux.Methods(http.MethodDelete).
		Path("/admin/admission/plugins/{name}").
		Handler(r.deleteAdmissionPlugin())

	mux.Methods(http.MethodPatch).
		Path("/admin/admission/plugins/{name}").
		Handler(r.updateAdmissionPlugin())

	// Defines a set of HTTP endpoints for the seeds
	mux.Methods(http.MethodGet).
		Path("/admin/seeds").
		Handler(r.listSeeds())

	mux.Methods(http.MethodGet).
		Path("/admin/seeds/{seed_name}").
		Handler(r.getSeed())

	mux.Methods(http.MethodPatch).
		Path("/admin/seeds/{seed_name}").
		Handler(r.updateSeed())

	mux.Methods(http.MethodDelete).
		Path("/admin/seeds/{seed_name}").
		Handler(r.deleteSeed())
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
		)(admin.KubermaticSettingsEndpoint(r.settingsProvider)),
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

// swagger:route GET /api/v1/admin admin getAdmins
//
//     Returns list of admin users.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Admin
//       401: empty
//       403: empty
func (r Routing) getAdmins() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.GetAdminEndpoint(r.userInfoGetter, r.adminProvider)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/admin admin setAdmin
//
//     Allows setting and clearing admin role for users.
//
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Admin
//       401: empty
//       403: empty
func (r Routing) setAdmin() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.SetAdminEndpoint(r.userInfoGetter, r.adminProvider)),
		admin.DecodeSetAdminReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/admission/plugins admin listAdmissionPlugins
//
//     Returns all admission plugins from the CRDs.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []AdmissionPlugin
//       401: empty
//       403: empty
func (r Routing) listAdmissionPlugins() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.ListAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/admission/plugins/{name} admin getAdmissionPlugin
//
//     Gets the admission plugin.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AdmissionPlugin
//       401: empty
//       403: empty
func (r Routing) getAdmissionPlugin() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.GetAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		admin.DecodeAdmissionPluginReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/admin/admission/plugins/{name} admin deleteAdmissionPlugin
//
//     Deletes the admission plugin.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteAdmissionPlugin() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.DeleteAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		admin.DecodeAdmissionPluginReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v1/admin/admission/plugins/{name} admin updateAdmissionPlugin
//
//     Updates the admission plugin.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: AdmissionPlugin
//       401: empty
//       403: empty
func (r Routing) updateAdmissionPlugin() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		admin.DecodeUpdateAdmissionPluginReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/seeds admin listSeeds
//
//     Returns all seeds from the CRDs.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []Seed
//       401: empty
//       403: empty
func (r Routing) listSeeds() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.ListSeedEndpoint(r.userInfoGetter, r.seedsGetter)),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/seeds/{seed_name} admin getSeed
//
//     Returns the seed object.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Seed
//       401: empty
//       403: empty
func (r Routing) getSeed() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.GetSeedEndpoint(r.userInfoGetter, r.seedsGetter)),
		admin.DecodeSeedReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PATCH /api/v1/admin/seeds/{seed_name} admin updateSeed
//
//     Updates the seed.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Seed
//       401: empty
//       403: empty
func (r Routing) updateSeed() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateSeedEndpoint(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeUpdateSeedReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/admin/seeds/{seed_name} admin deleteSeed
//
//     Deletes the seed CRD object from the Kubermatic.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteSeed() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers),
			middleware.UserSaver(r.userProvider),
		)(admin.DeleteSeedEndpoint(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeSeedReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
