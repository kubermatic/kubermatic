package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-test/deep"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestKubermaticVersion(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	res := httptest.NewRecorder()
	ep, err := test.CreateTestEndpoint(*test.GenDefaultAPIUser(), []runtime.Object{}, nil, nil, nil, hack.NewTestRouting)
	if err != nil {
		t.Fatalf("failed to create testStruct endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
	}

	var gotVersion apiv1.KubermaticVersions
	err = json.Unmarshal(res.Body.Bytes(), &gotVersion)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersions := apiv1.KubermaticVersions{
		API: "manual_build",
	}

	if diff := deep.Equal(gotVersion, expectedVersions); diff != nil {
		t.Fatalf("got different upgrade response than expected. Diff: %v", diff)
	}
}
