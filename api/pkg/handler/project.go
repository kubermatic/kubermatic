package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
)

// Project is a top-level container for a set of resources
// swagger:model Project
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func decodeProjectPathReq(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func getProjectsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return []Project{}, nil
	}
}

func deleteProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Don't return project just success.
		return nil, errors.New("not implemented")
	}
}

func decodeUpdateProject(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func updateProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Project{}, nil
	}
}

func decodeCreateProject(c context.Context, r *http.Request) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func createProjectEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return Project{}, nil
	}
}
