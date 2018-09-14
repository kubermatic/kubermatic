package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestGetUsersForProject(t *testing.T) {
	const longForm = "Jan 2, 2006 at 3:04pm (MST)"
	testcases := []struct {
		Name                        string
		ExpectedResponse            []apiv1.NewUser
		ExpectedResponseString      string
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
			ExpectedResponse: []apiv1.NewUser{
				apiv1.NewUser{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:   "john",
						Name: "john",
						CreationTimestamp: func() time.Time {
							creationTime, err := time.Parse(longForm, "Jan 1, 0001 at 0:00am (PST)")
							if err != nil {
								t.Fatal(err)
							}
							return creationTime
						}(),
					},
					Email: "john@acme.com",
					Projects: []apiv1.ProjectGroup{
						apiv1.ProjectGroup{
							GroupPrefix: "owners",
							ID:          "fooInternalName",
						},
					},
				},

				apiv1.NewUser{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:   "alice",
						Name: "Alice",
						CreationTimestamp: func() time.Time {
							creationTime, err := time.Parse(longForm, "Jan 1, 0001 at 0:00am (PST)")
							if err != nil {
								t.Fatal(err)
							}
							return creationTime
						}(),
					},
					Email: "alice@acme.com",
					Projects: []apiv1.ProjectGroup{
						apiv1.ProjectGroup{
							GroupPrefix: "viewers",
							ID:          "fooInternalName",
						},
					},
				},

				apiv1.NewUser{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:   "bob",
						Name: "Bob",
						CreationTimestamp: func() time.Time {
							creationTime, err := time.Parse(longForm, "Jan 1, 0001 at 0:00am (PST)")
							if err != nil {
								t.Fatal(err)
							}
							return creationTime
						}(),
					},
					Email: "bob@acme.com",
					Projects: []apiv1.ProjectGroup{
						apiv1.ProjectGroup{
							GroupPrefix: "editors",
							ID:          "fooInternalName",
						},
					},
				},
			},
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
			ExpectedResponseString: `{"error":{"code":403,"message":"forbidden: The user \"alice2@acme.com\" doesn't belong to the given project = foo2InternalName"}}`,
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

			if len(tc.ExpectedResponse) > 0 {
				usersFromResponse := []apiv1.NewUser{}
				rawBody, err := ioutil.ReadAll(res.Body)
				if err != nil {
					t.Fatal(err)
				}
				err = json.Unmarshal(rawBody, &usersFromResponse)
				if err != nil {
					t.Fatal(err)
				}
				if len(usersFromResponse) != len(tc.ExpectedResponse) {
					t.Fatalf("expected to get %d keys but got %d", len(tc.ExpectedResponse), len(usersFromResponse))
				}

				for _, expectedUser := range tc.ExpectedResponse {
					found := false
					for _, actualUser := range usersFromResponse {
						if actualUser.ID == expectedUser.ID {
							if !areEqualOrDie(t, actualUser, expectedUser) {
								t.Fatalf("actual user != expected user, diff = %v", diff.ObjectDiff(actualUser, expectedUser))
							}
							found = true
						}
					}
					if !found {
						t.Fatalf("the user with the name = %s was not found in the returned output", expectedUser.Name)
					}
				}
			}

			if len(tc.ExpectedResponseString) > 0 {
				compareWithResult(t, res, tc.ExpectedResponseString)
			}
		})
	}

}

