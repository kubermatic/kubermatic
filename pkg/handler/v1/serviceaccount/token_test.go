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

package serviceaccount_test

import (
	"context"
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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		projectToSync          string
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
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:          "plan9-ID",
			saToSync:               "1",
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
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test", "1"),
			},
			existingAPIUser:       *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:         "plan9-ID",
			saToSync:              "1",
			expectedErrorResponse: `{"error":{"code":409,"message":"token \"test\" already exists"}}`,
		},
		{
			name:       "scenario 3: the admin can create token for any SA",
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
				genUser("bob", "bob@acme.com", true),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:          "plan9-ID",
			saToSync:               "1",
			expectedName:           "test",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens", tc.projectToSync, tc.saToSync), strings.NewReader(tc.body))
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
				if saTokenClaim.ProjectID != tc.projectToSync {
					t.Fatalf("expected project name %s got %s", tc.projectToSync, saTokenClaim.ProjectID)
				}
				if saTokenClaim.Email != fmt.Sprintf("serviceaccount-%s@sa.kubermatic.io", tc.saToSync) {
					t.Fatalf("expected email %s@sa.kubermatic.io got %s", tc.saToSync, saTokenClaim.Email)
				}
			}
		})
	}
}

func TestListTokens(t *testing.T) {
	t.Parallel()
	expiry, err := test.GenDefaultExpiry()
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		name                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
		expectedTokens         []apiv1.PublicServiceAccountToken
		projectToSync          string
		saToSync               string
		httpStatus             int
		existingAPIUser        apiv1.User
	}{
		{
			name:       "scenario 1: list tokens",
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan10-ID", "serviceaccount-2", "test-2", "2"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-3", "3"),
				test.GenDefaultSaToken("plan11-ID", "serviceaccount-3", "test-4", "4"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "1",
			expectedTokens: []apiv1.PublicServiceAccountToken{
				genPublicServiceAccountToken("1", "test-1", expiry),
				genPublicServiceAccountToken("3", "test-3", expiry),
			},
		},
		{
			name:       "scenario 2: the admin can list tokens for any service account",
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenServiceAccount("5", "test-2", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan10-ID", "serviceaccount-2", "test-2", "2"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-3", "3"),
				test.GenDefaultSaToken("plan11-ID", "serviceaccount-3", "test-4", "4"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-5", "test-5", "5"),
			},
			existingAPIUser: *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "1",
			expectedTokens: []apiv1.PublicServiceAccountToken{
				genPublicServiceAccountToken("1", "test-1", expiry),
				genPublicServiceAccountToken("3", "test-3", expiry),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens", tc.projectToSync, tc.saToSync), strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			actualSA := test.NewServiceAccountTokenV1SliceWrapper{}
			actualSA.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedToken := test.NewServiceAccountTokenV1SliceWrapper(tc.expectedTokens)
			wrappedExpectedToken.Sort()

			actualSA.EqualOrDie(wrappedExpectedToken, t)

		})
	}
}

