package auth

import (
	"context"
)

// User represents an API user that is used for authentication.
type User struct {
	Name  string
	Roles map[string]struct{}
}

// GetUser retrieves a user from a context
// If there was an error this function will panic.
func GetUser(ctx context.Context) User {
	obj := ctx.Value(UserContextKey)
	user, ok := obj.(User)
	if !ok {
		panic("called with an invalid user in the context. Validate that the authentication Verifier ran before calling this function.")
	}
	return user
}
