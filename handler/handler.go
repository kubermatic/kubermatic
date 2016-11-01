package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// StatusOK returns a handler always returning http status code 200 (StatusOK).
func StatusOK(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func encodeJSON(w http.ResponseWriter, response interface{}) (err error) {
	return json.NewEncoder(w).Encode(response)
}

func encodeText(w http.ResponseWriter, response interface{}) (err error) {
	rc, ok := response.(io.ReadCloser)
	if !ok {
		return errors.New("response does not implement io.ReadCloser")
	}
	defer rc.Close()

	_, err = io.Copy(w, rc)

	return err
}
