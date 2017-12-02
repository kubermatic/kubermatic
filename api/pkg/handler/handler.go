package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

const (
	headerContentType = "Content-Type"

	contentTypeJSON = "application/json"
)

// APIError we need to work with github.com/yvasiyarov/swagger
// based on https://github.com/yvasiyarov/swagger/blob/master/example/data_structures.go
// swagger:response APIError
type APIError struct {
	ErrorCode    int    `json:"code"`
	ErrorMessage string `json:"message"`
}

func errorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	errorCode := http.StatusInternalServerError
	msg := err.Error()
	if h, ok := err.(errors.HTTPError); ok {
		errorCode = h.StatusCode()
		msg = h.Error()
	}
	e := struct {
		Error APIError `json:"error"`
	}{
		Error: APIError{
			ErrorCode:    errorCode,
			ErrorMessage: msg,
		},
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(errorCode)
	err = encodeJSON(ctx, w, e)
	if err != nil {
		glog.Info(err)
	}
}

// StatusOK returns a handler always returning http status code 200 (StatusOK).
// swagger:route GET /api/healthz status health StatusOK
//
// Return a HTTP 200 if everythings fine
//
//     Schemes: http, https
//
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
	w.Header().Set(headerContentType, contentTypeJSON)
	return json.NewEncoder(w).Encode(response)
}
