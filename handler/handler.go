package handler

import (
	"encoding/json"
	"net/http"
)

// StatusOK returns a handler always returning http status code 200 (StatusOK).
func StatusOK(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func encodeJSON(w http.ResponseWriter, response interface{}) (err error) {
	return json.NewEncoder(w).Encode(response)
}
