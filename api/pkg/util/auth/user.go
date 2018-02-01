package auth

import (
	"context"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

// IsAdmin tells if the user has the admin role
func IsAdmin(u *apiv1.User) bool {
	_, ok := u.Roles[AdminRoleKey]
	return ok
}

// GetUser retrieves a user from a context
// If there was an error this function will panic.
func GetUser(ctx context.Context) apiv1.User {
	obj := ctx.Value(TokenUserContextKey)
	user, ok := obj.(apiv1.User)
	if !ok {
		panic("called with an invalid user in the context. Validate that the authentication Verifier ran before calling this function.")
	}
	return user
}
