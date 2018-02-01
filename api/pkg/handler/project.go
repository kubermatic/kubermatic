package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// Project resources live in
type Project struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
	RoleNames []string `json:"roleNames"`
}

type projectReq struct {
	Project Project
}

// ProjectList a list of full projects
type ProjectList struct {
	Projects []Project `json:"projects"`
}

// Member is a virtual user in a project
type Member struct {
	ID          string   `json:"id"`
	MemberEmail string   `json:"memberEmail"`
	RoleNames   []string `json:"roleNames"`
}

type memberReq struct {
	Member Member
}

// MemberList a list of members
type MemberList struct {
	ProjectMembers []Member `json:"projectMembers"`
}

// Role specifies the permissions a user has
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// projectPathReq represent a request for a project
type projectPathReq struct {
	ProjectID string `json:"project_id"`
}

// memberPathReq represent a request for a member
type memberPathReq struct {
	MemberID string `json:"member_id"`
}

func decodeMemberPathReqInto(c context.Context, req *memberPathReq, r *http.Request) error {
	var ok bool
	pReq, err := decodeMemberPathReq(c, r)
	if err != nil {
		return err
	}
	*req, ok = pReq.(memberPathReq)
	if !ok {
		return errors.NewBadRequest("bad member request")
	}
	return nil
}

func decodeProjectPathReqInto(c context.Context, req *projectPathReq, r *http.Request) error {
	var ok bool
	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return err
	}
	*req, ok = pReq.(projectPathReq)
	if !ok {
		return errors.NewBadRequest("bad project request")
	}
	return nil
}

func decodeMemberPathReq(c context.Context, r *http.Request) (interface{}, error) {
	var req memberPathReq
	req.MemberID = mux.Vars(r)["member_id"]
	return req, nil
}

func decodeMemberBodyReq(c context.Context, r *http.Request) (interface{}, error) {
	var p Member
	var _ memberReq
	err := json.NewDecoder(r.Body).Decode(&p)
	return p, err
}

func decodeProjectBodyReq(c context.Context, r *http.Request) (interface{}, error) {
	var p Project
	var _ projectReq
	err := json.NewDecoder(r.Body).Decode(&p)
	return p, err
}

func getProjectMeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Member{}, nil
	}
}

func decodeProjectPathReq(c context.Context, r *http.Request) (interface{}, error) {
	var req projectPathReq
	req.ProjectID = mux.Vars(r)["project_id"]
	return req, nil
}

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

type updateProjectReq struct {
	projectPathReq
	Project
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	var req updateProjectReq
	var err error
	var ok bool

	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.projectPathReq, ok = pReq.(projectPathReq)
	if !ok {
		return nil, errors.NewBadRequest("bad project request")
	}

	pbReq, err := decodeProjectBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Project, ok = pbReq.(Project)
	if !ok {
		return nil, errors.NewBadRequest("bad project body type request")
	}

	return req, nil
}

func updateProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Project{}, nil
	}
}

type createProjectReq struct {
	Project
}

func decodeCreateProject(c context.Context, r *http.Request) (interface{}, error) {
	var req createProjectReq
	var ok bool
	pbReq, err := decodeProjectBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Project, ok = pbReq.(Project)
	if !ok {
		return nil, errors.NewBadRequest("bad project body type request")
	}

	return req, nil
}

func createProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Project{}, nil
	}
}

func getProjectMembersEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return MemberList{}, nil
	}
}

type deleteProjectMemberReq struct {
	projectPathReq
	memberPathReq
}

func decodeDeleteProjectMember(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteProjectMemberReq
	var err error
	var ok bool

	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.projectPathReq, ok = pReq.(projectPathReq)
	if !ok {
		return nil, errors.NewBadRequest("bad project request")
	}

	mpReq, err := decodeMemberPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.memberPathReq, ok = mpReq.(memberPathReq)
	if !ok {
		return nil, errors.NewBadRequest("bad member request")
	}

	return req, nil
}

func deleteProjectMemberEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Don't return member just success.
		return nil, nil
	}
}

type addProjectMemberReq struct {
	projectPathReq
	Member
}

func decodeAddProjectMember(c context.Context, r *http.Request) (interface{}, error) {
	var req addProjectMemberReq
	var err error
	var ok bool

	pReq, err := decodeProjectPathReq(c, r)
	if err != nil {
		return nil, err
	}
	req.projectPathReq, ok = pReq.(projectPathReq)
	if !ok {
		return nil, errors.NewBadRequest("bad project request")
	}

	mpReq, err := decodeMemberBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Member, ok = mpReq.(Member)
	if !ok {
		return nil, errors.NewBadRequest("bad member request")
	}

	return req, nil
}

func addProjectMemberEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Member{}, nil
	}
}

type updateProjectMemberReq struct {
	projectPathReq
	memberPathReq
	Member
}

func decodeUpdateProjectMember(c context.Context, r *http.Request) (interface{}, error) {
	var req updateProjectMemberReq
	var err error
	var ok bool

	err = decodeProjectPathReqInto(c, &req.projectPathReq, r)
	if err != nil {
		return nil, err
	}
	err = decodeMemberPathReqInto(c, &req.memberPathReq, r)
	if err != nil {
		return nil, err
	}

	mReq, err := decodeMemberBodyReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Member, ok = mReq.(Member)
	if !ok {
		return nil, errors.NewBadRequest("bad member request")
	}

	return req, nil
}

func updateProjectMemberEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Member{}, nil
	}
}
