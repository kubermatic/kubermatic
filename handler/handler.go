package handler

import (
	"encoding/json"
	"golang.org/x/net/context"
	"net/http"
)

// StatusOK returns a handler always returning http status code 200 (StatusOK).
func StatusOK(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func encodeJSON(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	return json.NewEncoder(w).Encode(response)
}
