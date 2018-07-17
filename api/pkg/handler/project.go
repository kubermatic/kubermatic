package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

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

		projects := []*apiv1.Project{}
		for _, pg := range user.Spec.Projects {
			projectInternal, err := projectProvider.Get(user, pg.Name)
			if err != nil {
				return nil, kubernetesErrorToHTTPError(err)
			}
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
		if len(req.ProjectName) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		err := projectProvider.Delete(user, req.ProjectName)
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
		req, ok := request.(GetProjectRq)

		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if len(req.Name) == 0 {
			return nil, errors.NewBadRequest("the name of the project cannot be empty")
		}

		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		kubermaticProject, err := projectProvider.Get(user, req.Name)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		return convertInternalProjectToExternal(kubermaticProject), nil
	}
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
// swagger:parameters getProject
type GetProjectRq struct {
	// in: path
	Name string `json:"project_id"`
}

func decodeGetProject(c context.Context, r *http.Request) (interface{}, error) {
	var req GetProjectRq

	projectID, ok := mux.Vars(r)["project_id"]
	if !ok {
		return nil, fmt.Errorf("'node_name' parameter is required but was not provided")
	}

	req.Name = projectID
	return req, nil
}

// UpdateProjectRq defines HTTP request for updateProject endpoint
// swagger:parameters updateProject
type UpdateProjectRq struct {
	// in: path
	Name string `json:"project_id"`
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	var req UpdateProjectRq
	return req, errors.NewNotImplemented()
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
	// in: path
	ProjectName string `json:"project_id"`
}

func decodeDeleteProject(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteProjectRq
	var err error
	req.ProjectName, err = decodeProjectPathReq(c, r)
	return req, err
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
