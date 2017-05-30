package handler

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// BasicAuth is a middleware for simple basic auth authentication. If authUser & authPass are empty, every request will go trough
func BasicAuth(authUser string, authPass string, next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		user, pass, ok := r.BasicAuth()
		authDisabled := authUser == "" && authPass == ""
		//We only want to check the authentication if we got started with --auth-user or --auth-password
		//If both are empty, we allow all
		if (ok && user == authUser && pass == authPass) || authDisabled {
			next(w, r, p)
			return
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return

	}
}
