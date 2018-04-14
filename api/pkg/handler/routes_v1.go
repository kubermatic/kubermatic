package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// RegisterV1 declares all router paths for v1
func (r Routing) RegisterV1(mux *mux.Router) {
	// swagger:route GET /api/v1/healthz healthz
	//
	// Health endpoint
	//
	//     Responses:
	//       default: empty
	mux.Methods(http.MethodGet).
		Path("/healthz").
		HandlerFunc(StatusOK)

	mux.Methods(http.MethodGet).
		Path("/dc").
		Handler(r.datacentersHandler())

	mux.Methods(http.MethodGet).
		Path("/dc/{dc}").
		Handler(r.datacenterHandler())

	mux.Methods(http.MethodGet).
		Path("/ssh-keys").
		Handler(r.listSSHKeys())

	mux.Methods(http.MethodGet).
		Path("/user").
		Handler(r.getUser())

	mux.Methods(http.MethodPost).
		Path("/ssh-keys").
		Handler(r.createSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/ssh-keys/{meta_name}").
		Handler(r.deleteSSHKey())

	mux.Methods(http.MethodGet).
		Path("/digitalocean/sizes").
		Handler(r.listDigitaloceanSizes())

	mux.Methods(http.MethodGet).
		Path("/openstack/sizes").
		Handler(r.listOpenstackSizes())

	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/ssh-keys").
		Handler(r.listProjectSSHKeys())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/ssh-keys").
		Handler(r.createProjectSSHKey())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/ssh-keys/{meta_name}").
		Handler(r.deleteProjectSSHKey())

	// Member and organization endpoints
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/me").
		Handler(r.getProjectMe())

	// Project management
	mux.Methods(http.MethodGet).
		Path("/projects").
		Handler(r.getProjects())

	mux.Methods(http.MethodPost).
		Path("projects/{project_id}/projects").
		Handler(r.createProject())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}").
		Handler(r.updateProject())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}").
		Handler(r.deleteProject())

	// Members in project
	mux.Methods(http.MethodGet).
		Path("/projects/{project_id}/members").
		Handler(r.getProjectMembers())

	mux.Methods(http.MethodPut).
		Path("/projects/{project_id}/members").
		Handler(r.updateProjectMember())

	mux.Methods(http.MethodPost).
		Path("/projects/{project_id}/member").
		Handler(r.addProjectMember())

	mux.Methods(http.MethodDelete).
		Path("/projects/{project_id}/member/{member_id}").
		Handler(r.deleteProjectMember())
}

// swagger:route GET /api/v1/ssh-keys ssh-keys listSSHKeys
//
// Lists SSH keys from the user
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: SSHKey
func (r Routing) listSSHKeys() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(listSSHKeyEndpoint(r.sshKeyProvider)),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route POST /api/v1/ssh-keys ssh-keys createSSHKey
//
// Creates a SSH keys for the user
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: SSHKey
func (r Routing) createSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(createSSHKeyEndpoint(r.sshKeyProvider)),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

// swagger:route DELETE /api/v1/ssh-keys/{meta_name} ssh-keys deleteSSHKey
//
// Deletes a SSH keys for the user
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: empty
func (r Routing) deleteSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteSSHKeyEndpoint(r.sshKeyProvider)),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/digitalocean/sizes digitalocean listDigitaloceanSizes
//
// Lists sizes from digitalocean
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: DigitaloceanSizeList
func (r Routing) listDigitaloceanSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(digitaloceanSizeEndpoint()),
		decodeDoSizesReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/openstack/sizes openstack listOpenstackSizes
//
// Lists sizes from openstack
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: []OpenstackSize
func (r Routing) listOpenstackSizes() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(openstackSizeEndpoint(r.cloudProviders)),
		decodeOpenstackSizeReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// swagger:route GET /api/v1/dc datacenter listDatacenters
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: DatacenterList
func (r Routing) datacentersHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(datacentersEndpoint(r.datacenters)),
		decodeDatacentersReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

// Get the datacenter
// swagger:route GET /api/v1/dc/{dc} datacenter getDatacenter
//
//     Produces:
//     - application/json
//
//     Responses:
//       default: errorResponse
//       200: Datacenter
func (r Routing) datacenterHandler() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(datacenterEndpoint(r.datacenters)),
		decodeDcReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getUser() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getUserHandler()),
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectMe() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getProjectMeEndpoint()),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjects() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getProjectsEndpoint()),
		// We don't have to write a decoder only for a request without incoming information
		decodeEmptyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(createProjectEndpoint()),
		decodeCreateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) updateProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(updateProjectEndpoint()),
		decodeUpdateProject,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProject() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteProjectEndpoint()),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) getProjectMembers() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(getProjectMembersEndpoint()),
		decodeProjectPathReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) addProjectMember() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(addProjectMemberEndpoint()),
		decodeAddProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectMember() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteProjectMemberEndpoint()),
		decodeDeleteProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) updateProjectMember() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(updateProjectMemberEndpoint()),
		decodeUpdateProjectMember,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) listProjectSSHKeys() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(listSSHKeyEndpoint(r.sshKeyProvider)),
		decodeListSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}

func (r Routing) createProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(createSSHKeyEndpoint(r.sshKeyProvider)),
		decodeCreateSSHKeyReq,
		createStatusResource(encodeJSON),
		r.defaultServerOptions()...,
	)
}

func (r Routing) deleteProjectSSHKey() http.Handler {
	return httptransport.NewServer(
		endpoint.Chain(
			r.authenticator.Verifier(),
			r.userSaverMiddleware(),
		)(deleteSSHKeyEndpoint(r.sshKeyProvider)),
		decodeDeleteSSHKeyReq,
		encodeJSON,
		r.defaultServerOptions()...,
	)
}
