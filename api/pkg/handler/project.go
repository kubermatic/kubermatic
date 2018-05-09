package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
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
			if err == kubernetes.ErrProjectAlreadyExist {
				return nil, errors.NewAlreadyExists("Project", projectRq.Name)
			}
			return nil, err
		}

		return apiv1.Project{
			ID:   string(kubermaticProject.UID),
			Name: kubermaticProject.Spec.Name,
			// TODO (p0lyn0mial): add/set the status
		}, nil
	}
}

func getProjectsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return []apiv1.Project{}, nil
	}
}

func deleteProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, errors.NewNotImplemented()
	}
}

func updateProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return apiv1.Project{}, nil
	}
}

type projectReq struct {
	Name string `json:"name"`
}

func decodeProjectPathReq(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.NewNotImplemented()
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