func TestDeleteUserFromProject(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                         string
		Body                         string
		ExpectedResponse             string
		ExpectedBindingIDAfterDelete string
		ProjectToSync                string
		UserIDToDelete               string
		HTTPStatus                   int
		ExistingProjects             []*kubermaticapiv1.Project
		ExistingKubermaticUsers      []*kubermaticapiv1.User
		ExistingAPIUser              apiv1.User
		ExistingMembersBindings      []*kubermaticapiv1.UserProjectBinding
	}{
		// scenario 1
		{
			Name:          "scenario 1: john the owner of the plan9 project removes bob from the project",
			Body:          `{"id":"bobID", "email":"bob@acme.com", "projects":[{"id":"plan9", "group":"editors"}]}`,
			HTTPStatus:    http.StatusOK,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
			},
			UserIDToDelete: "bobID",
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{}`,
			ExistingMembersBindings: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-plan9",
						ProjectID: "plan9",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobPlanXBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "planX",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-planX",
						ProjectID: "planX",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnBidning",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: testUserEmail,
						Group:     "owners-plan9",
						ProjectID: "plan9",
					},
				},
			},
			ExpectedBindingIDAfterDelete: "bobBindings",
		},

		// scenario 2
		{
			Name:          "scenario 2: john the owner of the plan9 project removes bob, but bob is not a member of the project",
			Body:          `{"id":"bobID", "email":"bob@acme.com", "projects":[{"id":"plan9", "group":"editors"}]}`,
			HTTPStatus:    http.StatusBadRequest,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
			},
			UserIDToDelete: "bobID",
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":400,"message":"cannot delete the user = bob@acme.com from the project plan9 because the user is not a member of the project"}}`,
			ExistingMembersBindings: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobPlanXBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "planX",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-planX",
						ProjectID: "planX",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnBidning",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: testUserEmail,
						Group:     "owners-plan9",
						ProjectID: "plan9",
					},
				},
			},
		},

		// scenario 3
		{
			Name:          "scenario 3: john the owner of the plan9 project removes himself from the projec",
			Body:          fmt.Sprintf(`{"id":"%s", "email":"%s", "projects":[{"id":"plan9", "group":"owners"}]}`, testUserID, testUserEmail),
			HTTPStatus:    http.StatusForbidden,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
			},
			UserIDToDelete: testUserID,
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: testUserID,
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"you cannot delete yourself from the project"}}`,
			ExistingMembersBindings: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobPlanXBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "planX",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-planX",
						ProjectID: "planX",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnBidning",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: testUserEmail,
						Group:     "owners-plan9",
						ProjectID: "plan9",
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/users/%s", tc.ProjectToSync, tc.UserIDToDelete), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			for _, existingProject := range tc.ExistingProjects {
				kubermaticObj = append(kubermaticObj, existingProject)
			}
			for _, existingUser := range tc.ExistingKubermaticUsers {
				kubermaticObj = append(kubermaticObj, runtime.Object(existingUser))
			}
			for _, existingMember := range tc.ExistingMembersBindings {
				kubermaticObj = append(kubermaticObj, existingMember)
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
				if len(tc.ExpectedBindingIDAfterDelete) > 0 {
					actionWasValidated := false
					for _, action := range kubermaticFakeClient.Actions() {
						if action.Matches("delete", "userprojectbindings") {
							deleteAction, ok := action.(clienttesting.DeleteAction)
							if !ok {
								t.Fatalf("unexpected action %#v", action)
							}
							if deleteAction.GetName() != tc.ExpectedBindingIDAfterDelete {
								t.Fatalf("wrong binding removed, wanted = %s, actual = %s", tc.ExpectedBindingIDAfterDelete, deleteAction.GetName())
							}
							actionWasValidated = true
							break
						}
					}
					if !actionWasValidated {
						t.Fatal("create action was not validated, a binding for a user was not updated ?")
					}
				}
			}
		})
	}
}

func TestEditUserInProject(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                       string
		Body                       string
		ExpectedResponse           string
		ExpectedBindingAfterUpdate *kubermaticapiv1.UserProjectBinding
		ProjectToSync              string
		UserIDToUpdate             string
		HTTPStatus                 int
		ExistingProjects           []*kubermaticapiv1.Project
		ExistingKubermaticUsers    []*kubermaticapiv1.User
		ExistingAPIUser            apiv1.User
		ExistingMembersBindings    []*kubermaticapiv1.UserProjectBinding
	}{
		// scenario 1
		{
			Name:          "scenario 1: john the owner of the plan9 project changes the group for bob from viewers to editors",
			Body:          `{"id":"bobID", "email":"bob@acme.com", "projects":[{"id":"plan9", "group":"editors"}]}`,
			HTTPStatus:    http.StatusOK,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
			},
			UserIDToUpdate: "bobID",
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"id":"bobID","name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com","projects":[{"id":"plan9","group":"editors"}]}`,
			ExistingMembersBindings: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-plan9",
						ProjectID: "plan9",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobPlanXBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "planX",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-planX",
						ProjectID: "planX",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnBidning",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: testUserEmail,
						Group:     "owners-plan9",
						ProjectID: "plan9",
					},
				},
			},
			ExpectedBindingAfterUpdate: &kubermaticapiv1.UserProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bobBindings",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
							Kind:       kubermaticapiv1.ProjectKindName,
							Name:       "plan9",
						},
					},
				},
				Spec: kubermaticapiv1.UserProjectBindingSpec{
					UserEmail: "bob@acme.com",
					Group:     "editors-plan9",
					ProjectID: "plan9",
				},
			},
		},

		// scenario 2
		{
			Name:          "scenario 2: john the owner of the plan9 project changes the group for bob, but bob is not a member of the project",
			Body:          `{"id":"bobID", "email":"bob@acme.com", "projects":[{"id":"plan9", "group":"editors"}]}`,
			HTTPStatus:    http.StatusBadRequest,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
			},
			UserIDToUpdate: "bobID",
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":400,"message":"cannot change the membership of the user = bob@acme.com for the project plan9 because the user is not a member of the project"}}`,
			ExistingMembersBindings: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobPlanXBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "planX",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-planX",
						ProjectID: "planX",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnBidning",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: testUserEmail,
						Group:     "owners-plan9",
						ProjectID: "plan9",
					},
				},
			},
		},

		// scenario 3
		{
			Name:          "scenario 3: john the owner of the plan9 project changes the group for bob from viewers to owners",
			Body:          `{"id":"bobID", "email":"bob@acme.com", "projects":[{"id":"plan9", "group":"owners"}]}`,
			HTTPStatus:    http.StatusForbidden,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
			},
			UserIDToUpdate: "bobID",
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"the given user cannot be assigned to owners group"}}`,
			ExistingMembersBindings: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-plan9",
						ProjectID: "plan9",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobPlanXBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "planX",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-planX",
						ProjectID: "planX",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnBidning",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: testUserEmail,
						Group:     "owners-plan9",
						ProjectID: "plan9",
					},
				},
			},
		},

		// scenario 4
		{
			Name:          "scenario 4: john the owner of the plan9 project changes the group for bob from viewers to admins(wrong name)",
			Body:          `{"id":"bobID", "email":"bob@acme.com", "projects":[{"id":"plan9", "group":"admins"}]}`,
			HTTPStatus:    http.StatusBadRequest,
			ProjectToSync: "plan9",
			ExistingProjects: []*kubermaticapiv1.Project{
				createTestProject("my-first-project", kubermaticapiv1.ProjectActive),
				createTestProject("my-third-project", kubermaticapiv1.ProjectActive),
				plan9,
			},
			UserIDToUpdate: "bobID",
			ExistingKubermaticUsers: []*kubermaticapiv1.User{
				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  testUserName,
						ID:    testUserID,
						Email: testUserEmail,
					},
				},

				&kubermaticapiv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobID",
					},
					Spec: kubermaticapiv1.UserSpec{
						Name:  "Bob",
						Email: "bob@acme.com",
					},
				},
			},
			ExistingAPIUser: apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
			ExpectedResponse: `{"error":{"code":400,"message":"invalid group name admins"}}`,
			ExistingMembersBindings: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-plan9",
						ProjectID: "plan9",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bobPlanXBindings",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "planX",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						Group:     "viewers-planX",
						ProjectID: "planX",
					},
				},

				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "johnBidning",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: testUserEmail,
						Group:     "owners-plan9",
						ProjectID: "plan9",
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/users/%s", tc.ProjectToSync, tc.UserIDToUpdate), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			for _, existingProject := range tc.ExistingProjects {
				kubermaticObj = append(kubermaticObj, existingProject)
			}
			for _, existingUser := range tc.ExistingKubermaticUsers {
				kubermaticObj = append(kubermaticObj, runtime.Object(existingUser))
			}
			for _, existingMember := range tc.ExistingMembersBindings {
				kubermaticObj = append(kubermaticObj, existingMember)
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
				if tc.ExpectedBindingAfterUpdate != nil {
					actionWasValidated := false
					for _, action := range kubermaticFakeClient.Actions() {
						if action.Matches("update", "userprojectbindings") {
							updateAction, ok := action.(clienttesting.UpdateAction)
							if !ok {
								t.Fatalf("unexpected action %#v", action)
							}
							updatedBinding := updateAction.GetObject().(*kubermaticapiv1.UserProjectBinding)
							if !equality.Semantic.DeepEqual(updatedBinding, tc.ExpectedBindingAfterUpdate) {
								t.Fatalf("updated action mismatch %v", diff.ObjectDiff(updatedBinding, tc.ExpectedBindingAfterUpdate))
							}
							actionWasValidated = true
							break
						}
					}
					if !actionWasValidated {
						t.Fatal("create action was not validated, a binding for a user was not updated ?")
					}
				}
			}
		})
	}
}

func TestAddUserToProject(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                           string
		Body                           string
		ExpectedResponse               string
		ExpectedBindingAfterInvitation *kubermaticapiv1.UserProjectBinding
		ProjectToSync                  string
		HTTPStatus                     int
		ExistingProjects               []*kubermaticapiv1.Project
		ExistingKubermaticUsers        []*kubermaticapiv1.User
		ExistingAPIUser                apiv1.User
		ExistingMembers                []*kubermaticapiv1.UserProjectBinding
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
			ExpectedResponse: `{"id":"bob","name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com","projects":[{"id":"plan9","group":"editors"}]}`,
			ExpectedBindingAfterInvitation: &kubermaticapiv1.UserProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
							Kind:       kubermaticapiv1.ProjectKindName,
							Name:       "plan9",
						},
					},
				},
				Spec: kubermaticapiv1.UserProjectBindingSpec{
					UserEmail: "bob@acme.com",
					Group:     "editors-plan9",
					ProjectID: "plan9",
				},
			},
		},

		{
			Name:          "scenario 2: john the owner of the plan9 project tries to invite bob to another project",
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
		},

		{
			Name:          "scenario 3: john the owner of the plan9 project tries to invite  himself to another group",
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
		},

		{
			Name:          "scenario 4: john the owner of the plan9 project tries to invite bob to the project as an owner",
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
		},

		{
			Name:          "scenario 5: john the owner of the plan9 project invites bob to the project second time",
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
			ExistingMembers: []*kubermaticapiv1.UserProjectBinding{
				&kubermaticapiv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
								Kind:       kubermaticapiv1.ProjectKindName,
								Name:       "plan9",
							},
						},
					},
					Spec: kubermaticapiv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						ProjectID: "plan9",
						Group:     "editors-plan9",
					},
				},
			},
			ExpectedResponse: `{"error":{"code":400,"message":"cannot add the user = bob@acme.com to the project plan9 because user is already in the project"}}`,
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
			for _, existingMember := range tc.ExistingMembers {
				kubermaticObj = append(kubermaticObj, existingMember)
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
				if tc.ExpectedBindingAfterInvitation != nil {
					actionWasValidated := false
					for _, action := range kubermaticFakeClient.Actions() {
						if action.Matches("create", "userprojectbindings") {
							updateAction, ok := action.(clienttesting.CreateAction)
							if !ok {
								t.Fatalf("unexpected action %#v", action)
							}
							createdBinding := updateAction.GetObject().(*kubermaticapiv1.UserProjectBinding)
							// Name was generated by the test framework, just rewrite it
							tc.ExpectedBindingAfterInvitation.Name = createdBinding.Name
							if !equality.Semantic.DeepEqual(createdBinding, tc.ExpectedBindingAfterInvitation) {
								t.Fatalf("%v", diff.ObjectDiff(createdBinding, tc.ExpectedBindingAfterInvitation))
							}
							actionWasValidated = true
							break
						}
					}
					if !actionWasValidated {
						t.Fatal("create action was not validated, a binding for a user was not created ?")
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
			ExpectedResponse: `{"id":"john","name":"user1","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com","projects":[{"id":"plan9","group":"owners"},{"id":"myThirdProjectInternalName","group":"editors"}]}`,
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
	if len(actions) != 12 {
		t.Fatalf("expected to get exactly 10 action but got %d, actions = %v", len(actions), actions)
	}

	action := actions[11]
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
