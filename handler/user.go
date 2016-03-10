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
	roles map[string]*struct{}
}

func decodeUserReq(r *http.Request) (interface{}, error) {
	req := userReq{
		roles: map[string]*struct{}{},
	}

	obj := context.Get(r, "user")
	token := obj.(*jwt.Token)
	var ok bool
	req.user, ok = token.Claims["sub"].(string)
	if req.user == "" || !ok {
		// this is security critical. Better stop here hard. If auth0
		// works, this should never happen though.
		return nil, fmt.Errorf("No user in JWT in request %v", r)
	}

	md, ok := token.Claims["app_metadata"].(map[string]interface{})
	if ok && md != nil {
		roles, ok := md["roles"].([]string)
		if ok && roles != nil {
			for _, r := range roles {
				req.roles[r] = &struct{}{}
			}
		}
	}

	glog.V(6).Infof("Request for user %q", req.user)
	glog.V(7).Infof("Request for user %q, token %+v", req.user, token)
	return req, nil
}
