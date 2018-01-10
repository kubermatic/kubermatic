package handler

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
)

type contextKey int

const (
	// UserContextKey defines the context key to find the kubermatic-user
	UserContextKey contextKey = 1
)

func (r Routing) userSaverMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			cUser := ctx.Value(auth.TokenUserContextKey)
			if cUser == nil {
				return nil, errors.New("no user in context found")
			}
			user, err := r.provider.UserByEmail(cUser.(auth.User).Email)
			if err != nil {
				if err == provider.ErrNotFound {
					user, err = r.provider.CreateUser(cUser.(auth.User).ID, cUser.(auth.User).Name, cUser.(auth.User).Email)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			}

			return next(context.WithValue(ctx, UserContextKey, user), request)
		}
	}
}

func getUserHandler() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		return user, nil
	}
}
