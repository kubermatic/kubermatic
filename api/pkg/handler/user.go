package handler

import (
	"context"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type userReq struct {
	user provider.User
}

func decodeUserReq(ctx context.Context) (interface{}, error) {
	obj := ctx.Value(UserContextKey)
	user := obj.(provider.User)
	req := userReq{
		user: user,
	}

	return req, nil
}