func TestServiceAccountCanGetProject(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingSa             *kubermaticapiv1.User
		expectedResponse       string
		projectToSync          string
		httpStatus             int
		existingAPIUser        apiv1.User
	}{
		{
			name:       "scenario 1: use a valid service account token (static) to get a project",
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
			},
			existingSa:       test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "plan9-ID",
			expectedResponse: `{"id":"plan9-ID","name":"plan9","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// set up
			token := ""
			var ep http.Handler
			{
				req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens", tc.projectToSync, "1"), strings.NewReader(`{"name":"ci-v","group":"viewers"}`))
				res := httptest.NewRecorder()

				tc.existingKubermaticObjs = append(tc.existingKubermaticObjs, tc.existingSa)
				lep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
				if err != nil {
					t.Fatalf("failed to create test endpoint due to %v", err)
				}

				// act 1 - create a service account token
				lep.ServeHTTP(res, req)

				// validate
				if http.StatusCreated != res.Code {
					t.Fatalf("expected HTTP status code %d, got %d: %s", http.StatusCreated, res.Code, res.Body.String())
				}
				tokenRsp := &apiv1.ServiceAccountToken{}
				err = json.Unmarshal(res.Body.Bytes(), tokenRsp)
				if err != nil {
					t.Fatalf("unable to read the token from the response, err %v", err)
				}

				token = tokenRsp.Token
				ep = lep
			}

			// act 2 - get the project using sa token
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s", tc.projectToSync), strings.NewReader(""))
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
			res := httptest.NewRecorder()
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestPatchToken(t *testing.T) {
	t.Parallel()
	expiry, err := test.GenDefaultExpiry()
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		name                   string
		body                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
		expectedToken          apiv1.PublicServiceAccountToken
		expectedErrorMsg       string
		projectToSync          string
		saToSync               string
		tokenToSync            string
		httpStatus             int
		existingAPIUser        apiv1.User
	}{
		{
			name:       "scenario 1: change token name successfully",
			httpStatus: http.StatusOK,
			body:       `{"name":"test-new-name"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "1",
			tokenToSync:     "1",
			expectedToken:   genPublicServiceAccountToken("1", "test-new-name", expiry),
		},
		{
			name:       "scenario 2: changed name is empty",
			httpStatus: http.StatusBadRequest,
			body:       `{"name":""}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "plan9-ID",
			saToSync:         "1",
			tokenToSync:      "sa-token-1",
			expectedErrorMsg: `{"error":{"code":400,"message":"new name can not be empty"}}`,
		},
		{
			name:       "scenario 3: new name exists for other token",
			httpStatus: http.StatusConflict,
			body:       `{"name":"test-2"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-2", "2"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "plan9-ID",
			saToSync:         "1",
			tokenToSync:      "sa-token-1",
			expectedErrorMsg: `{"error":{"code":409,"message":"token \"test-2\" already exists"}}`,
		},
		{
			name:       "scenario 4: the admin can change any token name successfully",
			httpStatus: http.StatusOK,
			body:       `{"name":"test-new-name"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
			},
			existingAPIUser: *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "1",
			tokenToSync:     "1",
			expectedToken:   genPublicServiceAccountToken("1", "test-new-name", expiry),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens/%s", tc.projectToSync, tc.saToSync, tc.tokenToSync), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if len(tc.expectedErrorMsg) == 0 {
				var token apiv1.PublicServiceAccountToken
				err = json.Unmarshal(res.Body.Bytes(), &token)
				if err != nil {
					t.Fatal(err.Error())
				}

				if token.Name != tc.expectedToken.Name {
					t.Fatalf("expected new name %s got %s", tc.expectedToken.Name, token.Name)
				}
				if token.ID != tc.expectedToken.ID {
					t.Fatalf("expected ID %s got %s", tc.expectedToken.ID, token.ID)
				}
				if token.Expiry != tc.expectedToken.Expiry {
					t.Fatalf("expected expiry %v got %v", tc.expectedToken.Expiry, token.Expiry)
				}
			} else {
				test.CompareWithResult(t, res, tc.expectedErrorMsg)
			}
		})
	}
}

func TestUpdateToken(t *testing.T) {
	t.Parallel()
	expiry, err := test.GenDefaultExpiry()
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		name                   string
		body                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
		expectedToken          apiv1.PublicServiceAccountToken
		expectedErrorMsg       string
		projectToSync          string
		saToSync               string
		tokenToSync            string
		httpStatus             int
		existingAPIUser        apiv1.User
	}{
		{
			name:       "scenario 1: change token name successfully and regenerate token",
			httpStatus: http.StatusOK,
			body:       `{"name":"test-new-name", "id":"1"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "1",
			tokenToSync:     "1",
			expectedToken:   genPublicServiceAccountToken("1", "test-new-name", expiry),
		},
		{
			name:       "scenario 2: changed name is empty",
			httpStatus: http.StatusBadRequest,
			body:       `{"name":"","id":"sa-token-1"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "plan9-ID",
			saToSync:         "1",
			tokenToSync:      "sa-token-1",
			expectedErrorMsg: `{"error":{"code":400,"message":"new name can not be empty"}}`,
		},
		{
			name:       "scenario 3: new name exists for other token",
			httpStatus: http.StatusConflict,
			body:       `{"name":"test-2","id":"sa-token-1"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-2", "2"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "plan9-ID",
			saToSync:         "1",
			tokenToSync:      "sa-token-1",
			expectedErrorMsg: `{"error":{"code":409,"message":"token \"test-2\" already exists"}}`,
		},
		{
			name:       "scenario 4: the admin can change any token name successfully and regenerate token",
			httpStatus: http.StatusOK,
			body:       `{"name":"test-new-name", "id":"1"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
			},
			existingAPIUser: *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "1",
			tokenToSync:     "1",
			expectedToken:   genPublicServiceAccountToken("1", "test-new-name", expiry),
		},
		{
			name:       "scenario 5: the user Bob can change John's token name and regenerate token",
			httpStatus: http.StatusForbidden,
			body:       `{"name":"test-new-name", "id":"1"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", false),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
			},
			existingAPIUser:  *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:    "plan9-ID",
			saToSync:         "1",
			tokenToSync:      "1",
			expectedErrorMsg: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't belong to the given project = plan9-ID"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens/%s", tc.projectToSync, tc.saToSync, tc.tokenToSync), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if len(tc.expectedErrorMsg) == 0 {
				var token apiv1.ServiceAccountToken
				err = json.Unmarshal(res.Body.Bytes(), &token)
				if err != nil {
					t.Fatal(err.Error())
				}

				if token.Name != tc.expectedToken.Name {
					t.Fatalf("expected new name %s got %s", tc.expectedToken.Name, token.Name)
				}
				if token.ID != tc.expectedToken.ID {
					t.Fatalf("expected ID %s got %s", tc.expectedToken.ID, token.ID)
				}
				if token.Expiry == tc.expectedToken.Expiry {
					t.Fatalf("token should be regenerated and expiration times should not be equal but got %v", token.Expiry)
				}
				if token.Token == test.TestFakeToken {
					t.Fatalf("token should be regenerated")
				}

			} else {
				test.CompareWithResult(t, res, tc.expectedErrorMsg)
			}
		})
	}
}

