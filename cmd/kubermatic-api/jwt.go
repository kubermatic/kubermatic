package main

import (
	"encoding/base64"
	"net/http"

	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
)

func jwtMiddleware() *jwtmiddleware.JWTMiddleware {
	return jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			decoded, err := base64.URLEncoding.DecodeString("RE93Ef1Yt5-mrp2asikmfalfmcRaaa27gpH8hTAlby48LQQbUbn9d4F7yh01g_cc")
			if err != nil {
				return nil, err
			}
			return decoded, nil
		},
	})
}

func securedPingHandler(w http.ResponseWriter, r *http.Request) {
	msg := "All good. You only get this message if you're authenticated"
	_, _ = w.Write([]byte(msg))
}
