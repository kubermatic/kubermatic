package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// createProjectEndpoint defines an HTTP endpoint that creates a new project in the system
func createProjectEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		projectRq, ok := request.(projectReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(projectRq.Name) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		kubermaticProject, err := projectProvider.New(user, projectRq.Name)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		return apiv1.Project{
			NewObjectMeta: apiv1.NewObjectMeta{
				ID:                kubermaticProject.Name,
				Name:              kubermaticProject.Spec.Name,
				CreationTimestamp: kubermaticProject.CreationTimestamp.Time,
			},
			Status: kubermaticProject.Status.Phase,
		}, nil
	}
}

func listProjectsEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)

		// old approach were we stored info about bindings in the user resource object
		projects := []*apiv1.Project{}
		for _, pg := range user.Spec.Projects {
			userInfo := &provider.UserInfo{Email: user.Spec.Email, Group: pg.Group}
			projectInternal, err := projectProvider.Get(userInfo, pg.Name, &provider.ProjectGetOptions{})
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
			projects = append(projects, convertInternalProjectToExternal(projectInternal))
		}

		// TODO: implement new approach
		// TODO: remove old apprach when we migrate to the new one

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
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		err := projectProvider.Delete(userInfo, req.ProjectID)
		return nil, kubernetesErrorToHTTPError(err)
	}
}

func updateProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return apiv1.Project{}, errors.NewNotImplemented()
	}
}

func getProjectEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		kubermaticProject, err := getKubermaticProject(ctx, projectProvider, request)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return convertInternalProjectToExternal(kubermaticProject), nil
	}
}

func getKubermaticProject(ctx context.Context, projectProvider provider.ProjectProvider, request interface{}) (*kubermaticapiv1.Project, error) {
	req, ok := request.(GetProjectRq)

	if !ok {
		return nil, errors.NewBadRequest("invalid request")
	}

	if len(req.ProjectID) == 0 {
		return nil, errors.NewBadRequest("the name of the project cannot be empty")
	}

	userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
	kubermaticProject, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
	if err != nil {
		return nil, kubernetesErrorToHTTPError(err)
	}
	return kubermaticProject, nil
}

func convertInternalProjectToExternal(kubermaticProject *kubermaticapiv1.Project) *apiv1.Project {
	return &apiv1.Project{
		NewObjectMeta: apiv1.NewObjectMeta{
			ID:                kubermaticProject.Name,
			Name:              kubermaticProject.Spec.Name,
			CreationTimestamp: kubermaticProject.CreationTimestamp.Time,
			DeletionTimestamp: func() *time.Time {
				if kubermaticProject.DeletionTimestamp != nil {
					return &kubermaticProject.DeletionTimestamp.Time
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
	ProjectReq
}

func decodeGetProject(c context.Context, r *http.Request) (interface{}, error) {
	projectReq, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	return GetProjectRq{projectReq.(ProjectReq)}, nil
}

// UpdateProjectRq defines HTTP request for updateProject
// swagger:parameters updateProject
type UpdateProjectRq struct {
	ProjectReq
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	var rq UpdateProjectRq
	return rq, errors.NewNotImplemented()
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
	ProjectReq
}

func decodeDeleteProject(c context.Context, r *http.Request) (interface{}, error) {
	req, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, nil
	}
	return DeleteProjectRq{ProjectReq: req.(ProjectReq)}, err
}

// kubernetesErrorToHTTPError constructs HTTPError only if the given err is of type *StatusError.
// Otherwise unmodified err will be returned to the caller.
func kubernetesErrorToHTTPError(err error) error {
	if kubernetesError, ok := err.(*kerrors.StatusError); ok {
		httpCode := kubernetesError.Status().Code
		httpMessage := kubernetesError.Status().Message
		return errors.New(int(httpCode), httpMessage)
	}
	return err
}
