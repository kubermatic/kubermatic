package handler

import (
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/gorilla/context"
)

type userReq struct {
	user string
}

func decodeUserReq(r *http.Request) (interface{}, error) {
	var req userReq
	obj := context.Get(r, "user")
	token := obj.(*jwt.Token)
	req.user = token.Claims["sub"].(string)
	if req.user == "" {
		// this is security critical. Better stop here hard. If auth0
		// works, this should never happen though.
		return nil, fmt.Errorf("No user in JWT in request %v", r)
	}
	glog.V(6).Infof("Request for user %q", req.user)
	return req, nil
}
