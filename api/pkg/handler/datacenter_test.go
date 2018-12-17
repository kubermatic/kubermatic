package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestDatacentersEndpoint(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc", nil)
	apiUser := getUser(testUserEmail, testUserID, testUserName, false)

	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{apiUserToKubermaticUser(apiUser)}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, `[{"metadata":{"name":"private-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"}}},{"metadata":{"name":"regular-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"}}},{"metadata":{"name":"us-central1","resourceVersion":"1"},"spec":{"seed":"","country":"US","location":"us-central","provider":"digitalocean","digitalocean":{"region":"ams2"}},"seed":true}]`)
}

func TestDatacenterEndpointNotFound(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/not-existent", nil)
	apiUser := getUser(testUserEmail, testUserID, testUserName, false)

	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{apiUserToKubermaticUser(apiUser)}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("Expected route to return code 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDatacenterEndpointPrivate(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/eu-central-1", nil)
	apiUser := getUser(testUserEmail, testUserID, testUserName, false)

	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{apiUserToKubermaticUser(apiUser)}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("Expected route to return code 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDatacenterEndpointAdmin(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/private-do1", nil)
	apiUser := getUser(testUserEmail, testUserID, testUserName, true)

	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{apiUserToKubermaticUser(apiUser)}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, `{"metadata":{"name":"private-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"}}}`)

}

func TestDatacenterEndpointFound(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/regular-do1", nil)
	apiUser := getUser(testUserEmail, testUserID, testUserName, false)

	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{apiUserToKubermaticUser(apiUser)}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, `{"metadata":{"name":"regular-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"}}}`)
}
