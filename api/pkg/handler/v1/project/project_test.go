package project_test

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
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRenameProjectEndpoint(t *testing.T) {
	t.Parallel()

	oRef := func(user *kubermaticapiv1.User) metav1.OwnerReference {
		return metav1.OwnerReference{
			APIVersion: "kubermatic.io/v1",
			Kind:       "User",
			UID:        user.UID,
			Name:       user.Name,
		}
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
			ProjectToRename: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: []runtime.Object{
				test.GenDefaultProject(),
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
			},
			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: `{"id":"my-first-project-ID","name":"Super-Project","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
		},
		{
			Name:            "scenario 2: rename existing project with existing name",
			Body:            `{"Name": "my-second-project"}`,
			HTTPStatus:      http.StatusConflict,
			ProjectToRename: "my-first-project-ID",
			ExistingKubermaticObjects: []runtime.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp(), oRef(test.GenDefaultUser())),
				test.GenProject("my-second-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(time.Minute), oRef(test.GenDefaultUser())),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute), oRef(test.GenDefaultUser())),
				test.GenDefaultUser(),
				test.GenBinding("my-first-project-ID", test.GenDefaultUser().Spec.Email, "owners"),
				test.GenBinding("my-second-project-ID", test.GenDefaultUser().Spec.Email, "owners"),
				test.GenBinding("my-third-project-ID", test.GenDefaultUser().Spec.Email, "owners"),
			},
			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: `{"error":{"code":409,"message":"project name \"my-second-project\" already exists"}}`,
		},
		{
			Name:            "scenario 3: rename existing project with existing name where user is not the owner",
			Body:            `{"Name": "my-second-project"}`,
			HTTPStatus:      http.StatusOK,
			ProjectToRename: "my-first-project-ID",
			ExistingKubermaticObjects: []runtime.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp(), oRef(test.GenDefaultUser())),
				test.GenProject("my-second-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(time.Minute), oRef(test.GenDefaultUser())),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute), oRef(test.GenDefaultUser())),
				// add John and Bob
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenDefaultUser(),
				// make John the owner of the projects
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-second-project-ID", test.GenDefaultUser().Spec.Email, "editors"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "owners"),
			},
			ExistingAPIUser:  *test.GenAPIUser("John", "john@acme.com"),
			ExpectedResponse: `{"id":"my-first-project-ID","name":"my-second-project","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"John","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
		},

		{
			Name:            "scenario 4: rename not existing project",
			Body:            `{"Name": "Super-Project"}`,
			HTTPStatus:      http.StatusForbidden,
			ProjectToRename: "some-ID",
			ExistingKubermaticObjects: []runtime.Object{
				test.GenDefaultProject(),
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
			},
			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: The user \"bob@acme.com\" doesn't belong to the given project = some-ID"}}`,
		},
		{
			Name:            "scenario 5: rename a project with empty name",
			Body:            `{"Name": ""}`,
			HTTPStatus:      http.StatusBadRequest,
			ProjectToRename: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: []runtime.Object{
				test.GenDefaultProject(),
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
			},
			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: `{"error":{"code":400,"message":"the name of the project cannot be empty"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToRename), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)
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
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-second-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(time.Minute)),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "editors"),
			},
			ExistingAPIUser: func() apiv1.User {
				apiUser := test.GenDefaultAPIUser()
				apiUser.Email = "john@acme.com"
				return *apiUser
			}(),
			ExpectedResponse: []apiv1.Project{
				{
					Status: "Active",
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "my-first-project-ID",
						Name:              "my-first-project",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
					Owners: []apiv1.User{
						{
							ObjectMeta: apiv1.ObjectMeta{
								Name: "John",
							},
							Email: "john@acme.com",
						},
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
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualProjects := test.ProjectV1SliceWrapper{}
			actualProjects.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedProjects := test.ProjectV1SliceWrapper(tc.ExpectedResponse)
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
			ProjectToSync:             test.GenDefaultProject().Name,
			ExpectedResponse:          `{"id":"my-first-project-ID","name":"my-first-project","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
			HTTPStatus:                http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)
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
			ExpectedResponse: `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
			HTTPStatus:       http.StatusCreated,
			ExistingKubermaticObjects: []runtime.Object{
				test.GenDefaultUser(),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},

		{
			Name:                      "scenario 2: a user has a project with the given name, thus creating one fails",
			Body:                      fmt.Sprintf(`{"name":"%s"}`, test.GenDefaultProject().Spec.Name),
			ExpectedResponse:          `{"error":{"code":409,"message":"projects.kubermatic.k8s.io \"my-first-project\" already exists"}}`,
			HTTPStatus:                http.StatusConflict,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
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
			test.CompareWithResult(t, res, expectedResponse)

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
			ProjectToSync:             test.GenDefaultProject().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
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
