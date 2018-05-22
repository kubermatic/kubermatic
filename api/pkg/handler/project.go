package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
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
			return nil, errors.KubernetesErrorToHTTPError(err)
		}

		return apiv1.Project{
			ID:     kubermaticProject.Name,
			Name:   kubermaticProject.Spec.Name,
			Status: kubermaticProject.Status.Phase,
		}, nil
	}
}

func getProjectsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return []apiv1.Project{}, nil
	}
}

func deleteProjectEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		projectName, ok := request.(string)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		if len(projectName) == 0 {
			return nil, errors.NewBadRequest("the name of the project to delete cannot be empty")
		}

		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		err := projectProvider.Delete(user, projectName)
		return nil, errors.KubernetesErrorToHTTPError(err)
	}
}

func updateProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return apiv1.Project{}, errors.NewNotImplemented()
	}
}

type projectReq struct {
	Name string `json:"name"`
}

func decodeProjectPathReq(c context.Context, r *http.Request) (interface{}, error) {
	// project_id is actually an internal name of the object
	projectName, ok := mux.Vars(r)["project_id"]
	if !ok {
		return nil, fmt.Errorf("'project_id' parameter is required in order to delete the project")
	}
	return projectName, nil
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.NewNotImplemented()
}

func decodeCreateProject(c context.Context, r *http.Request) (interface{}, error) {
	var req projectReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}
