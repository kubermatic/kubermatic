package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"golang.org/x/net/context"
)

// StatusOK returns a handler always returning http status code 200 (StatusOK).
func StatusOK(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func createStatusResource(f func(context.Context, http.ResponseWriter, interface{}) error) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, r http.ResponseWriter, i interface{}) error {
		err := f(ctx, r, i)
		if err != nil {
			return err
		}
		r.WriteHeader(http.StatusCreated)
		return nil
	}
}

func encodeJSON(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	w.Header().Set("Content-Type", "application/json")
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
