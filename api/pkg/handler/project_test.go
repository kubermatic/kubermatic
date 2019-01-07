package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRenameProjectEndpoint(t *testing.T) {
	t.Parallel()

	oRef := metav1.OwnerReference{
		APIVersion: "kubermatic.io/v1",
		Kind:       "User",
		UID:        "",
		Name:       testUserName,
	}

	testcases := []struct {
		Name                      string
		Body                      string
		ProjectToRename           string
		ExpectedResponse          string
		HTTPStatus                int
		ExistingKubermaticObjects []runtime.Object
		ExistingAPIUser           apiv1.User
	}{
		{
			Name:            "scenario 1: rename existing project",
			Body:            `{"Name": "Super-Project"}`,
			HTTPStatus:      http.StatusOK,
			ProjectToRename: "my-first-project-ID",
			ExistingKubermaticObjects: []runtime.Object{
				// add some projects
				genProject("my-first-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp(), oRef),
				// add John
				genUser("JohnID", "John", "john@acme.com"),
				// make John the owner of the first project and the editor of the second
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
			},
			ExistingAPIUser: apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					ID: testUserName,
				},
				Email: testUserEmail,
			},
			ExpectedResponse: `{"metadata":{"name":"my-first-project-ID","creationTimestamp":"2013-02-03T19:54:00Z","ownerReferences":[{"apiVersion":"kubermatic.io/v1","kind":"User","name":"user1","uid":""}]},"spec":{"name":"Super-Project"},"status":{"phase":"Active"}}`,
		},
		{
			Name:            "scenario 2: rename existing project with existing name",
			Body:            `{"Name": "my-second-project"}`,
			HTTPStatus:      http.StatusConflict,
			ProjectToRename: "my-first-project-ID",
			ExistingKubermaticObjects: []runtime.Object{
				// add some projects
				genProject("my-first-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp(), oRef),
				genProject("my-second-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp().Add(time.Minute), oRef),
				genProject("my-third-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp().Add(2*time.Minute), oRef),
				// add John
				genUser("JohnID", "John", "john@acme.com"),
				// make John the owner of the projects
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
				genBinding("my-second-project-ID", "john@acme.com", "owners"),
				genBinding("my-third-project-ID", "john@acme.com", "owners"),
			},
			ExistingAPIUser: apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					ID: testUserName,
				},
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":409,"message":"project name \"my-second-project\" already exists"}}`,
		},
		{
			Name:            "scenario 3: rename not existing project",
			Body:            `{"Name": "Super-Project"}`,
			HTTPStatus:      http.StatusForbidden,
			ProjectToRename: "some-ID",
			ExistingKubermaticObjects: []runtime.Object{
				genProject("my-first-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp(), oRef),
				genUser("JohnID", "John", "john@acme.com"),
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
			},
			ExistingAPIUser: apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					ID: testUserName,
				},
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: The user \"john@acme.com\" doesn't belong to the given project = some-ID"}}`,
		},
		{
			Name:            "scenario 4: rename a project where user is an editor",
			Body:            `{"Name": "Super-Project"}`,
			HTTPStatus:      http.StatusOK,
			ProjectToRename: "my-second-project-ID",
			ExistingKubermaticObjects: []runtime.Object{
				genProject("my-first-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp(), oRef),
				genProject("my-second-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp().Add(time.Minute)),
				genProject("my-third-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp().Add(2*time.Minute)),
				genUser("JohnID", "John", "john@acme.com"),
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
				genBinding("my-second-project-ID", "john@acme.com", "editors"),
			},
			ExistingAPIUser: apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					ID: testUserName,
				},
				Email: testUserEmail,
			},
			ExpectedResponse: `{"metadata":{"name":"my-second-project-ID","creationTimestamp":"2013-02-03T19:55:00Z"},"spec":{"name":"Super-Project"},"status":{"phase":"Active"}}`,
		},
		{
			Name:            "scenario 5: rename a project with empty name",
			Body:            `{"Name": ""}`,
			HTTPStatus:      http.StatusBadRequest,
			ProjectToRename: "my-first-project-ID",
			ExistingKubermaticObjects: []runtime.Object{
				genProject("my-first-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp(), oRef),
				genUser("JohnID", "John", "john@acme.com"),
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
			},
			ExistingAPIUser: apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					ID: testUserName,
				},
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":400,"message":"the name of the project cannot be empty"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToRename), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListProjectEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		Body                      string
		ExpectedResponse          []apiv1.Project
		HTTPStatus                int
		ExistingKubermaticObjects []runtime.Object
		ExistingAPIUser           apiv1.User
	}{
		{
			Name:       "scenario 1: list projects that John is the member of",
			Body:       ``,
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjects: []runtime.Object{
				// add some projects
				genProject("my-first-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp()),
				genProject("my-second-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp().Add(time.Minute)),
				genProject("my-third-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				genUser("JohnID", "John", "john@acme.com"),
				// make John the owner of the first project and the editor of the second
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
				genBinding("my-third-project-ID", "john@acme.com", "editors"),
			},
			ExistingAPIUser: apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					ID: testUserName,
				},
				Email: testUserEmail,
			},
			ExpectedResponse: []apiv1.Project{
				{
					Status: "Active",
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "my-first-project-ID",
						Name:              "my-first-project",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
				},
				{
					Status: "Active",
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "my-third-project-ID",
						Name:              "my-third-project",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 56, 0, 0, time.UTC),
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("GET", "/api/v1/projects", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualProjects := projectV1SliceWrapper{}
			actualProjects.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedProjects := projectV1SliceWrapper(tc.ExpectedResponse)
			wrappedExpectedProjects.Sort()

			actualProjects.EqualOrDie(wrappedExpectedProjects, t)
		})
	}
}

func TestGetProjectEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		Body                      string
		ProjectToSync             string
		ExpectedResponse          string
		HTTPStatus                int
		ExistingKubermaticUser    *kubermaticapiv1.User
		ExistingKubermaticObjects []runtime.Object
		ExistingAPIUser           *apiv1.User
	}{
		{
			Name:                      "scenario 1: get an existing project assigned to the given user",
			Body:                      ``,
			ProjectToSync:             genDefaultProject().Name,
			ExpectedResponse:          `{"id":"my-first-project-ID","name":"my-first-project","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active"}`,
			HTTPStatus:                http.StatusOK,
			ExistingKubermaticObjects: genDefaultKubermaticObjects(),
			ExistingAPIUser:           genDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestCreateProjectEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		Body                      string
		RewriteProjectID          bool
		ExpectedResponse          string
		HTTPStatus                int
		ExistingKubermaticObjects []runtime.Object
		ExistingAPIUser           *apiv1.User
	}{
		{
			Name:             "scenario 1: a user doesn't have any projects, thus creating one succeeds",
			Body:             `{"name":"my-first-project"}`,
			RewriteProjectID: true,
			ExpectedResponse: `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive"}`,
			HTTPStatus:       http.StatusCreated,
			ExistingKubermaticObjects: []runtime.Object{
				genDefaultUser(),
			},
			ExistingAPIUser: genDefaultAPIUser(),
		},

		{
			Name:                      "scenario 2: a user has a project with the given name, thus creating one fails",
			Body:                      fmt.Sprintf(`{"name":"%s"}`, genDefaultProject().Spec.Name),
			ExpectedResponse:          `{"error":{"code":409,"message":"project name \"my-first-project\" already exists"}}`,
			HTTPStatus:                http.StatusConflict,
			ExistingKubermaticObjects: genDefaultKubermaticObjects(),
			ExistingAPIUser:           genDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// valdiate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			expectedResponse := tc.ExpectedResponse
			// since Project.ID is automatically generated by the system just rewrite it.
			if tc.RewriteProjectID {
				actualProject := &apiv1.Project{}
				err = json.Unmarshal(res.Body.Bytes(), actualProject)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualProject.ID)
			}
			compareWithResult(t, res, expectedResponse)

		})
	}
}

func TestDeleteProjectEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		HTTPStatus                int
		ProjectToSync             string
		ExistingKubermaticObjects []runtime.Object
		ExistingAPIUser           *apiv1.User
	}{
		{
			Name:                      "scenario 1: the owner of the project can delete the project",
			HTTPStatus:                http.StatusOK,
			ProjectToSync:             genDefaultProject().Name,
			ExistingKubermaticObjects: genDefaultKubermaticObjects(),
			ExistingAPIUser:           genDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected route to return code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
		})
	}
}

const testingProjectName = "my-first-project-ID"

func defaultCreationTimestamp() time.Time {
	return test.DefaultCreationTimestamp()
}

func genProject(name, phase string, creationTime time.Time, oRef ...metav1.OwnerReference) *kubermaticapiv1.Project {
	return test.GenProject(name, phase, creationTime, oRef...)
}

func genDefaultProject() *kubermaticapiv1.Project {
	return test.GenDefaultProject()
}

func genDefaultKubermaticObjects(objs ...runtime.Object) []runtime.Object {
	return test.GenDefaultKubermaticObjects(objs...)
}
