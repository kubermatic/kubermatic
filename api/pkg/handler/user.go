package handler

import (
	"context"

	"github.com/kubermatic/kubermatic/api/pkg/handler/errors"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func GetUser(ctx context.Context) (provider.User, error) {
	obj := ctx.Value(UserContextKey)
	user, ok := obj.(provider.User)
	if !ok {
		return provider.User{}, errors.NewNotAuthorized()
	}

	return user, nil
}
