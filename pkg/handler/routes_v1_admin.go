/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/admin"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
)

// RegisterV1Admin declares all router paths for the admin users
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

	mux.Methods(http.MethodGet).
		Path("/admin/settings/customlinks").
		Handler(r.getKubermaticCustomLinks())

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

	mux.Methods(http.MethodPut).
		Path("/admin/metering/credentials").
		Handler(r.createOrUpdateMeteringCredentials())

	mux.Methods(http.MethodPut).
		Path("/admin/metering/configurations").
		Handler(r.createOrUpdateMeteringConfigurations())
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.KubermaticSettingsEndpoint(r.settingsProvider)),
		common.DecodeEmptyReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/settings/customlinks admin getKubermaticCustomLinks
//
//     Gets the custom links.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: GlobalCustomLinks
//       401: empty
//       403: empty
func (r Routing) getKubermaticCustomLinks() http.Handler {
	return httptransport.NewServer(
		admin.KubermaticCustomLinksEndpoint(r.settingsProvider),
		common.DecodeEmptyReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateKubermaticSettingsEndpoint(r.userInfoGetter, r.settingsProvider)),
		admin.DecodePatchKubermaticSettingsReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.GetAdminEndpoint(r.userInfoGetter, r.adminProvider)),
		common.DecodeEmptyReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.SetAdminEndpoint(r.userInfoGetter, r.adminProvider)),
		admin.DecodeSetAdminReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.ListAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		common.DecodeEmptyReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.GetAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		admin.DecodeAdmissionPluginReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.DeleteAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		admin.DecodeAdmissionPluginReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateAdmissionPluginEndpoint(r.userInfoGetter, r.admissionPluginProvider)),
		admin.DecodeUpdateAdmissionPluginReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.ListSeedEndpoint(r.userInfoGetter, r.seedsGetter)),
		common.DecodeEmptyReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.GetSeedEndpoint(r.userInfoGetter, r.seedsGetter)),
		admin.DecodeSeedReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateSeedEndpoint(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeUpdateSeedReq,
		EncodeJSON,
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
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.DeleteSeedEndpoint(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeSeedReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/admin/metering/credentials admin updateOrCreateMeteringCredentials
//
//     Creates or updates the metering tool credentials.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) createOrUpdateMeteringCredentials() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.CreateOrUpdateMeteringCredentials(r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeMeteringReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/admin/metering/configurations admin createOrUpdateMeteringConfigurations
//
//     Configures KKP metering tool.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) createOrUpdateMeteringConfigurations() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.CreateOrUpdateMeteringConfigurations(r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeMeteringConfigurationsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}
