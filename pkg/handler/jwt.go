package handler

import (
	"encoding/base64"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

func jwtMiddleware(key string) *jwtmiddleware.JWTMiddleware {
	return jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			decoded, err := base64.URLEncoding.DecodeString(key)
			return decoded, err
		},
		Debug: bool(glog.V(6)),
	})
}

func jwtGetMiddleware(key string) *jwtmiddleware.JWTMiddleware {
	mw := jwtMiddleware(key)
	mw.Options.Extractor = jwtmiddleware.FromParameter("token")
	return mw
}
