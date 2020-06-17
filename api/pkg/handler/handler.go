/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

const (
	headerContentType = "Content-Type"

	contentTypeJSON = "application/json"
)

// ErrorResponse is the default representation of an error
// swagger:model errorResponse
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
	Code int `json:"code"`
	// The error message
	//
	// Required: true
	Message string `json:"message"`
	// Additional error messages
	//
	// Required: false
	Additional []string `json:"details,omitempty"`
}

// EmptyResponse is a empty response
// swagger:response empty
type EmptyResponse struct{}

func decodeEmptyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req struct{}
	return req, nil
}

func errorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	var additional []string
	errorCode := http.StatusInternalServerError
	msg := err.Error()
	if h, ok := err.(errors.HTTPError); ok {
		errorCode = h.StatusCode()
		msg = h.Error()
		additional = h.Details()
	}
	e := ErrorResponse{
		Error: ErrorDetails{
			Code:       errorCode,
			Message:    msg,
			Additional: additional,
		},
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(errorCode)
	err = encodeJSON(ctx, w, e)
	if err != nil {
		log.Logger.Error(err)
	}
}

// encodeJSON writes the JSON encoding of response to the http response writer
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

// statusOK returns the status code 200
func statusOK(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func setStatusCreatedHeader(f func(context.Context, http.ResponseWriter, interface{}) error) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, r http.ResponseWriter, i interface{}) error {
		r.Header().Set(headerContentType, contentTypeJSON)
		r.WriteHeader(http.StatusCreated)
		return f(ctx, r, i)
	}
}
