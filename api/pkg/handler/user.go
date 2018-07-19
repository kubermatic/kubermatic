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
			cAPIUser := ctx.Value(apiUserContextKey)
			if cAPIUser == nil {
				return nil, errors.New("no user in context found")
			}
			apiUser := cAPIUser.(apiv1.User)

			user, err := r.userProvider.UserByEmail(apiUser.Email)
			if err != nil {
				if err == provider.ErrNotFound {
					user, err = r.userProvider.CreateUser(apiUser.ID, apiUser.Name, apiUser.Email)
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

// IsAdmin tells if the user has the admin role
func IsAdmin(u apiv1.User) bool {
	_, ok := u.Roles[AdminRoleKey]
	return ok
}
