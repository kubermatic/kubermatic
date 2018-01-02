package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

type Project struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
	RoleNames []string `json:"role_names"`
}

type ProjectList struct {
	projects []Project `json:"projects"`
}

type Member struct {
	ID          string   `json:"id"`
	MemberEmail string   `json:"member_email"`
	RoleNames   []string `json:"role_names"`
}

type MemberList struct {
	ProjectMembers []Member `json:"project_members"`
}

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleList struct {
	Roles []Role `json:"roles"`
}

type MemberRoles struct {
	RoleNames []string `json:"role_names"`
}

type ProjectPathReq struct {
	ProjectID string
}

type MemberPathReq struct {
	MemberID string
}

type DeleteProjectReq struct {
	ProjectPathReq
}

func decodeMemberPathReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterReq
	req.Cluster = mux.Vars(r)["member_id"]
	return req, nil
}

func decodeMemberBodyReq(c context.Context, r *http.Request) (interface{}, error) {
	var p Member
	err := json.NewDecoder(r.Body).Decode(&p)
	return p, err
}

func decodeProjectBodyReq(c context.Context, r *http.Request) (interface{}, error) {
	var p Project
	err := json.NewDecoder(r.Body).Decode(&p)
	return p, err
}

// Project member self information (me) endpoint
func getProjectMeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Member{}, nil
	}
}

func decodeProjectPathReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterReq
	req.Cluster = mux.Vars(r)["project_id"]
	return req, nil
}

// Project endpoints
func getProjectsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return ProjectList{}, nil
	}
}

func deleteProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Don't return project just success.
		return nil, nil
	}
}

// Update Project
//
//
type UpdateProjectReq struct {
	ProjectPathReq
	Project
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	var req UpdateProjectReq
	var err error
	var ok bool

	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectPathReq, ok = pReq.(ProjectPathReq)
	if !ok {
		return nil, errors.NewBadRequest("Bad project request")
	}

	pbReq, err := decodeProjectBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Project, ok = pbReq.(Project)
	if !ok {
		return nil, errors.NewBadRequest("Bad project body type request")
	}

	return req, nil
}
func updateProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Project{}, nil
	}
}

// Create Project
//
//
type CreateProjectReq struct {
	Project
}

func decodeCreateProject(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateProjectReq
	var ok bool
	pbReq, err := decodeProjectBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Project, ok = pbReq.(Project)
	if !ok {
		return nil, errors.NewBadRequest("Bad project body type request")
	}

	return req, nil
}
func createProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Project{}, nil
	}
}

// Get Project Members
//
//
func getProjectMembersEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return MemberList{}, nil
	}
}

// Delete Project Members
//
//
type DeleteProjectMemberReq struct {
	ProjectPathReq
	MemberPathReq
}

func decodeDeleteProjectMember(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteProjectMemberReq
	var err error
	var ok bool

	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectPathReq, ok = pReq.(ProjectPathReq)
	if !ok {
		return nil, errors.NewBadRequest("Bad project request")
	}

	mpReq, err := decodeMemberPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.MemberPathReq, ok = mpReq.(MemberPathReq)
	if !ok {
		return nil, errors.NewBadRequest("Bad member request")
	}

	return req, nil
}
func deleteProjectMemberEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Don't return member just success.
		return nil, nil
	}
}

// Add Project Member
//
//
type AddProjectMemberReq struct {
	ProjectPathReq
	Member
}

func decodeAddProjectMember(c context.Context, r *http.Request) (interface{}, error) {
	var req AddProjectMemberReq
	var err error
	var ok bool

	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectPathReq, ok = pReq.(ProjectPathReq)
	if !ok {
		return nil, errors.NewBadRequest("Bad project request")
	}

	mpReq, err := decodeMemberBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Member, ok = mpReq.(Member)
	if !ok {
		return nil, errors.NewBadRequest("Bad member request")
	}

	return req, nil
}
func addProjectMemberEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Member{}, nil
	}
}

// Update Project Member
//
//
type UpdateProjectMemberReq struct {
	ProjectPathReq
	MemberPathReq
	Member
}

func decodeUpdateProjectMember(c context.Context, r *http.Request) (interface{}, error) {
	var req UpdateProjectMemberReq
	var err error
	var ok bool

	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectPathReq, ok = pReq.(ProjectPathReq)
	if !ok {
		return nil, errors.NewBadRequest("Bad project request")
	}

	mpReq, err := decodeMemberPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.MemberPathReq, ok = mpReq.(MemberPathReq)
	if !ok {
		return nil, errors.NewBadRequest("Bad member request")
	}

	mReq, err := decodeMemberBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Member, ok = mReq.(Member)
	if !ok {
		return nil, errors.NewBadRequest("Bad member request")
	}

	return req, nil
}
func updateProjectMemberEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Member{}, nil
	}
}
