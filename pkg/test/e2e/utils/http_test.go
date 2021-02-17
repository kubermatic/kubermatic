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

package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHttpClientWithRetries(t *testing.T) {
	var testcases = []struct {
		name              string
		handlerFuncs      []http.HandlerFunc
		retryInterval     time.Duration
		numRetries        int
		requestTimeout    time.Duration
		allowedErrorCodes []int
		expStatus         int
		expErr            bool
	}{
		{
			name: "success at first attempt",
			handlerFuncs: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					fmt.Fprintln(w, "success")
				},
			},

			numRetries: 1,
			expStatus:  200,
		},
		{
			name: "success after 2 allowed error codes",
			handlerFuncs: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(404)
					fmt.Fprintln(w, "failed")
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(404)
					fmt.Fprintln(w, "failed")
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					fmt.Fprintln(w, "success")
				},
			},
			retryInterval:     1 * time.Millisecond,
			numRetries:        3,
			allowedErrorCodes: []int{404},
			expStatus:         200,
		},
		{
			name: "success after 5xx",
			handlerFuncs: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(500)
					fmt.Fprintln(w, "temporary server error")
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					fmt.Fprintln(w, "failed")
				},
			},
			retryInterval: 1 * time.Millisecond,
			numRetries:    2,
			expStatus:     200,
		},
		{
			name: "do not retry after 501",
			handlerFuncs: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(501)
					fmt.Fprintln(w, "not implemented")
				},
			},
			expStatus: 501,
		},
		{
			name: "Error after retry timeout",
			handlerFuncs: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(500)
					fmt.Fprintln(w, "temporary server error")
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(503)
					fmt.Fprintln(w, "temporary server error")
				},
			},
			retryInterval: 1 * time.Millisecond,
			numRetries:    3,
			expErr:        true,
		},
		{
			name: "Success after initial request timeout",
			handlerFuncs: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					time.AfterFunc(20*time.Millisecond, func() {
						w.WriteHeader(200)
						fmt.Fprintln(w, "success")
					})
				},
				func(w http.ResponseWriter, r *http.Request) {
					time.AfterFunc(20*time.Millisecond, func() {
						w.WriteHeader(200)
						fmt.Fprintln(w, "success")
					})
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					fmt.Fprintln(w, "success")
				},
			},
			retryInterval:  1 * time.Millisecond,
			numRetries:     3,
			requestTimeout: 10 * time.Millisecond,
			expStatus:      200,
		},
		{
			name: "Failed due to request timeout",
			handlerFuncs: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					time.AfterFunc(20*time.Millisecond, func() {
						w.WriteHeader(200)
						fmt.Fprintln(w, "success")
					})
				},
			},
			retryInterval:  1 * time.Millisecond,
			numRetries:     3,
			requestTimeout: 10 * time.Microsecond,
			expErr:         true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(iterateHandlerFuncs(tt.handlerFuncs...))
			if tt.requestTimeout == 0 {
				tt.requestTimeout = apiRequestTimeout
			}
			rt := NewRoundTripperWithRetries(t, tt.requestTimeout, Backoff{Steps: tt.numRetries, Duration: tt.retryInterval, Factor: 1.0}, tt.allowedErrorCodes...)
			cli := &http.Client{Transport: rt}
			req, err := http.NewRequest("GET", ts.URL, nil)
			if err != nil {
				t.Fatalf("Error occurred while creating request: %v", err)
			}
			res, err := cli.Do(req)
			if err != nil {
				if !tt.expErr {
					t.Fatalf("Expected request success but get: %v", err)
				}
				return
			}
			defer res.Body.Close()
			if tt.expErr {
				t.Fatalf("Expected error but none get")
			}
			if res.StatusCode != tt.expStatus {
				t.Errorf("Expected status %d but got %d", tt.expStatus, res.StatusCode)
			}
		})
	}
}

func iterateHandlerFuncs(funcs ...http.HandlerFunc) http.HandlerFunc {
	// use a buffered channel for thread safety
	handlersChan := make(chan http.HandlerFunc, len(funcs))
	// fill the buffered channel with the given functions, this won't block as
	// the size of the buffered channel corresponds to the number of
	// functions
	for _, f := range funcs {
		handlersChan <- f
	}
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case h := <-handlersChan:
			h(w, r)
		default:
			// once exhausted the channel just run the latest func
			funcs[len(funcs)-1](w, r)
		}
	}
}
