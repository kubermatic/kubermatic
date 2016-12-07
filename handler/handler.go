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

	// Dirty but metalinter won't let us build.
	defer func() {
		err := rc.Close()
		_ = err
	}()

	_, err = io.Copy(w, rc)

	return err
}
