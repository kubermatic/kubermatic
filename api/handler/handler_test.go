package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIDescriptionHandlerStatusNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/", nil)
	if err != nil {
		t.Fatal(err)
	}
	res := httptest.NewRecorder()
	e := http.HandlerFunc(APIDescriptionHandler)
	e.ServeHTTP(res, req)

	if res.Code!= http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			res.Code, http.StatusNotFound)
	}
}