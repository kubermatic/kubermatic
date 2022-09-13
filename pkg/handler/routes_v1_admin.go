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

// RegisterV1Admin declares all router paths for the admin users.
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
	mux.Methods(http.MethodPost).
		Path("/admin/seeds").
		Handler(r.createSeed())

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

	mux.Methods(http.MethodDelete).
		Path("/admin/seeds/{seed_name}/backupdestinations/{backup_destination}").
		Handler(r.deleteBackupDestination())

	// Defines a set of HTTP endpoints for metering tool
	mux.Methods(http.MethodPut).
		Path("/admin/metering/credentials").
		Handler(r.createOrUpdateMeteringCredentials())

	mux.Methods(http.MethodPut).
		Path("/admin/metering/configurations").
		Handler(r.createOrUpdateMeteringConfigurations())

	mux.Methods(http.MethodGet).
		Path("/admin/metering/configurations/reports/{name}").
		Handler(r.GetMeteringReportConfiguration())

	mux.Methods(http.MethodGet).
		Path("/admin/metering/configurations/reports").
		Handler(r.ListMeteringReportConfigurations())

	mux.Methods(http.MethodPost).
		Path("/admin/metering/configurations/reports/{name}").
		Handler(r.CreateMeteringReportConfiguration())

	mux.Methods(http.MethodPut).
		Path("/admin/metering/configurations/reports/{name}").
		Handler(r.UpdateMeteringReportConfiguration())

	mux.Methods(http.MethodDelete).
		Path("/admin/metering/configurations/reports/{name}").
		Handler(r.DeleteMeteringReportConfiguration())

	mux.Methods(http.MethodGet).
		Path("/admin/metering/reports").
		Handler(r.listMeteringReports())

	mux.Methods(http.MethodGet).
		Path("/admin/metering/reports/{report_name}").
		Handler(r.getMeteringReport())

	mux.Methods(http.MethodDelete).
		Path("/admin/metering/reports/{report_name}").
		Handler(r.deleteMeteringReport())
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

// swagger:route POST /api/v1/admin/seeds admin createSeed
//
//     Creates a new seed object.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Seed
//       401: empty
//       403: empty
func (r Routing) createSeed() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.CreateSeedEndpoint(r.userInfoGetter, r.seedsGetter, r.seedProvider)),
		admin.DecodeCreateSeedReq,
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
		)(admin.UpdateSeedEndpoint(r.userInfoGetter, r.seedsGetter, r.seedProvider)),
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
		)(admin.DeleteSeedEndpoint(r.userInfoGetter, r.seedsGetter, r.masterClient)),
		admin.DecodeSeedReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/admin/seeds/{seed_name}/backupdestinations/{backup_destination} admin deleteBackupDestination
//
//     Deletes a backup destination from the Seed.
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) deleteBackupDestination() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.DeleteBackupDestinationEndpoint(r.userInfoGetter, r.seedsGetter, r.masterClient)),
		admin.DecodeBackupDestinationReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/admin/metering/credentials admin createOrUpdateMeteringCredentials
//
//     Creates or updates the metering tool credentials. Only available in Kubermatic Enterprise Edition
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
		)(admin.CreateOrUpdateMeteringCredentials(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeMeteringSecretReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/admin/metering/configurations admin createOrUpdateMeteringConfigurations
//
//     Configures KKP metering tool. Only available in Kubermatic Enterprise Edition
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
		)(admin.CreateOrUpdateMeteringConfigurations(r.userInfoGetter, r.seedsGetter, r.masterClient)),
		admin.DecodeMeteringConfigurationsReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/metering/configurations/reports/{name} admin getMeteringReportConfiguration
//
//     Gets report configuration for KKP metering tool. Only available in Kubermatic Enterprise Edition
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: MeteringReportConfiguration
//       401: empty
//       403: empty
func (r Routing) GetMeteringReportConfiguration() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.GetMeteringReportConfigurationEndpoint(r.userInfoGetter, r.seedsGetter)),
		admin.DecodeGetMeteringReportConfigurationReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/metering/configurations/reports admin listMeteringReportConfigurations
//
//     Lists report configurations for KKP metering tool. Only available in Kubermatic Enterprise Edition
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []MeteringReportConfiguration
//       401: empty
//       403: empty
func (r Routing) ListMeteringReportConfigurations() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.ListMeteringReportConfigurationsEndpoint(r.userInfoGetter, r.seedsGetter)),
		common.DecodeEmptyReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/admin/metering/configurations/reports/{name} admin createMeteringReportConfiguration
//
//     Creates report configuration for KKP metering tool. Only available in Kubermatic Enterprise Edition
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       201: MeteringReportConfiguration
//       401: empty
//       403: empty
func (r Routing) CreateMeteringReportConfiguration() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.CreateMeteringReportConfigurationEndpoint(r.userInfoGetter, r.seedsGetter, r.masterClient)),
		admin.DecodeCreateMeteringReportConfigurationReq,
		SetStatusCreatedHeader(EncodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route PUT /api/v1/admin/metering/configurations/reports/{name} admin updateMeteringReportConfiguration
//
//     Updates existing report configuration for KKP metering tool. Only available in Kubermatic Enterprise Edition
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: MeteringReportConfiguration
//       401: empty
//       403: empty
func (r Routing) UpdateMeteringReportConfiguration() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.UpdateMeteringReportConfigurationEndpoint(r.userInfoGetter, r.seedsGetter, r.masterClient)),
		admin.DecodeUpdateMeteringReportConfigurationReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/admin/metering/configurations/reports/{name} admin deleteMeteringReportConfiguration
//
//     Removes report configuration for KKP metering tool. Only available in Kubermatic Enterprise Edition
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
//       401: empty
//       403: empty
func (r Routing) DeleteMeteringReportConfiguration() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.DeleteMeteringReportConfigurationEndpoint(r.userInfoGetter, r.seedsGetter, r.masterClient)),
		admin.DecodeDeleteMeteringReportConfigurationReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/admin/metering/reports metering reports listMeteringReports
//
//     List metering reports. Only available in Kubermatic Enterprise Edition
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []MeteringReport
//       401: empty
//       403: empty
func (r Routing) listMeteringReports() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.ListMeteringReportsEndpoint(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeListMeteringReportReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

//swagger:route GET /api/v1/admin/metering/reports/{report_name} metering report getMeteringReport
//
//    Download a specific metering report. Provides an S3 pre signed URL valid for 1 hour. Only available in Kubermatic Enterprise Edition
//
//    Produces:
//    - application/json
//
//    Responses:
//      default: errorResponse
//      200: MeteringReportURL
//      401: empty
//      403: empty
func (r Routing) getMeteringReport() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.GetMeteringReportEndpoint(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeGetMeteringReportReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}

//swagger:route DELETE /api/v1/admin/metering/reports/{report_name} metering report deleteMeteringReport
//
//    Removes a specific metering report. Only available in Kubermatic Enterprise Edition
//
//    Produces:
//    - application/json
//
//    Responses:
//      default: errorResponse
//      200: empty
//      401: empty
//      403: empty
func (r Routing) deleteMeteringReport() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			middleware.TokenVerifier(r.tokenVerifiers, r.userProvider),
			middleware.UserSaver(r.userProvider),
		)(admin.DeleteMeteringReportEndpoint(r.userInfoGetter, r.seedsGetter, r.seedsClientGetter)),
		admin.DecodeDeleteMeteringReportReq,
		EncodeJSON,
		r.defaultServerOptions()...,
	)
}
