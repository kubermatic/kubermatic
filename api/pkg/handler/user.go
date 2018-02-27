package handler

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func (r Routing) userSaverMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			apiUser := ctx.Value(apiUserContextKey)
			if apiUser == nil {
				return nil, errors.New("no user in context found")
			}
			user, err := r.userProvider.UserByEmail(apiUser.(apiv1.User).Email)
			if err != nil {
				if err == provider.ErrNotFound {
					user, err = r.userProvider.CreateUser(apiUser.(apiv1.User).ID, apiUser.(apiv1.User).Name, apiUser.(apiv1.User).Email)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			}

			return next(context.WithValue(ctx, userCRContextKey, user), request)
		}
	}
}

func getUserHandler() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		apiUser := ctx.Value(apiUserContextKey)
		return apiUser, nil
	}
}

// IsAdmin tells if the user has the admin role
func IsAdmin(u apiv1.User) bool {
	_, ok := u.Roles[AdminRoleKey]
	return ok
}
