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

package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetAdmins(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: unauthorized user gets admin list",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user gets admin list",
			expectedResponse:       `[{"email":"bob@acme.com","name":"Bob","isAdmin":true}]`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("GET", "/api/v1/admin", strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj := []ctrlruntimeclient.Object{test.GenTestSeed()}
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestSetAdmin(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		body                   string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: unauthorized user tries set admin",
			body:                   `{"email":"john@acme.com","isAdmin":true}`,
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user adds admin rights",
			body:                   `{"email":"john@acme.com","isAdmin":true}`,
			expectedResponse:       `{"email":"john@acme.com","name":"John","isAdmin":true}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), genUser("John", "john@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                   "scenario 3: authorized user adds admin for non existing user",
			body:                   `{"email":"patric@acme.com","isAdmin":true}`,
			expectedResponse:       `{"error":{"code":500,"message":"the given user patric@acme.com was not found"}}`,
			httpStatus:             http.StatusInternalServerError,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), genUser("John", "john@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 4
		{
			name:                   "scenario 4: authorized user wants change own role",
			body:                   `{"email":"bob@acme.com","isAdmin":true}`,
			expectedResponse:       `{"error":{"code":400,"message":"can not change own privileges"}}`,
			httpStatus:             http.StatusBadRequest,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), genUser("John", "john@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("PUT", "/api/v1/admin", strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			kubermaticObj := []ctrlruntimeclient.Object{test.GenTestSeed()}
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}
