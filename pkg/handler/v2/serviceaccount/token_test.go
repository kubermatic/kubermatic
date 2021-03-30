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
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("4", "test-4", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan10-ID", "serviceaccount-2", "test-2", "2"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-3", "3"),
				test.GenDefaultSaToken("plan11-ID", "serviceaccount-3", "test-4", "4"),

				test.GenDefaultSaToken("", "4", "test-1", "5"),
				test.GenDefaultSaToken("", "4", "test-2", "6"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			saToSync:        "4",
			expectedTokens: []apiv1.PublicServiceAccountToken{
				genPublicServiceAccountToken("5", "test-1", expiry),
				genPublicServiceAccountToken("6", "test-2", expiry),
			},
		},
		{
			name:       "scenario 2: user John can't list Bob's tokens",
			httpStatus: http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenUser("", "bob", "bob@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("4", "test-4", "viewers", "bob@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan10-ID", "serviceaccount-2", "test-2", "2"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-3", "3"),
				test.GenDefaultSaToken("plan11-ID", "serviceaccount-3", "test-4", "4"),

				test.GenDefaultSaToken("", "4", "test-1", "5"),
				test.GenDefaultSaToken("", "4", "test-2", "6"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			saToSync:        "4",
			expectedTokens: []apiv1.PublicServiceAccountToken{
				genPublicServiceAccountToken("5", "test-1", expiry),
				genPublicServiceAccountToken("6", "test-2", expiry),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/serviceaccounts/%s/tokens", tc.saToSync), strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if res.Code == http.StatusOK {
				actualSA := test.NewServiceAccountTokenV1SliceWrapper{}
				actualSA.DecodeOrDie(res.Body, t).Sort()

				wrappedExpectedToken := test.NewServiceAccountTokenV1SliceWrapper(tc.expectedTokens)
				wrappedExpectedToken.Sort()

				actualSA.EqualOrDie(wrappedExpectedToken, t)
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
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-4", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("", "2", "test-4", "1"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			saToSync:        "2",
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
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-4", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("", "2", "test-1", "1"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			saToSync:         "2",
			tokenToSync:      "1",
			expectedErrorMsg: `{"error":{"code":400,"message":"new name can not be empty"}}`,
		},
		{
			name:       "scenario 3: new name exists for other token",
			httpStatus: http.StatusConflict,
			body:       `{"name":"test-2","id":"3"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-4", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-2", "2"),
				test.GenDefaultSaToken("", "2", "test-1", "3"),
				test.GenDefaultSaToken("", "2", "test-2", "4"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			saToSync:         "2",
			tokenToSync:      "3",
			expectedErrorMsg: `{"error":{"code":409,"message":"token \"test-2\" already exists"}}`,
		},
		{
			name:       "scenario 4: the user Bob can't change John's token name and regenerate token",
			httpStatus: http.StatusForbidden,
			body:       `{"name":"test-new-name", "id":"2"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenUser("", "bob", "bob@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-4", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenDefaultSaToken("", "2", "test-2", "2"),
			},
			existingAPIUser:  *test.GenAPIUser("bob", "bob@acme.com"),
			saToSync:         "2",
			tokenToSync:      "2",
			expectedErrorMsg: `{"error":{"code":403,"message":"forbidden: actual user bob@acme.com is not the owner of the service account main-serviceaccount-2"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v2/serviceaccounts/%s/tokens/%s", tc.saToSync, tc.tokenToSync), strings.NewReader(tc.body))
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
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-2", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("", "2", "test-1", "1"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			saToSync:        "2",
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
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-2", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("", "2", "test-1", "1"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			saToSync:         "2",
			tokenToSync:      "1",
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
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-2", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("", "2", "test-1", "1"),
				test.GenDefaultSaToken("", "2", "test-2", "2"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			saToSync:         "2",
			tokenToSync:      "1",
			expectedErrorMsg: `{"error":{"code":409,"message":"token \"test-2\" already exists"}}`,
		},
		{
			name:       "scenario 4: the user Bob can't change any John's token name",
			httpStatus: http.StatusForbidden,
			body:       `{"name":"test-new-name"}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenUser("", "bob", "bob@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenMainServiceAccount("2", "test-2", "viewers", "john@acme.com"),
			},
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultSaToken("", "2", "test-1", "1"),
			},
			existingAPIUser:  *test.GenAPIUser("bob", "bob@acme.com"),
			saToSync:         "2",
			tokenToSync:      "1",
			expectedErrorMsg: `{"error":{"code":403,"message":"forbidden: actual user bob@acme.com is not the owner of the service account main-serviceaccount-2"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v2/serviceaccounts/%s/tokens/%s", tc.saToSync, tc.tokenToSync), strings.NewReader(tc.body))
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

func genPublicServiceAccountToken(id, name string, expiry apiv1.Time) apiv1.PublicServiceAccountToken {
	token := apiv1.PublicServiceAccountToken{}
	token.ID = id
	token.Name = name
	token.Expiry = expiry
	return token
}
