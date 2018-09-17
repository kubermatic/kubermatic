package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	goerrors "errors"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

const (
	headerContentType = "Content-Type"

	contentTypeJSON = "application/json"
)

// ErrorResponse is the default representation of an error
// swagger:response errorResponse
type ErrorResponse struct {
	// The error details
	// in: body
	Error ErrorDetails `json:"error"`
}

// ErrorDetails contains details about the error
type ErrorDetails struct {
	// The error code
	//
	// Required: true
	ErrorCode int `json:"code"`
	// The error message
	//
	// Required: true
	ErrorMessage string `json:"message"`
}

// RawResponse is the default representation of a raw (proxied) HTTP response
type RawResponse struct {
	Header http.Header
	Body   []byte
}

func errorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	errorCode := http.StatusInternalServerError
	msg := err.Error()
	if h, ok := err.(errors.HTTPError); ok {
		errorCode = h.StatusCode()
		msg = h.Error()
	}
	e := ErrorResponse{
		Error: ErrorDetails{
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

// EmptyResponse is a empty response
// swagger:response empty
type EmptyResponse struct{}

// StatusOK returns the status code 200
func StatusOK(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func setStatusCreatedHeader(f func(context.Context, http.ResponseWriter, interface{}) error) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, r http.ResponseWriter, i interface{}) error {
		r.WriteHeader(http.StatusCreated)
		return f(ctx, r, i)
	}
}

func encodeJSON(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	w.Header().Set(headerContentType, contentTypeJSON)

	// As long as we pipe the response from the listers we need this.
	// The listers might return a uninitialized slice in case it has no results.
	// This results to "null" when marshaling to json.
	t := reflect.TypeOf(response)
	if t != nil && t.Kind() == reflect.Slice {
		v := reflect.ValueOf(response)
		if v.Len() == 0 {
			_, err := w.Write([]byte("[]"))
			return err
		}
	}

	// For completely empty responses, we still want to ensure that we
	// send a JSON object instead of the string "null".
	if response == nil {
		_, err := w.Write([]byte("{}"))
		return err
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeRawResponse(c context.Context, w http.ResponseWriter, response interface{}) error {
	if resp, ok := response.(RawResponse); ok {
		for field, values := range resp.Header {
			for _, value := range values {
				w.Header().Set(field, value)
			}
		}
		_, err := w.Write(resp.Body)
		return err
	}
	return goerrors.New("internal error (unexpected raw response object)")
}
