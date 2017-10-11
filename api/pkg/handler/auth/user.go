package auth

import (
	"context"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// GetUser retrieves a user from a context
// If there was an error this function will panic.
func GetUser(ctx context.Context) provider.User {
	obj := ctx.Value(UserContextKey)
	user, ok := obj.(provider.User)
	if !ok {
		panic("User isn't authenticated!")
	}
	return user
}
