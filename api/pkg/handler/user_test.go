package handler

import (
	"fmt"
	"log"
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

var plan9 = &kubermaticapiv1.Project{
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
	Status: kubermaticapiv1.ProjectStatus{
		Phase: kubermaticapiv1.ProjectActive,
	},
}

func TestDeleteUserFromProject(t *testing.T) {
	testcases := []struct {
		Name string
		// ExpectedUserAfterInvitation *kubermaticapiv1.User
		ExpectedResponse        string
		Project                 string
		UserToRemove            string
		HTTPStatus              int
		ExistingProjects        []*kubermaticapiv1.Project
		ExistingKubermaticUsers []*kubermaticapiv1.User
		ExistingAPIUser         apiv1.User
	}{
		{
			Name:             "scenario 1: project owner removes a member from the project",
			HTTPStatus:       http.StatusOK,
			ExpectedResponse: `[{"id":"john","name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com","projects":[{"id":"fooInternalName","group":"owners-fooInternalName"}]}]`,
			Project:          "fooInternalName",
			UserToRemove:     "2",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("foo", kubermaticapiv1.ProjectActive),
			},
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "john",
						ID:    "1",
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-fooInternalName",
								Name:  "fooInternalName",
							},
						},
					},
				},
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alice",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Alice",
						ID:    "2",
						Email: "alice@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-fooInternalName",
								Name:  "fooInternalName",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    "1",
				Name:  "john",
				Email: testUserEmail,
			},
		},

		{
			Name:             "scenario 2: project editor removes a member from the project",
			HTTPStatus:       http.StatusForbidden,
			Project:          "fooInternalName",
			ExpectedResponse: `[{"id":"john","name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com","projects":[{"id":"fooInternalName","group":"owners-fooInternalName"}]}]`,
			UserToRemove:     "1",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("foo", kubermaticapiv1.ProjectActive),
			},
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "john",
						ID:    "1",
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-fooInternalName",
								Name:  "fooInternalName",
							},
						},
					},
				},
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alice",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Alice",
						ID:    "2",
						Email: "alice@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-fooInternalName",
								Name:  "fooInternalName",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    "2",
				Name:  "alice",
				Email: "alice@acme.com",
			},
		},

		{
			Name:             "scenario 3: project owner removes a non exsisting member from the project",
			HTTPStatus:       http.StatusNotFound,
			Project:          "fooInternalName",
			UserToRemove:     "2",
			ExpectedResponse: `[{"id":"john","name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com","projects":[{"id":"fooInternalName","group":"owners-fooInternalName"}]}]`,
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("bar", kubermaticapiv1.ProjectActive),
				createTestProject("foo", kubermaticapiv1.ProjectActive),
			},
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "john",
						ID:    "1",
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-fooInternalName",
								Name:  "fooInternalName",
							},
						},
					},
				},
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alice",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Alice",
						ID:    "2",
						Email: "alice@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "editors-barInternalName",
								Name:  "barInternalName",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    "1",
				Name:  "john",
				Email: "john@acme.com",
			},
		},

		// {
		// 	Name:             "scenario 4: project viewer removes a member from the project",
		// 	HTTPStatus:       http.StatusForbidden,
		// 	Project:          "plan9",
		// 	ExistingProjects: []*kubermaticapiv1.Project{plan9},
		// },
		// {
		// 	Name:             "scenario 5: non project member removes a member from the project",
		// 	HTTPStatus:       http.StatusForbidden,
		// 	Project:          "plan9",
		// 	ExistingProjects: []*kubermaticapiv1.Project{plan9},
		// },
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/users/%s", tc.Project, tc.UserToRemove), nil)
			res := httptest.NewRecorder()
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

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d", tc.HTTPStatus, res.Code)
			}

			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/users", tc.Project), nil)
			res = httptest.NewRecorder()
			ep.ServeHTTP(res, req)
			log.Printf(">>> %s\n", res.Body)
		})
	}
}

