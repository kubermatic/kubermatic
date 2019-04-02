package serviceaccount_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"

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
		existingKubermaticObjs []runtime.Object
		expectedResponse       string
		projectToSync          string
		saToSync               string
		httpStatus             int
		existingAPIUser        apiv1.User
	}{
		{
			name:       "scenario 1: create service account token for serviceaccount-1",
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
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:   "plan9-ID",
			saToSync:        "serviceaccount-1",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s/tokens", tc.projectToSync, tc.saToSync), strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if tc.httpStatus == http.StatusCreated {
				var saToken apiv1.ServiceAccountToken
				err = json.Unmarshal(res.Body.Bytes(), &saToken)
				if err != nil {
					t.Fatal(err)
				}

				tokenAuthenticator := serviceaccount.JWTTokenAuthenticator([]byte(test.TestServiceAccountHashKey))
				saTokenClaim, err := tokenAuthenticator.AuthenticateToken(saToken.Token)
				if err != nil {
					t.Fatal(err)
				}
				if saTokenClaim.TokenID != saToken.Name {
					t.Fatalf("expected token name %s got %s", saToken.Name, saTokenClaim.TokenID)
				}
				if saTokenClaim.ProjectID != tc.projectToSync {
					t.Fatalf("expected project name %s got %s", tc.projectToSync, saTokenClaim.ProjectID)
				}
				if saTokenClaim.Email != fmt.Sprintf("%s@sa.kubermatic.io", tc.saToSync) {
					t.Fatalf("expected email %s@sa.kubermatic.io got %s", tc.saToSync, saTokenClaim.Email)
				}

			} else {
				test.CompareWithResult(t, res, tc.expectedResponse)
			}

		})
	}
}
