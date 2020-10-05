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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	apiRequestTimeout = 10 * time.Second
)

type requestParameterHolder interface {
	SetHTTPClient(*http.Client)
}

func SetupParams(t *testing.T, p requestParameterHolder, interval time.Duration, timeout time.Duration, ignoredStatusCodes ...int) {
	p.SetHTTPClient(newHTTPClient(t, interval, timeout, ignoredStatusCodes...))
}

func newHTTPClient(t *testing.T, interval time.Duration, timeout time.Duration, ignoredStatusCodes ...int) *http.Client {
	return &http.Client{
		Transport: &relaxedRoundtripper{
			test:               t,
			ignoredStatusCodes: ignoredStatusCodes,
			interval:           interval,
			timeout:            timeout,
		},
	}
}

type relaxedRoundtripper struct {
	test               *testing.T
	ignoredStatusCodes []int
	interval           time.Duration
	timeout            time.Duration
}

func (r *relaxedRoundtripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var (
		bodyBytes []byte
		response  *http.Response
		lastErr   error
	)

	// clone request body
	if request.Body != nil {
		var err error

		bodyBytes, err = ioutil.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %v", err)
		}
	}

	if r.test != nil {
		r.test.Logf("%s %s", request.Method, request.URL.Path)
	}

	attempts := 0
	err := wait.PollImmediate(r.interval, r.timeout, func() (bool, error) {
		var reqErr error

		attempts++

		// create a fresh timeout that starts *now*
		// NB: Do *not* cancel this context, as the context controls the
		// request and response lifecycle. Cancelling the context here will
		// make it impossible for the caller to read the response body.
		// As this context times out anyway, and timing out means it closes
		// itself, it's okay to not call cancel() here.
		//nolint:lostcancel
		ctx, _ := context.WithTimeout(context.Background(), apiRequestTimeout)

		// replace any preexisting context with our new one
		requestClone := request.Clone(ctx)

		if bodyBytes != nil {
			requestClone.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		}

		// perform request
		//nolint:bodyclose
		response, reqErr = http.DefaultTransport.RoundTrip(requestClone)

		// swallow network errors like timeouts
		if reqErr != nil {
			// only record the request error if it was not the ctx being cancelled or
			// expiring, as the lastErr is meant to give the cause for the timeout
			if reqErr != context.DeadlineExceeded && reqErr != context.Canceled {
				lastErr = reqErr
			}

			return false, nil
		}

		// ignore transient errors
		if r.isTransientError(response) {
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				lastErr = fmt.Errorf("HTTP %s: (failed to read body: %v)", response.Status, err)
			} else {
				lastErr = fmt.Errorf("HTTP %s: %s", response.Status, string(body))
			}

			response.Body.Close()
			return false, nil
		}

		// success!
		return true, nil
	})

	if err != nil {
		// because our Poll function never returns an error, err must be ErrWaitTimeout;
		// RoundTrippers must never return a response and an error at the same time.
		return nil, fmt.Errorf("request did not succeed after %v (%d attempts, ignoring HTTP codes %v), last error was: %v", r.timeout, attempts, r.ignoredStatusCodes, lastErr)
	}

	return response, nil
}

func (r *relaxedRoundtripper) isTransientError(response *http.Response) bool {
	if response.StatusCode >= http.StatusInternalServerError {
		return true
	}

	for _, code := range r.ignoredStatusCodes {
		if code == response.StatusCode {
			return true
		}
	}

	return false
}
