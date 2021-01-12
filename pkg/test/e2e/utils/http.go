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

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Backoff wait.Backoff

const (
	apiRequestTimeout = 10 * time.Second
)

type requestParameterHolder interface {
	SetHTTPClient(*http.Client)
}

// SetupParams configures retries for HTTP calls for a total period defined by
// 'timeout' parameter and for an 'interval' duration.
// Deprecated: Use SetupRetryParams instead.
func SetupParams(t *testing.T, p requestParameterHolder, interval time.Duration, timeout time.Duration, ignoredStatusCodes ...int) {
	// set backoff factor to 1 to fallback to linear backoff
	SetupRetryParams(t, p, Backoff{Duration: interval, Steps: int(timeout / interval), Factor: 1},
		ignoredStatusCodes...)
}

// SetupRetryParams configure retries for HTTP calls based on backoff
// parameters.
func SetupRetryParams(t *testing.T, p requestParameterHolder, backoff Backoff, ignoredStatusCodes ...int) {
	p.SetHTTPClient(&http.Client{
		Transport: NewRoundTripperWithRetries(t, apiRequestTimeout, backoff, ignoredStatusCodes...),
	})
}

func NewRoundTripperWithRetries(t *testing.T, requestTimeout time.Duration, backoff Backoff, ignoredStatusCodes ...int) http.RoundTripper {
	return &retryRoundTripper{
		Backoff:            backoff,
		test:               t,
		ignoredStatusCodes: sets.NewInt(ignoredStatusCodes...),
		requestTimeout:     requestTimeout,
	}
}

type retryRoundTripper struct {
	Backoff
	requestTimeout     time.Duration
	test               *testing.T
	ignoredStatusCodes sets.Int
}

func (r *retryRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var (
		bodyBytes []byte
		response  *http.Response
		multiErr  error
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

	// do at least an attempt
	if r.Backoff.Steps <= 0 {
		r.Backoff.Steps = 1
	}
	err := wait.ExponentialBackoff(wait.Backoff(r.Backoff), func() (bool, error) {
		var reqErr error

		// create a fresh timeout that starts *now*
		// NB: Do *not* cancel this context, as the context controls the
		// request and response lifecycle. Cancelling the context here will
		// make it impossible for the caller to read the response body.
		// As this context times out anyway, and timing out means it closes
		// itself, it's okay to not call cancel() here.
		//nolint:lostcancel
		ctx, _ := context.WithTimeout(context.Background(), r.requestTimeout)

		// replace any preexisting context with our new one
		requestClone := request.WithContext(ctx)

		if bodyBytes != nil {
			requestClone.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		}

		// perform request
		//nolint:bodyclose
		response, reqErr = http.DefaultTransport.RoundTrip(requestClone)
		if reqErr != nil {
			multiErr = multierror.Append(multiErr, errors.Wrap(reqErr, "error occurred while executing http call"))
			return false, nil
		}

		// ignore transient errors
		if r.isTransientError(response) {
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "HTTP %s", response.Status))
			} else {
				multiErr = multierror.Append(multiErr, fmt.Errorf("HTTP %s: %s", response.Status, string(body)))
			}

			response.Body.Close()
			return false, nil
		}

		// success!
		return true, nil
	})

	if err != nil {
		return nil, errors.Wrapf(multiErr, "request did not succeed after %d attempts (ignoring HTTP codes %v)", r.Steps, r.ignoredStatusCodes.List())
	}

	return response, nil
}

func (r *retryRoundTripper) isTransientError(resp *http.Response) bool {
	// 5xx return codes may be associated to recoverable
	// conditions, with the exception of 501 (Not implemented)
	return resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented) || r.ignoredStatusCodes.Has(resp.StatusCode)
}
