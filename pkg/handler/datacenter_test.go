package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatacentersEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, "./fixtures/responses/datacenters.json")
}

func TestDatacenterEndpointNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/not-existent", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("Expected route to return code 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDatacenterEndpointPrivate(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/eu-central-1", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("Expected route to return code 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDatacenterEndpointAdmin(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/eu-central-1", nil)
	authenticateHeader(req, true)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, "./fixtures/responses/datacenter_private.json")

}

func TestDatacenterEndpointFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/us-west-2", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, "./fixtures/responses/datacenter_public.json")
}