func TestDeleteToken(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
		projectToSync          string
		saToSync               string
		tokenToDelete          string
		httpStatus             int
		existingAPIUser        apiv1.User
		expectedResponse       string
		privilegedOperation    bool
	}{
		{
			name:       "scenario 1: delete token",
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan10-ID", "serviceaccount-2", "test-2", "2"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-3", "3"),
				test.GenDefaultSaToken("plan11-ID", "serviceaccount-3", "test-4", "4"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "plan9-ID",
			saToSync:         "1",
			tokenToDelete:    "sa-token-3",
			expectedResponse: "{}",
		},
		{
			name:       "scenario 2: the admin can delete any token",
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan10-ID", "serviceaccount-2", "test-2", "2"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-3", "3"),
				test.GenDefaultSaToken("plan11-ID", "serviceaccount-3", "test-4", "4"),
			},
			existingAPIUser:     *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:       "plan9-ID",
			saToSync:            "1",
			tokenToDelete:       "sa-token-3",
			expectedResponse:    "{}",
			privilegedOperation: true,
		},
	}

	ctx := context.Background()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens/%s", tc.projectToSync, tc.saToSync, tc.tokenToDelete), strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, clientset, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			if _, err := clientset.FakeKubernetesCoreClient.CoreV1().Secrets("kubermatic").Get(ctx, tc.tokenToDelete, metav1.GetOptions{}); err != nil {
				t.Fatalf("failed to check token %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)

			expectedToken := &corev1.Secret{}
			if tc.privilegedOperation {
				err = clientset.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: tc.tokenToDelete, Namespace: "kubermatic"}, expectedToken)
			} else {
				_, err = clientset.FakeKubermaticClient.KubermaticV1().Users().Get(ctx, tc.tokenToDelete, metav1.GetOptions{})
			}
			if err == nil {
				t.Fatalf("failed to delete token %s", tc.tokenToDelete)
			}

		})
	}
}

func genPublicServiceAccountToken(id, name string, expiry apiv1.Time) apiv1.PublicServiceAccountToken {
	token := apiv1.PublicServiceAccountToken{}
	token.ID = id
	token.Name = name
	token.Expiry = expiry
	return token
}
