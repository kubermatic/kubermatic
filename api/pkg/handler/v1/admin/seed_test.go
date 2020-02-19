package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestListSeedsEndpoint(t *testing.T) {
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
			name:                   "scenario 1: not authorized user gets seeds",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user gets default list",
			expectedResponse:       `[{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henriks basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}]`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("GET", "/api/v1/admin/seeds", strings.NewReader(""))
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

func TestGetSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		seedName               string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user gets seeds",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":404,"message":"Seed \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                   "scenario 3: authorized user gets seed",
			seedName:               "us-central1",
			expectedResponse:       `{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henriks basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/seeds/%s", tc.seedName), strings.NewReader(""))
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

func TestUpdateSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		body                   string
		seedName               string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user updates seeds",
			body:                   `{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henriks basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}`,
			seedName:               "us-central1",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			body:                   `{"name":"test","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henriks basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}`,
			seedName:               "test",
			expectedResponse:       `{"error":{"code":404,"message":"Seed \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                   "scenario 3: seed name mismatch",
			body:                   `{"name":"central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henriks basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}`,
			seedName:               "us-central1",
			expectedResponse:       `{"error":{"code":400,"message":"seed name mismatch, you requested to update Seed = us-central1 but body contains Seed = central1"}}`,
			httpStatus:             http.StatusBadRequest,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 4
		{
			name:                   "scenario 4: authorized user updates seed",
			body:                   `{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henriks basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomains":["example.com","noifsdsd.org"],"enforceAuditLogging":true}}}}}`,
			seedName:               "us-central1",
			expectedResponse:       `{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henriks basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmailDomains":["example.com","noifsdsd.org"],"enforceAuditLogging":true}}}}}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/admin/seeds/%s", tc.seedName), strings.NewReader(tc.body))
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

func TestDeleteSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		seedName               string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user tries to delete seed cluster",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user tries to delete not existing seed cluster",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":404,"message":"Seed \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                   "scenario 3: authorized user tries to delete seed cluster",
			seedName:               "us-central1",
			expectedResponse:       `{}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/seeds/%s", tc.seedName), strings.NewReader(""))
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
