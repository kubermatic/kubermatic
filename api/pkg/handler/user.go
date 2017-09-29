package handler

import (
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type userReq struct {
	user provider.User
}

func decodeUserReq(r *http.Request) (interface{}, error) {
	obj := r.Context().Value(UserContextKey)
	user := obj.(provider.User)
	req := userReq{
		user: user,
	}

	return req, nil
}
