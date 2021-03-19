/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package serviceaccount_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateTokenProject(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
		expectedErrorResponse  string
		expectedName           string
		saToSync               string
		httpStatus             int
		existingAPIUser        apiv1.User
	}{
		{
			name:       "scenario 1: create service account token with name 'test' for serviceaccount-1",
			body:       `{"name":"test"}`,
			httpStatus: http.StatusCreated,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenProjectServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenProjectServiceAccount("3", "test-3", "viewers", "plan9-ID"),
				test.GenMainServiceAccount("4", "test-4", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("5", "test-5", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("6", "test-5", "viewers", "bob@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			saToSync:               "4",
			expectedName:           "test",
		},
		{
			name:       "scenario 2: create service account token with existing name",
			body:       `{"name":"test"}`,
			httpStatus: http.StatusConflict,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenProjectServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenProjectServiceAccount("3", "test-3", "viewers", "plan9-ID"),
				test.GenMainServiceAccount("4", "test-4", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("5", "test-5", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("6", "test-5", "viewers", "bob@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("", "4", "test", "1"),
			},
			existingAPIUser:       *test.GenAPIUser("john", "john@acme.com"),
			saToSync:              "4",
			expectedErrorResponse: `{"error":{"code":409,"message":"token \"test\" already exists"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/serviceaccounts/%s/tokens", tc.saToSync), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			ep, fakeClients, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if len(tc.expectedErrorResponse) > 0 {
				test.CompareWithResult(t, res, tc.expectedErrorResponse)
			} else {
				var saToken apiv1.ServiceAccountToken
				err = json.Unmarshal(res.Body.Bytes(), &saToken)
				if err != nil {
					t.Fatal(err)
				}

				if tc.expectedName != saToken.Name {
					t.Fatalf("expected token name %s got %s", tc.expectedName, saToken.Name)
				}

				_, saTokenClaim, err := fakeClients.TokenAuthenticator.Authenticate(saToken.Token)
				if err != nil {
					t.Fatal(err)
				}
				if saTokenClaim.TokenID != saToken.ID {
					t.Fatalf("expected ID %s got %s", saToken.ID, saTokenClaim.TokenID)
				}
				if saTokenClaim.Email != fmt.Sprintf("main-serviceaccount-%s@sa.kubermatic.io", tc.saToSync) {
					t.Fatalf("expected email main-serviceaccount-%s@sa.kubermatic.io got %s", tc.saToSync, saTokenClaim.Email)
				}
			}
		})
	}
}
