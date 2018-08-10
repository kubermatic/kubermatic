package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
)

func TestListProjectEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       []apiv1.Project
		HTTPStatus             int
		ExistingProjects       []*kubermaticapiv1.Project
		ExistingKubermaticUser *kubermaticapiv1.User
		ExistingAPIUser        apiv1.User
	}{
		{
			Name:       "scenario 1: list projects that the user is member of",
			Body:       ``,
			HTTPStatus: http.StatusOK,
			ExistingProjects: []*kubermaticapiv1.Project{
				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myProjectInternalName",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "my-first-project",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"},
				},

				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mySecondProjectInternalName",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "my-second-project",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-second-project"},
				},

				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myThirdProjectInternalName",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "my-third-project",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-third-project"},
				},
			},
			ExistingKubermaticUser: &kubermaticapiv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticapiv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticapiv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
						{
							Group: "editors-myThirdProjectInternalName",
							Name:  "myThirdProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserName,
				Email: testEmail,
			},
			ExpectedResponse: []apiv1.Project{
				apiv1.Project{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:   "myProjectInternalName",
						Name: "my-first-project",
					},
				},

				apiv1.Project{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:   "myThirdProjectInternalName",
						Name: "my-third-project",
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/projects", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			for _, existingProject := range tc.ExistingProjects {
				kubermaticObj = append(kubermaticObj, existingProject)
			}
			kubermaticObj = append(kubermaticObj, runtime.Object(tc.ExistingKubermaticUser))
			ep, err := createTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			// validate the response, we actually need to compare the output against ExpectedResponse
			// note that the output contains the list which is not stable (sorted)
			{
				projectsFromResponse := []apiv1.Project{}
				rawBody, err := ioutil.ReadAll(res.Body)
				if err != nil {
					t.Fatal(err)
				}

				err = json.Unmarshal(rawBody, &projectsFromResponse)
				if err != nil {
					t.Fatal(err)
				}

				if len(projectsFromResponse) != len(tc.ExpectedResponse) {
					t.Fatalf("expected to get %d keys but got %d", len(tc.ExpectedResponse), len(projectsFromResponse))
				}

				for _, expectedProject := range tc.ExpectedResponse {
					found := false
					for _, actualProject := range projectsFromResponse {
						if actualProject.ID == expectedProject.ID {
							if !areEqualOrDie(t, actualProject, expectedProject) {
								t.Fatalf("actual project != expected project, diff = %v", diff.ObjectDiff(actualProject, expectedProject))
							}
							found = true
						}
					}
					if !found {
						t.Fatalf("the project with the name = %s was not found in the returned output", expectedProject.Name)
					}
				}
			}
		})
	}
}

func TestGetProjectEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ProjectToSync          string
		ExpectedResponse       string
		HTTPStatus             int
		ExistingProject        *kubermaticapiv1.Project
		ExistingKubermaticUser *kubermaticapiv1.User
		ExistingAPIUser        apiv1.User
	}{
		{
			Name:             "scenario 1: get an existing project assigned to the given user",
			Body:             ``,
			ProjectToSync:    `myProjectInternalName`,
			ExpectedResponse: `{"id":"myProjectInternalName","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":""}`,
			HTTPStatus:       http.StatusOK,
			ExistingProject: &kubermaticapiv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "my-first-project",
						},
					},
				},
				Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticapiv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticapiv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticapiv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserName,
				Email: testEmail,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = []runtime.Object{tc.ExistingProject}
			}
			kubermaticObj = append(kubermaticObj, runtime.Object(tc.ExistingKubermaticUser))
			ep, err := createTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

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
		Name                   string
		Body                   string
		RewriteProjectID       bool
		ExpectedResponse       string
		HTTPStatus             int
		ExistingProject        *kubermaticapiv1.Project
		ExistingKubermaticUser *kubermaticapiv1.User
		ExistingAPIUser        apiv1.User
	}{
		{
			Name:             "scenario 1: a user doesn't have any projects, thus creating one succeeds",
			Body:             `{"name":"my-first-project"}`,
			RewriteProjectID: true,
			ExpectedResponse: `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive"}`,
			HTTPStatus:       http.StatusCreated,
			ExistingAPIUser: apiv1.User{
				ID:    testUserName,
				Email: testEmail,
			},
		},

		{
			Name:             "scenario 2: a user has a project with the given name, thus creating one fails",
			Body:             `{"name":"my-first-project"}`,
			ExpectedResponse: `{"error":{"code":409,"message":"projects.kubermatic.k8s.io \"my-first-project\" already exists"}}`,
			HTTPStatus:       http.StatusConflict,
			ExistingProject: &kubermaticapiv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "my-first-project",
						},
					},
				},
				Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticapiv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticapiv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticapiv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserName,
				Email: testEmail,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = []runtime.Object{tc.ExistingProject}
			}
			ep, err := createTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

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
		Name                   string
		HTTPStatus             int
		ExistingKubermaticUser *kubermaticapiv1.User
		ExistingAPIUser        *apiv1.User
		ExistingProject        *kubermaticapiv1.Project
	}{
		{
			Name:       "scenario 1: the user is the owner of the project thus can delete the project",
			HTTPStatus: http.StatusOK,
			ExistingKubermaticUser: &kubermaticapiv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticapiv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticapiv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testEmail,
			},
			ExistingProject: &kubermaticapiv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "myProjectInternalName"}, Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"}},
		},
		{
			Name:       "scenario 2: the user is NOT the owner of the project thus cannot delete the project",
			HTTPStatus: http.StatusForbidden,
			ExistingKubermaticUser: &kubermaticapiv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticapiv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticapiv1.ProjectGroup{
						{
							Group: "owners-mySecondProjectInternalName",
							Name:  "mySecondProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testEmail,
			},
			ExistingProject: &kubermaticapiv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "myProjectInternalName"}, Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"}},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", tc.ExistingProject.Name), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, runtime.Object(tc.ExistingProject))
			kubermaticObj = append(kubermaticObj, runtime.Object(tc.ExistingKubermaticUser))
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected route to return code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
		})
	}
}
