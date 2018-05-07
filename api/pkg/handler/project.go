package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

func decodeProjectPathReq(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func getProjectsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return []apiv1.Project{}, nil
	}
}

func deleteProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, errors.New("not implemented")
	}
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func updateProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return apiv1.Project{}, nil
	}
}

func decodeCreateProject(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func createProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return apiv1.Project{}, nil
	}
}
