package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// createProjectEndpoint defines an HTTP endpoint that creates a new project in the system
func createProjectEndpoint(projectProvider provider.ProjectProvider, memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		projectRq, ok := request.(projectReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(projectRq.Name) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		existingProject, err := listUserProjects(userInfo.Email, projectProvider, memberMapper, &UserProjectsListOptions{ProjectName: projectRq.Name})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(existingProject) > 0 {
			return nil, errors.NewAlreadyExists("project name", projectRq.Name)
		}

		user := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)
		kubermaticProject, err := projectProvider.New(user, projectRq.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return apiv1.Project{
			ObjectMeta: apiv1.ObjectMeta{
				ID:                kubermaticProject.Name,
				Name:              kubermaticProject.Spec.Name,
				CreationTimestamp: apiv1.NewTime(kubermaticProject.CreationTimestamp.Time),
			},
			Status: kubermaticProject.Status.Phase,
		}, nil
	}
}

func listProjectsEndpoint(projectProvider provider.ProjectProvider, memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		internalUserProjects, err := listUserProjects(userInfo.Email, projectProvider, memberMapper, &UserProjectsListOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		projects := []*apiv1.Project{}
		for _, projectInternal := range internalUserProjects {
			projects = append(projects, convertInternalProjectToExternal(projectInternal))
		}

		return projects, nil
	}
}

func deleteProjectEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(DeleteProjectRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the id of the project cannot be empty")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		err := projectProvider.Delete(userInfo, req.ProjectID)
		return nil, common.KubernetesErrorToHTTPError(err)
	}
}

// updateProjectEndpoint defines an HTTP endpoint that updates an existing project in the system
// in the current implementation only project renaming is supported
func updateProjectEndpoint(projectProvider provider.ProjectProvider, memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(UpdateProjectRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the id of the project cannot be empty")
		}
		if len(req.Name) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		kubermaticProject, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingProject, err := listUserProjects(userInfo.Email, projectProvider, memberMapper, &UserProjectsListOptions{ProjectName: req.Name})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(existingProject) > 0 {
			return nil, errors.NewAlreadyExists("project name", req.Name)
		}

		kubermaticProject.Spec.Name = req.Name

		project, err := projectProvider.Update(userInfo, kubermaticProject)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return project, nil
	}
}

func getProjectEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GetProjectRq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, errors.NewBadRequest("the id of the project cannot be empty")
		}

		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		kubermaticProject, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalProjectToExternal(kubermaticProject), nil
	}
}

func convertInternalProjectToExternal(kubermaticProject *kubermaticapiv1.Project) *apiv1.Project {
	return &apiv1.Project{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                kubermaticProject.Name,
			Name:              kubermaticProject.Spec.Name,
			CreationTimestamp: apiv1.NewTime(kubermaticProject.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if kubermaticProject.DeletionTimestamp != nil {
					dt := apiv1.NewTime(kubermaticProject.DeletionTimestamp.Time)
					return &dt
				}
				return nil
			}(),
		},
		Status: kubermaticProject.Status.Phase,
	}
}

// GetProjectRq defines HTTP request for getProject endpoint
// swagger:parameters getProject getUsersForProject
type GetProjectRq struct {
	common.ProjectReq
}

func decodeGetProject(c context.Context, r *http.Request) (interface{}, error) {
	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	return GetProjectRq{projectReq.(common.ProjectReq)}, nil
}

// UpdateProjectRq defines HTTP request for updateProject
// swagger:parameters updateProject
type UpdateProjectRq struct {
	common.ProjectReq
	projectReq
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	pReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	cReq, err := decodeCreateProject(c, r)
	if err != nil {
		return nil, err
	}

	return UpdateProjectRq{
		pReq.(common.ProjectReq),
		cReq.(projectReq),
	}, nil
}

type projectReq struct {
	Name string `json:"name"`
}

func decodeCreateProject(c context.Context, r *http.Request) (interface{}, error) {
	var req projectReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

// DeleteProjectRq defines HTTP request for deleteProject endpoint
// swagger:parameters deleteProject
type DeleteProjectRq struct {
	common.ProjectReq
}

func decodeDeleteProject(c context.Context, r *http.Request) (interface{}, error) {
	req, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, nil
	}
	return DeleteProjectRq{ProjectReq: req.(common.ProjectReq)}, err
}

// UserProjectsListOptions allows to set filters for listing user projects
type UserProjectsListOptions struct {
	// ProjectName set the project name for the given user
	ProjectName string
}

func listUserProjects(email string, projectProvider provider.ProjectProvider, memberMapper provider.ProjectMemberMapper, options *UserProjectsListOptions) ([]*kubermaticapiv1.Project, error) {
	userMappings, err := memberMapper.MappingsFor(email)
	if err != nil {
		return nil, err
	}

	userProjects := []*kubermaticapiv1.Project{}
	for _, mapping := range userMappings {
		userInfo := &provider.UserInfo{Email: email, Group: mapping.Spec.Group}
		project, err := projectProvider.Get(userInfo, mapping.Spec.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
		if err != nil {
			return nil, err
		}
		userProjects = append(userProjects, project)
	}

	if options == nil {
		return userProjects, nil
	}
	if len(options.ProjectName) == 0 {
		return userProjects, nil
	}

	filteredUserProjects := []*kubermaticapiv1.Project{}
	for _, project := range userProjects {
		if len(options.ProjectName) != 0 && project.Spec.Name == options.ProjectName {
			filteredUserProjects = append(filteredUserProjects, project)
			break
		}
	}

	return filteredUserProjects, nil
}
