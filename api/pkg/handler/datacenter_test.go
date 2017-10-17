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

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, nil, nil)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, "[{\"metadata\":{\"name\":\"regular-do1\",\"revision\":\"1\"},\"spec\":{\"country\":\"NL\",\"location\":\"Amsterdam\",\"provider\":\"digitalocean\",\"digitalocean\":{\"region\":\"ams2\"}}}]")
}

func TestDatacenterEndpointNotFound(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/not-existent", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, nil, nil)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("Expected route to return code 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDatacenterEndpointPrivate(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/eu-central-1", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, nil, nil)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("Expected route to return code 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDatacenterEndpointAdmin(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/private-do1", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(testUsername, true), []runtime.Object{}, nil, nil)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, "{\"metadata\":{\"name\":\"private-do1\",\"revision\":\"1\"},\"spec\":{\"country\":\"NL\",\"location\":\"US \",\"provider\":\"digitalocean\",\"digitalocean\":{\"region\":\"ams2\"}}}")

}

func TestDatacenterEndpointFound(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/dc/regular-do1", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, nil, nil)
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, "{\"metadata\":{\"name\":\"regular-do1\",\"revision\":\"1\"},\"spec\":{\"country\":\"NL\",\"location\":\"Amsterdam\",\"provider\":\"digitalocean\",\"digitalocean\":{\"region\":\"ams2\"}}}")
}