func TestGetUsersForProject(t *testing.T) {
	testcases := []struct {
		Name                        string
		ExpectedResponse            string
		ExpectedActions             int
		ExpectedUserAfterInvitation *kubermaticapiv1.User
		ProjectToGet                string
		HTTPStatus                  int
		ExistingProjects            []*kubermaticapiv1.Project
		ExistingKubermaticUsers     []*kubermaticapiv1.User
		ExistingAPIUser             apiv1.User
	}{
		{
			Name:         "scenario 1: get a list of user for a project 'foo'",
			HTTPStatus:   http.StatusOK,
			ProjectToGet: "fooInternalName",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("foo", kubermaticapiv1.ProjectActive),
				createTestProject("bar", kubermaticapiv1.ProjectActive),
				createTestProject("zorg", kubermaticapiv1.ProjectActive),
			},
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "john",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "john",
						ID:    "12345",
						Email: testUserEmail,
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "owners-foo",
								Name:  "fooInternalName",
							},
							{
								Group: "editors-bar",
								Name:  "barInternalName",
							},
						},
					},
				},
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alice",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Alice",
						Email: "alice@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "viewers-foo",
								Name:  "fooInternalName",
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
								Group: "editors-zorg",
								Name:  "zorgInternalName",
							},
							{
								Group: "editors-foo",
								Name:  "fooInternalName",
							},
							{
								Group: "editors-bar",
								Name:  "barInternalName",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    "12345",
				Name:  "john",
				Email: testUserEmail,
			},
			ExpectedResponse: `[{"id":"john","name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com","projects":[{"id":"fooInternalName","group":"owners-foo"}]},{"id":"alice","name":"Alice","creationTimestamp":"0001-01-01T00:00:00Z","email":"alice@acme.com","projects":[{"id":"fooInternalName","group":"viewers-foo"}]},{"id":"bob","name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com","projects":[{"id":"fooInternalName","group":"editors-foo"}]}]`,
		},
		{
			Name:         "scenario 2: get a list of user for a project 'foo' for external user",
			HTTPStatus:   http.StatusForbidden,
			ProjectToGet: "foo2InternalName",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("foo2", kubermaticapiv1.ProjectActive),
				createTestProject("bar2", kubermaticapiv1.ProjectActive),
			},
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alice2",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Alice2",
						Email: "alice2@acme.com",
						Projects: []kubermaticapiv1.ProjectGroup{
							{
								Group: "viewers-bar",
								Name:  "barInternalName",
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
								Group: "editors-foo2",
								Name:  "foo2InternalName",
							},
							{
								Group: "editors-bar2",
								Name:  "bar2InternalName",
							},
						},
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				Name:  "alice2",
				Email: "alice2@acme.com",
			},
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: The user \"Alice2\" doesn't belong to the given project = foo2InternalName"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/users", tc.ProjectToGet), nil)
			res := httptest.NewRecorder()
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

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}

}

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
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
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
								Group: "editors-my-third-projectInternalName",
								Name:  "my-third-projectInternalName",
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
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
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
								Group: "editors-my-third-projectInternalName",
								Name:  "my-third-projectInternalName",
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
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
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
								Group: "editors-my-third-projectInternalName",
								Name:  "my-third-projectInternalName",
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
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
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
								Group: "editors-my-third-projectInternalName",
								Name:  "my-third-projectInternalName",
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
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
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
								Group: "editors-my-third-projectInternalName",
								Name:  "my-third-projectInternalName",
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

		{
			Name:          "scenario 6: john the owner of the plan9 project invites bob to the project second time",
			Body:          `{"email":"bob@acme.com", "projects":[{"id":"plan9", "group":"editors"}]}`,
			HTTPStatus:    http.StatusBadRequest,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
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
								Group: "editors-my-third-projectInternalName",
								Name:  "my-third-projectInternalName",
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
			ExpectedResponse: `{"error":{"code":400,"message":"cannot add the user = bob@acme.com to the project plan9 because user is already in the project"}}`,
			ExpectedActions:  9,
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
