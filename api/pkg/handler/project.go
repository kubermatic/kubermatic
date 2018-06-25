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
			ID:     kubermaticProject.Name,
			Name:   kubermaticProject.Spec.Name,
			Status: kubermaticProject.Status.Phase,
		}, nil
	}
}

func getProjectsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return []apiv1.Project{}, errors.NewNotImplemented()
	}
}

func deleteProjectEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		projectName, ok := request.(string)
		if !ok {
			return nil, errors.NewBadRequest("the name of the project to delete cannot be empty")
		}

		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		err := projectProvider.Delete(user, projectName)
		return nil, kubernetesErrorToHTTPError(err)
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

func decodeDeleteProject(c context.Context, r *http.Request) (interface{}, error) {
	return decodeProjectPathReq(c, r)
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
