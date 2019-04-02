package serviceaccount_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateTokenProject(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		existingKubermaticObjs []runtime.Object
		existingKubernetesObjs []runtime.Object
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
			existingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genActiveServiceAccount("1", "test-1", "editors", "plan9-ID"),
				genActiveServiceAccount("2", "test-2", "editors", "test-ID"),
				genActiveServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingKubernetesObjs: []runtime.Object{},
			existingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:          "plan9-ID",
			saToSync:               "1",
			expectedName:           "test",
		},
		{
			name:       "scenario 2: create service account token with existing name",
			body:       `{"name":"test"}`,
			httpStatus: http.StatusConflict,
			existingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genActiveServiceAccount("1", "test-1", "editors", "plan9-ID"),
				genActiveServiceAccount("2", "test-2", "editors", "test-ID"),
				genActiveServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingKubernetesObjs: []runtime.Object{
				test.GenSecret("plan9-ID", "serviceaccount-1", "test", "1"),
			},
			existingAPIUser:       *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:         "plan9-ID",
			saToSync:              "1",
			expectedErrorResponse: `{"error":{"code":409,"message":"token \"test\" already exists"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens", tc.projectToSync, tc.saToSync), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			ep, fakeClients, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []runtime.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
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
		existingKubermaticObjs []runtime.Object
		existingKubernetesObjs []runtime.Object
		expectedTokens         []apiv1.PublicServiceAccountToken
		projectToSync          string
		saToSync               string
		httpStatus             int
		existingAPIUser        apiv1.User
	}{
		{
			name:       "scenario 1: list tokens",
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genActiveServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingKubernetesObjs: []runtime.Object{
				test.GenSecret("plan9-ID", "serviceaccount-1", "test-1", "1"),
				test.GenSecret("plan10-ID", "serviceaccount-2", "test-2", "2"),
				test.GenSecret("plan9-ID", "serviceaccount-1", "test-3", "3"),
				test.GenSecret("plan11-ID", "serviceaccount-3", "test-4", "4"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "1",
			expectedTokens: []apiv1.PublicServiceAccountToken{
				genPublicServiceAccountToken("sa-token-1", "test-1", expiry),
				genPublicServiceAccountToken("sa-token-3", "test-3", expiry),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens", tc.projectToSync, tc.saToSync), strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, tc.existingKubernetesObjs, []runtime.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
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

func genPublicServiceAccountToken(id, name string, expiry apiv1.Time) apiv1.PublicServiceAccountToken {
	token := apiv1.PublicServiceAccountToken{}
	token.ID = id
	token.Name = name
	token.Expiry = expiry
	return token
}
