package handler

import (
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/gorilla/context"
	"github.com/kubermatic/api/pkg/provider"
)

type userReq struct {
	user provider.User
}

func decodeUserReq(r *http.Request) (interface{}, error) {
	req := userReq{
		user: provider.User{
			Roles: map[string]struct{}{},
		},
	}

	obj := context.Get(r, "user")
	token := obj.(*jwt.Token)

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("Invalid JWT in request %v", r)
	}

	req.user.Name, ok = claims["sub"].(string)
	if req.user.Name == "" || !ok {
		// this is security critical. Better stop here hard. If auth0
		// works, this should never happen though.
		return nil, fmt.Errorf("No user in JWT in request %v", r)
	}

	md, ok := claims["app_metadata"].(map[string]interface{})
	if ok && md != nil {
		roles, ok := md["roles"].([]interface{})
		if ok && roles != nil {
			for _, r := range roles {
				s, ok := r.(string)
				if ok && s != "" {
					req.user.Roles[s] = struct{}{}
				}
			}
		}
	}

	glog.V(6).Infof("Request for user %q, roles %v", req.user.Name, req.user.Roles)
	glog.V(7).Infof("Request for user %q, token %+v", req.user.Name, token)

	overrideUser := r.URL.Query().Get("user")
	if overrideUser != "" {
		if _, isAdmin := req.user.Roles["admin"]; !isAdmin {
			return nil, NewNotAuthorized()
		}
		glog.V(4).Infof("Switching user to %q", overrideUser)
		req.user.Name = overrideUser
	}

	return req, nil
}
