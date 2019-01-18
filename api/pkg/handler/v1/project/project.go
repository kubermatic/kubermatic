package project

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

// CreateEndpoint defines an HTTP endpoint that creates a new project in the system
func CreateEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		projectRq, ok := request.(projectReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(projectRq.Name) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
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

// ListEndpoint defines an HTTP endpoint for listing projects
func ListEndpoint(projectProvider provider.ProjectProvider, memberMapper provider.ProjectMemberMapper) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)
		projects := []*apiv1.Project{}

		userMappings, err := memberMapper.MappingsFor(user.Spec.Email)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		for _, mapping := range userMappings {
			userInfo := &provider.UserInfo{Email: mapping.Spec.UserEmail, Group: mapping.Spec.Group}
			projectInternal, err := projectProvider.Get(userInfo, mapping.Spec.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			projects = append(projects, convertInternalProjectToExternal(projectInternal))
		}

		return projects, nil
	}
}

// DeleteEndpoint defines an HTTP endpoint for deleting a project
func DeleteEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteRq)
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

// UpdateEndpoint defines an HTTP endpoint that updates an existing project in the system
// in the current implementation only project renaming is supported
func UpdateEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateRq)
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
		user := ctx.Value(middleware.UserCRContextKey).(*kubermaticapiv1.User)

		kubermaticProject, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		alreadyExistingProjects, err := projectProvider.List(&provider.ProjectListOptions{ProjectName: req.Name, OwnerUID: user.UID})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(alreadyExistingProjects) > 0 {
			return nil, errors.NewAlreadyExists("project name", req.Name)
		}

		kubermaticProject.Spec.Name = req.Name
		project, err := projectProvider.Update(userInfo, kubermaticProject)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalProjectToExternal(project), nil
	}
}

// GeEndpoint defines an HTTP endpoint for getting a project
func GetEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetProjectRq)
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

// updateRq defines HTTP request for updateProject
// swagger:parameters updateProject
type updateRq struct {
	common.ProjectReq
	projectReq
}

// DecodeUpdateRq decodes an HTTP request into updateRq
func DecodeUpdateRq(c context.Context, r *http.Request) (interface{}, error) {
	pReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	cReq, err := DecodeCreate(c, r)
	if err != nil {
		return nil, err
	}

	return updateRq{
		pReq.(common.ProjectReq),
		cReq.(projectReq),
	}, nil
}

type projectReq struct {
	Name string `json:"name"`
}

// DecodeCreate decodes an HTTP request into projectReq
func DecodeCreate(c context.Context, r *http.Request) (interface{}, error) {
	var req projectReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

// deleteRq defines HTTP request for deleteProject endpoint
// swagger:parameters deleteProject
type deleteRq struct {
	common.ProjectReq
}

// DecodeDelete decodes an HTTP request into deleteRq
func DecodeDelete(c context.Context, r *http.Request) (interface{}, error) {
	req, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, nil
	}
	return deleteRq{ProjectReq: req.(common.ProjectReq)}, err
}
