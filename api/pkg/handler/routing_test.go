package handler_test

import (
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	testUserID    = test.UserID
	testUserName  = test.UserName
	testUserEmail = test.UserEmail
)

func TestUpRoute(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	res := httptest.NewRecorder()
	ep, err := test.CreateTestEndpoint(test.GetUser(testUserEmail, testUserID, testUserName, false), []runtime.Object{}, []runtime.Object{}, nil, nil, hack.NewTestRouting)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)
	test.CheckStatusCode(http.StatusOK, res, t)
}
