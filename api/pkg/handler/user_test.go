package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	clienttesting "k8s.io/client-go/testing"
)

func TestAddUserToProject(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                        string
		Body                        string
		ExpectedResponse            string
		ExpectedActions             int
		ExpectedUserAfterInvitation *kubermaticapiv1.User
		ProjectToSync               string
		HTTPStatus                  int
		ExistingProjects            []*kubermaticapiv1.Project
		ExistingKubermaticUsers     []*kubermaticapiv1.User
		ExistingAPIUser             apiv1.User
	}{
		{
			Name:          "scenario 1: john the owner of the plan9 project invites bob to the project as an editor",
			Body:          `{"email":"bob@acme.com", "projects":[{"id":"plan9", "group":"editors"}]}`,
			HTTPStatus:    http.StatusCreated,
			ProjectToSync: "plan9",
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
						Name: "plan9",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "John",
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
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "readers-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-placeX",
								Name:  "placeX",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"id":"bob","name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com","projects":[{"id":"placeX","group":"editors-placeX"},{"id":"plan9","group":"editors-plan9"}]}`,
			ExpectedActions:  10,
			ExpectedUserAfterInvitation: &kubermaticapiv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bob",
				},
				Spec: kubermaticapiv1.UserSpec{
					Name:  "Bob",
					Email: "bob@acme.com",
					Projects: []kubermaticapiv1.ProjectGroup{
						{
							Group: "editors-placeX",
							Name:  "placeX",
						},
						{
							Group: "editors-plan9",
							Name:  "plan9",
						},
					},
				},
			},
		},

		{
			Name:          "scenario 2: john the editor of the plan9 project tries to invite bob to the project",
			Body:          `{"email":"bob@acme.com", "projects":[{"id":"plan9", "group":"editors"}]}`,
			HTTPStatus:    http.StatusForbidden,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myProjectInternalName",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "Joe",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"},
				},

				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "plan9",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "John",
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
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-placeX",
								Name:  "placeX",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				Name:  testUserName,
				ID:    testUserID,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"only the owner of the project can invite the other users"}}`,
			ExpectedActions:  9,
		},

		{
			Name:          "scenario 3: john the owner of the plan9 project tries to invite bob to another project",
			Body:          `{"email":"bob@acme.com", "projects":[{"id":"moby", "group":"editors"}]}`,
			HTTPStatus:    http.StatusForbidden,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "moby",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "Joe",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"},
				},

				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "plan9",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "John",
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
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-placeX",
								Name:  "placeX",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				Name:  testUserName,
				ID:    testUserID,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"you can only assign the user to plan9 project"}}`,
			ExpectedActions:  8,
		},

		{
			Name:          "scenario 4: john the owner of the plan9 project tries to invite  himself to another group",
			Body:          fmt.Sprintf(`{"email":"%s", "projects":[{"id":"plan9", "group":"editors"}]}`, testUserEmail),
			HTTPStatus:    http.StatusForbidden,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "moby",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "Joe",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"},
				},

				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "plan9",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "John",
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
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-placeX",
								Name:  "placeX",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				Name:  testUserName,
				ID:    testUserID,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"you cannot assign yourself to a different group"}}`,
			ExpectedActions:  8,
		},

		{
			Name:          "scenario 5: john the owner of the plan9 project tries to invite bob to the project as an owner",
			Body:          `{"email":"bob@acme.com", "projects":[{"id":"plan9", "group":"owners"}]}`,
			HTTPStatus:    http.StatusForbidden,
			ProjectToSync: "plan9",
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
						Name: "plan9",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "John",
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
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bob",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-placeX",
								Name:  "placeX",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				Name:  testUserName,
				ID:    testUserID,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"the given user cannot be assigned to owners group"}}`,
			ExpectedActions:  8,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/users", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			for _, existingProject := range tc.ExistingProjects {
				kubermaticObj = append(kubermaticObj, existingProject)
			}
			for _, existingUser := range tc.ExistingKubermaticUsers {
				kubermaticObj = append(kubermaticObj, runtime.Object(existingUser))
			}

			ep, clients, err := createTestEndpointAndGetClients(tc.ExistingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponse)

			kubermaticFakeClient := clients.fakeKubermaticClient
			{
				actions := kubermaticFakeClient.Actions()
				if len(actions) != tc.ExpectedActions {
					t.Fatalf("expected to get %d actions but got %d from fake kubermatic client, actions = %v", tc.ExpectedActions, len(actions), actions)
				}

				if tc.ExpectedUserAfterInvitation != nil {
					updateAction, ok := actions[len(actions)-1].(clienttesting.CreateAction)
					if !ok {

					}
					updatedUser := updateAction.GetObject().(*kubermaticapiv1.User)
					if !equality.Semantic.DeepEqual(updatedUser, tc.ExpectedUserAfterInvitation) {
						t.Fatalf("%v", diff.ObjectDiff(updatedUser, tc.ExpectedUserAfterInvitation))
					}
				}
			}
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	tester := apiv1.User{
		Name:  testUserName,
		ID:    testUserID,
		Email: testUserEmail,
	}

	testcases := []struct {
		Name                    string
		ExpectedResponse        string
		ExpectedStatus          int
		ExistingProjects        []*kubermaticapiv1.Project
		ExistingKubermaticUsers []*kubermaticapiv1.User
		ExistingAPIUser         apiv1.User
	}{
		{
			Name: "scenario 1: a user with no projects yet should omit the `projects` key from the response",
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},
			},
			ExistingAPIUser:  tester,
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: `{"id":"john","name":"user1","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}`,
		},

		{
			Name: "scenario 2: a user who is assigned to projects",
			ExistingProjects: []*kubermaticapiv1.Project{
				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "moby",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "Joe",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-first-project"},
				},
				&kubermaticapiv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "plan9",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.io/v1",
								Kind:       "User",
								UID:        "",
								Name:       "John",
							},
						},
					},
					Spec: kubermaticapiv1.ProjectSpec{Name: "my-second-project"},
				},
			},
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-plan9",
								Name:  "plan9",
							},
							{
								Group: "editors-myThirdProjectInternalName",
								Name:  "myThirdProjectInternalName",
							},
						},
					},
				},
			},
			ExistingAPIUser:  tester,
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: `{"id":"john","name":"user1","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com","projects":[{"id":"plan9","group":"owners-plan9"},{"id":"myThirdProjectInternalName","group":"editors-myThirdProjectInternalName"}]}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			kubermaticObj := []runtime.Object{}
			for _, existingProject := range tc.ExistingProjects {
				kubermaticObj = append(kubermaticObj, existingProject)
			}
			for _, existingUser := range tc.ExistingKubermaticUsers {
				kubermaticObj = append(kubermaticObj, runtime.Object(existingUser))
			}

			ep, _, err := createTestEndpointAndGetClients(tc.ExistingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			req := httptest.NewRequest("GET", "/api/v1/me", nil)
			res := httptest.NewRecorder()
			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestNewUser(t *testing.T) {
	expectedResponse := `{"id":"","name":"user1","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}`
	apiUser := getUser(testUserEmail, testUserID, testUserName, false)

	expectedKubermaticUser := apiUserToKubermaticUser(apiUser)
	expectedKubermaticUser.GenerateName = "user-"

	kubermaticObj := []runtime.Object{}

	ep, clientSet, err := createTestEndpointAndGetClients(apiUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	res := httptest.NewRecorder()
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected HTTP status code %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	compareWithResult(t, res, expectedResponse)

	actions := clientSet.fakeKubermaticClient.Actions()
	if len(actions) != 10 {
		t.Fatalf("expected to get exactly 10 action but got %d, actions = %v", len(actions), actions)
	}

	action := actions[9]
	if !action.Matches("create", "users") {
		t.Fatalf("unexpected action %#v", action)
	}
	createAction, ok := action.(clienttesting.CreateAction)
	if !ok {
		t.Fatalf("unexpected action %#v", action)
	}
	if !equality.Semantic.DeepEqual(createAction.GetObject().(*kubermaticapiv1.User), expectedKubermaticUser) {
		t.Fatalf("%v", diff.ObjectDiff(expectedKubermaticUser, createAction.GetObject().(*kubermaticapiv1.User)))
	}
}

func TestCreateUserWithoutEmail(t *testing.T) {
	expectedResponse := `{"error":{"code":400,"message":"Email, ID and Name cannot be empty when creating a new user resource"}}`
	apiUser := getUser("", "", "", false)

	ep, _, err := createTestEndpointAndGetClients(apiUser, nil, []runtime.Object{}, []runtime.Object{}, []runtime.Object{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	res := httptest.NewRecorder()
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("Expected HTTP status code %d, got %d: %s", http.StatusBadRequest, res.Code, res.Body.String())
	}
	compareWithResult(t, res, expectedResponse)
}
