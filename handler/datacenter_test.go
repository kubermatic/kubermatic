package handler

import (
	"net/http/httptest"
	"testing"
)

func TestDatacentersEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc", nil)
	authenticateHeader(req)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Fatalf("Expected route to return code 200, got %d", res.Code)
	}

	compareWithResult(t, res, "./fixtures/responses/datacenters.json")
}
