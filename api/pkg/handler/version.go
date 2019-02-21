package handler

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

// getKubermaticVersion returns the versions of running Kubermatic components.
//
// At this time we're only interested in the version of the API server,
// since it knows its own version and can answer the endpoint promptly.
//
// This version string is constructed with `git describe` and embedded in the binary during build.
func getKubermaticVersion() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		versions := apiv1.KubermaticVersions{API: resources.KUBERMATICGITTAG}
		return versions, nil
	}
}
