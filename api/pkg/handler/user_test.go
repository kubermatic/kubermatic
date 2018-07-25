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
			Body:          `{"email":"bob@acme.com", "projects":[{"name":"plan9", "group":"editors"}]}`,
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
						Name:  testUsername,
						Email: testEmail,
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
				ID:    testUsername,
				Email: testEmail,
			},
			ExpectedResponse: `{"id":"bob","name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com","projects":[{"name":"placeX","group":"editors-placeX"},{"name":"plan9","group":"editors-plan9"}]}`,
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
			Body:          `{"email":"bob@acme.com", "projects":[{"name":"plan9", "group":"editors"}]}`,
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
						Name:  testUsername,
						Email: testEmail,
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
				ID:    testUsername,
				Email: testEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"only the owner of the project can invite the other users"}}`,
			ExpectedActions:  9,
		},

		{
			Name:          "scenario 3: john the owner of the plan9 project tries to invite bob to another project",
			Body:          `{"email":"bob@acme.com", "projects":[{"name":"moby", "group":"editors"}]}`,
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
						Name:  testUsername,
						Email: testEmail,
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
				ID:    testUsername,
				Email: testEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"you can only assign the user to plan9 project"}}`,
			ExpectedActions:  8,
		},

		{
			Name:          "scenario 4: john the owner of the plan9 project tries to invite  himself to another group",
			Body:          fmt.Sprintf(`{"email":"%s", "projects":[{"name":"plan9", "group":"editors"}]}`, testEmail),
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
						Name:  testUsername,
						Email: testEmail,
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
				ID:    testUsername,
				Email: testEmail,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"you cannot assign yourself to a different group"}}`,
			ExpectedActions:  8,
		},

		{
			Name:          "scenario 5: john the owner of the plan9 project tries to invite bob to the project as an owner",
			Body:          `{"email":"bob@acme.com", "projects":[{"name":"plan9", "group":"owners"}]}`,
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
						Name:  testUsername,
						Email: testEmail,
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
				ID:    testUsername,
				Email: testEmail,
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
