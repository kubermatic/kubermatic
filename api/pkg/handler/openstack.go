package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
)

func openstackSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, fmt.Errorf("not yet implemented! request = %#v", request)
	}
}
