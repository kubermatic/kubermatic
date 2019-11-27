package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetAdmins(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
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
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("GET", "/api/v1/admin", strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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
		existingKubermaticObjs []runtime.Object
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
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true), genUser("John", "john@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                   "scenario 3: authorized user adds admin for non existing user",
			body:                   `{"email":"patric@acme.com","isAdmin":true}`,
			expectedResponse:       `{"error":{"code":500,"message":"the given user patric@acme.com was not found"}}`,
			httpStatus:             http.StatusInternalServerError,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true), genUser("John", "john@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 4
		{
			name:                   "scenario 4: authorized user wants change own role",
			body:                   `{"email":"bob@acme.com","isAdmin":true}`,
			expectedResponse:       `{"error":{"code":400,"message":"can not change own privileges"}}`,
			httpStatus:             http.StatusBadRequest,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true), genUser("John", "john@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("PUT", "/api/v1/admin", strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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
