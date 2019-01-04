package middleware

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// contextKey defines a dedicated type for keys to use on contexts
type contextKey string

const (
	datacenterContextKey contextKey = "datacenter"
	// ClusterProviderContextKey key under which the current ClusterProvider is kept in the ctx
	ClusterProviderContextKey contextKey = "cluster-provider"
	// UserInfoContextKey key under which the current UserInfo is kept in the ctx
	UserInfoContextKey contextKey = "user-info"
	// UserCRContextKey key under which the current User (from the database) is kept in the ctx
	UserCRContextKey contextKey = "user-cr"
	// AuthenticatedUserContextKey key under which the current User (from OIDC provider) is kept in the ctx
	AuthenticatedUserContextKey contextKey = "authenticated-user"
)

//DCGetter defines functionality to retrieve a datacenter name
type dCGetter interface {
	GetDC() string
}

// Datacenter is a middleware that injects the current ClusterProvider into the ctx
func Datacenter(clusterProviders map[string]provider.ClusterProvider, datacenters map[string]provider.DatacenterMeta) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			getter := request.(dCGetter)
			dc, exists := datacenters[getter.GetDC()]
			if !exists {
				return nil, errors.NewNotFound("datacenter", getter.GetDC())
			}
			ctx = context.WithValue(ctx, datacenterContextKey, dc)

			clusterProvider, exists := clusterProviders[getter.GetDC()]
			if !exists {
				return nil, errors.NewNotFound("cluster-provider", getter.GetDC())
			}
			ctx = context.WithValue(ctx, ClusterProviderContextKey, clusterProvider)
			return next(ctx, request)
		}
	}
}
