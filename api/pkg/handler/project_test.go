package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateProjectEndpoint(t *testing.T) {
	testcases := []struct {
		Name            string
		Body            string
		Response        string
		HTTPStatus      int
		ExistingProject *kubermaticapiv1.Project
	}{
		{
			Name:     "scenario 1: a user doesn't have any projects, thus creating one succeeds",
			Body:     `{"name":"my-first-project"}`,
			Response: `{"id":"","name":"my-first-project","status":"Inactive"}`,
			// TODO(p0lyn0mial): the response should be http.StatusCreated
			HTTPStatus: http.StatusOK,
		},

		// TODO(p0lyn0mial): the following test case fails with:
		// more than one object matched gvr {kubermatic.k8s.io v1 projects}, ns: \"\" name: \"\""
		/*{
			Name:     "scenario 2: a user has a project with the given name, thus creating one fails",
			Body: `{"name":"my-first-project"}`,
			Response: `{"id":"","name":"my-first-project"}`,
			HTTPStatus: http.StatusOK,
			ExistingProject: &kubermaticapiv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "randomName"},Spec: kubermaticapiv1.ProjectSpec{Name : "my-first-project"}},
		},*/
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = []runtime.Object{tc.ExistingProject}
			}
			ep, err := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected route to return code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.Response)

		})
	}
}
