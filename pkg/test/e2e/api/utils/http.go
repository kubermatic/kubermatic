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

const (
	apiRequestTimeout = 10 * time.Second
)

type requestParameterHolder interface {
	SetHTTPClient(*http.Client)
}

func SetupParams(t *testing.T, p requestParameterHolder, interval time.Duration, timeout time.Duration, ignoredStatusCodes ...int) {
	p.SetHTTPClient(NewHTTPClientWithRetries(t, apiRequestTimeout, interval, timeout, ignoredStatusCodes...))
}

func NewHTTPClientWithRetries(t *testing.T, requestTimeout time.Duration, retryInterval, retryTimeout time.Duration, ignoredStatusCodes ...int) *http.Client {
	return &http.Client{
		Transport: &relaxedRoundtripper{
			test:               t,
			ignoredStatusCodes: sets.NewInt(ignoredStatusCodes...),
			retryInterval:      retryInterval,
			retryTimeout:       retryTimeout,
			requestTimeout:     requestTimeout,
		},
	}
}

type relaxedRoundtripper struct {
	test               *testing.T
	ignoredStatusCodes sets.Int
	retryInterval      time.Duration
	retryTimeout       time.Duration
	requestTimeout     time.Duration
}

func (r *relaxedRoundtripper) RoundTrip(request *http.Request) (*http.Response, error) {
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

	attempts := 0

	// TODO(irozzo): Probably using an exponential backoff instead is more
	// appropriate and avoids too many calls.
	err := wait.PollImmediate(r.retryInterval, r.retryTimeout, func() (bool, error) {
		var reqErr error

		attempts++

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
			multiErr = multierror.Append(multiErr, errors.Wrap(reqErr, "error ocurred during http call"))
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
		// because our Poll function never returns an error, err must be ErrWaitTimeout;
		// RoundTrippers must never return a response and an error at the same time.
		return nil, errors.Wrapf(multiErr, "request did not succeed after %v (%d attempts, ignoring HTTP codes %v)", r.retryTimeout, attempts, r.ignoredStatusCodes.List())
	}

	return response, nil
}

func (r *relaxedRoundtripper) isTransientError(resp *http.Response) bool {
	// 5xx return codes may be associated to recoverable
	// conditions, with the exception of 501 (Not implemented)
	return resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented) || r.ignoredStatusCodes.Has(resp.StatusCode)
}
