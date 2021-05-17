/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package project_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticfakeclentset "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/fake"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/handler/v1/project"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakerestclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           apiv1.User
	}{
		{
			Name:            "scenario 1: rename existing project",
			Body:            `{"Name": "Super-Project"}`,
			HTTPStatus:      http.StatusOK,
			ProjectToRename: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
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
			HTTPStatus:      http.StatusOK,
			ProjectToRename: "my-first-project-ID",
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
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
			ExpectedResponse: `{"id":"my-first-project-ID","name":"my-second-project","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
		},
		{
			Name:            "scenario 3: rename existing project with existing name where user is not the owner",
			Body:            `{"Name": "my-second-project"}`,
			HTTPStatus:      http.StatusOK,
			ProjectToRename: "my-first-project-ID",
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
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
			HTTPStatus:      http.StatusNotFound,
			ProjectToRename: "some-ID",
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
			},
			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: `{"error":{"code":404,"message":"projects.kubermatic.k8s.io \"some-ID\" not found"}}`,
		},
		{
			Name:            "scenario 5: rename a project with empty name",
			Body:            `{"Name": ""}`,
			HTTPStatus:      http.StatusBadRequest,
			ProjectToRename: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
			},
			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: `{"error":{"code":400,"message":"the name of the project cannot be empty"}}`,
		},
		{
			Name:             "scenario 6: the admin Bob can update John's project",
			Body:             `{"Name": "Super-Project"}`,
			ProjectToRename:  "my-first-project-ID",
			ExpectedResponse: `{"id":"my-first-project-ID","name":"Super-Project","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"John","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: *test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 7: the user John can't update Bob's project",
			Body:             `{"Name": "Super-Project"}`,
			ProjectToRename:  "my-third-project-ID",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-third-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: *test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToRename), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
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
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		DisplayAll                bool
	}{
		{
			Name:       "scenario 1: list projects that John is the member of",
			Body:       ``,
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
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
			ExistingAPIUser: func() *apiv1.User {
				apiUser := test.GenDefaultAPIUser()
				apiUser.Email = "john@acme.com"
				return apiUser
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
		{
			Name:       "scenario 2: list all projects for the admin user",
			Body:       ``,
			DisplayAll: true,
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-second-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(time.Minute)),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-second-project-ID", "john@acme.com", "editors"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
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
						ID:                "my-second-project-ID",
						Name:              "my-second-project",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 55, 0, 0, time.UTC),
					},
				},
				{
					Status: "Active",
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "my-third-project-ID",
						Name:              "my-third-project",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 56, 0, 0, time.UTC),
					},
					Owners: []apiv1.User{
						{
							ObjectMeta: apiv1.ObjectMeta{
								Name: "Bob",
							},
							Email: "bob@acme.com",
						},
					},
				},
			},
		},
		{
			Name:       "scenario 3: regular user can not list all projects",
			Body:       ``,
			DisplayAll: true,
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-second-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(time.Minute)),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", false),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-second-project-ID", "john@acme.com", "editors"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ExpectedResponse: []apiv1.Project{
				{
					Status: "Active",
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "my-third-project-ID",
						Name:              "my-third-project",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 56, 0, 0, time.UTC),
					},
					Owners: []apiv1.User{
						{
							ObjectMeta: apiv1.ObjectMeta{
								Name: "Bob",
							},
							Email: "bob@acme.com",
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects?displayAll=%v", tc.DisplayAll), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualProjects := test.ProjectV1SliceWrapper{}
			actualProjects.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedProjects := test.ProjectV1SliceWrapper(tc.ExpectedResponse)
			wrappedExpectedProjects.Sort()

			actualProjects.EqualOrDie(wrappedExpectedProjects, t)
		})
	}
}

func TestListProjectMethod(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *kubermaticapiv1.User
		ExpectedErrorMsg          string
		ExpectedDetails           []string
		ExpectedResponse          []apiv1.Project
	}{
		{
			Name: "scenario 1: project doesn't exist and it's forbidden for impersonated client, skipped in the result list",
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject(test.NoExistingFakeProject, kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-second-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(time.Minute)),
				test.GenProject(test.ExistingFakeProject, kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				// make John the owner of the first project and the editor of the second
				test.GenBinding(test.NoExistingFakeProjectID, "john@acme.com", "owners"),
				test.GenBinding(test.ExistingFakeProjectID, "john@acme.com", "editors"),
			},
			ExistingAPIUser: test.GenUser("JohnID", "John", "john@acme.com"),
			ExpectedResponse: []apiv1.Project{
				{
					Status: "Active",
					ObjectMeta: apiv1.ObjectMeta{
						ID:                test.ExistingFakeProjectID,
						Name:              test.ExistingFakeProject,
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 56, 0, 0, time.UTC),
					},
					Owners: []apiv1.User{},
				},
			},
		},
		{
			Name: "scenario 2: two project providers return 404 error code, the first error is added to the final error details list",
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject(test.ForbiddenFakeProject, kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject(test.ExistingFakeProject, kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				// make John the owner of the first project and the editor of the second
				test.GenBinding(test.ForbiddenFakeProjectID, "john@acme.com", "owners"),
				test.GenBinding(test.ExistingFakeProjectID, "john@acme.com", "editors"),
			},
			ExistingAPIUser:  test.GenUser("JohnID", "John", "john@acme.com"),
			ExpectedErrorMsg: "failed to get some projects, please examine details field for more info",
			ExpectedDetails:  []string{test.ImpersonatedClientErrorMsg},
			ExpectedResponse: []apiv1.Project{
				{
					Status: "Active",
					ObjectMeta: apiv1.ObjectMeta{
						ID:                test.ExistingFakeProjectID,
						Name:              test.ExistingFakeProject,
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 56, 0, 0, time.UTC),
					},
					Owners: []apiv1.User{},
				},
			},
		},
	}

	versions := kubermatic.NewDefaultVersions()

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			testSeed := test.GenTestSeed()
			tc.ExistingKubermaticObjects = append(tc.ExistingKubermaticObjects, testSeed)

			kubermaticClient := kubermaticfakeclentset.NewSimpleClientset()
			fakeClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.ExistingKubermaticObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			projectMemberProvider := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, fakeClient, kubernetes.IsProjectServiceAccount)
			userProvider := kubernetes.NewUserProvider(fakeClient, kubernetes.IsProjectServiceAccount, kubermaticClient)

			userInfoGetter, err := provider.UserInfoGetterFactory(projectMemberProvider)
			if err != nil {
				t.Fatal(err)
			}

			fUserClusterConnection := &fakeUserClusterConnection{fakeClient}
			kubernetesClient := fakerestclient.NewSimpleClientset()
			clusterProvider := kubernetes.NewClusterProvider(
				&restclient.Config{},
				fakeImpersonationClient,
				fUserClusterConnection,
				"",
				rbac.ExtractGroupPrefix,
				fakeClient,
				kubernetesClient,
				false,
				versions,
			)
			clusterProviders := map[string]provider.ClusterProvider{testSeed.Name: clusterProvider}
			clusterProviderGetter := func(seed *kubermaticapiv1.Seed) (provider.ClusterProvider, error) {
				if clusterProvider, exists := clusterProviders[seed.Name]; exists {
					return clusterProvider, nil
				}
				return nil, fmt.Errorf("cannot find clusterprovider for cluster %q", seed.Name)
			}

			endpointFun := project.ListEndpoint(userInfoGetter, test.NewFakeProjectProvider(), test.NewFakePrivilegedProjectProvider(), projectMemberProvider, projectMemberProvider, userProvider, clusterProviderGetter, test.CreateTestSeedsGetter(ctx, fakeClient))

			ctx = context.WithValue(ctx, middleware.UserCRContextKey, tc.ExistingAPIUser)

			projectsRaw, err := endpointFun(ctx, project.ListReq{})
			resultProjectList := make([]apiv1.Project, 0)

			if len(tc.ExpectedErrorMsg) > 0 {
				if err == nil {
					t.Fatal("expected error")
				}
				kubermaticError, ok := err.(errors.HTTPError)
				if !ok {
					t.Fatal("expected HTTPError")
				}
				if kubermaticError.Error() != tc.ExpectedErrorMsg {
					t.Fatalf("expected error message %s got %s", tc.ExpectedErrorMsg, kubermaticError.Error())
				}
				if !equality.Semantic.DeepEqual(kubermaticError.Details(), tc.ExpectedDetails) {
					t.Fatalf("expected error details %v got %v", tc.ExpectedDetails, kubermaticError.Details())
				}
			} else {
				if projectsRaw == nil {
					t.Fatal("project endpoint can not be nil")
				}

				for _, project := range projectsRaw.([]*apiv1.Project) {
					resultProjectList = append(resultProjectList, *project)
				}

				actualProjects := test.ProjectV1SliceWrapper(resultProjectList)
				actualProjects.Sort()

				wrappedExpectedProjects := test.ProjectV1SliceWrapper(tc.ExpectedResponse)
				wrappedExpectedProjects.Sort()

				actualProjects.EqualOrDie(wrappedExpectedProjects, t)
			}
		})
	}
}

type fakeUserClusterConnection struct {
	fakeDynamicClient ctrlruntimeclient.Client
}

func (f *fakeUserClusterConnection) GetClient(_ context.Context, _ *kubermaticapiv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.fakeDynamicClient, nil
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
		ExistingKubermaticObjects []ctrlruntimeclient.Object
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
		{
			Name:             "scenario 2: the admin Bob can get John's project",
			Body:             ``,
			ProjectToSync:    "my-first-project-ID",
			ExpectedResponse: `{"id":"my-first-project-ID","name":"my-first-project","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"John","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 3: the user John can't get Bob's project",
			Body:             ``,
			ProjectToSync:    "my-third-project-ID",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-third-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
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
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
	}{
		{
			Name:             "scenario 1: a user doesn't have any projects, thus creating one succeeds",
			Body:             `{"name":"my-first-project"}`,
			RewriteProjectID: true,
			ExpectedResponse: `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
			HTTPStatus:       http.StatusCreated,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultUser(),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},

		{
			Name:                      "scenario 2: having more than one project with the same name is allowed",
			Body:                      fmt.Sprintf(`{"name":"%s"}`, test.GenDefaultProject().Spec.Name),
			RewriteProjectID:          true,
			ExpectedResponse:          `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
			HTTPStatus:                http.StatusCreated,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
		},

		{
			Name:             "scenario 3: user reached maximum number of projects",
			Body:             fmt.Sprintf(`{"name":"%s"}`, test.GenDefaultProject().Spec.Name),
			RewriteProjectID: false,
			ExpectedResponse: `{"error":{"code":403,"message":"reached maximum number of projects"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticapiv1.KubermaticSetting {
					settings := test.GenDefaultSettings()
					settings.Spec.UserProjectsLimit = 1
					return settings
				}(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},

		{
			Name:             "scenario 4: user has not owned project and doesn't reach maximum number of projects",
			Body:             fmt.Sprintf(`{"name":"%s"}`, test.GenDefaultProject().Spec.Name),
			RewriteProjectID: true,
			ExpectedResponse: `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
			HTTPStatus:       http.StatusCreated,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),

				test.GenBinding("my-first-project-ID", "john@acme.com", "editors"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "viewers"),
				func() *kubermaticapiv1.KubermaticSetting {
					settings := test.GenDefaultSettings()
					settings.Spec.UserProjectsLimit = 1
					return settings
				}(),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},

		{
			Name:             "scenario 5: project creation is restricted for the users",
			Body:             fmt.Sprintf(`{"name":"%s"}`, test.GenDefaultProject().Spec.Name),
			RewriteProjectID: false,
			ExpectedResponse: `{"error":{"code":403,"message":"project creation is restricted"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticapiv1.KubermaticSetting {
					settings := test.GenDefaultSettings()
					settings.Spec.RestrictProjectCreation = true
					return settings
				}(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 6: project creation is not restricted for the admin",
			Body:             fmt.Sprintf(`{"name":"%s"}`, test.GenDefaultProject().Spec.Name),
			RewriteProjectID: true,
			ExpectedResponse: `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive","owners":[{"name":"Bob","creationTimestamp":"0001-01-01T00:00:00Z","email":"bob@acme.com"}]}`,
			HTTPStatus:       http.StatusCreated,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
				func() *kubermaticapiv1.KubermaticSetting {
					settings := test.GenDefaultSettings()
					settings.Spec.RestrictProjectCreation = true
					return settings
				}(),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
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
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
	}{
		{
			Name:                      "scenario 1: the owner of the project can delete the project",
			HTTPStatus:                http.StatusOK,
			ProjectToSync:             test.GenDefaultProject().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
		},
		{
			Name:          "scenario 2: the admin Bob can delete John's project",
			ProjectToSync: "my-first-project-ID",
			HTTPStatus:    http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:          "scenario 3: the user John can't delete Bob's project",
			ProjectToSync: "my-third-project-ID",
			HTTPStatus:    http.StatusForbidden,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				// add some projects
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp().Add(2*time.Minute)),
				// add John
				test.GenUser("JohnID", "John", "john@acme.com"),
				test.GenAdminUser("Bob", "bob@acme.com", true),
				// make John the owner of the first project and the editor of the second
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "bob@acme.com", "owners"),
			},
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			// test data
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", tc.ProjectToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
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

func TestServiceAccountProjectAccess(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                          string
		existingKubermaticObjs        []ctrlruntimeclient.Object
		existingSa                    *kubermaticapiv1.User
		bodyCreate                    string
		expectedGetProjectResponse    string
		expectedCreateProjectResponse string
		projectToSync                 string
		httpGetStatus                 int
		httpCreateStatus              int
		rewriteProjectID              bool
	}{
		{
			name:             "scenario 1: service account in editors group can't create project",
			httpGetStatus:    http.StatusOK,
			httpCreateStatus: http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
			},
			existingSa:                    test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
			projectToSync:                 "plan9-ID",
			bodyCreate:                    `{"name":"my-first-project"}`,
			expectedGetProjectResponse:    `{"id":"plan9-ID","name":"plan9","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
			expectedCreateProjectResponse: `{"error":{"code":403,"message":"the Service Account is not allowed to create a project"}}`,
		},
		{
			name:             "scenario 2: service account in viewers group can't create project",
			httpGetStatus:    http.StatusOK,
			httpCreateStatus: http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "viewers", "plan9-ID"),
			},
			existingSa:                    test.GenProjectServiceAccount("1", "test-1", "viewers", "plan9-ID"),
			projectToSync:                 "plan9-ID",
			bodyCreate:                    `{"name":"my-first-project"}`,
			expectedGetProjectResponse:    `{"id":"plan9-ID","name":"plan9","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
			expectedCreateProjectResponse: `{"error":{"code":403,"message":"the Service Account is not allowed to create a project"}}`,
		},
		{
			name:             "scenario 3: service account in projectmanagers group can create project",
			httpGetStatus:    http.StatusOK,
			httpCreateStatus: http.StatusCreated,
			rewriteProjectID: true,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "projectmanagers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "projectmanagers", "plan9-ID"),
			},
			existingSa:                    test.GenProjectServiceAccount("1", "test-1", "projectmanagers", "plan9-ID"),
			projectToSync:                 "plan9-ID",
			bodyCreate:                    `{"name":"my-first-project","users":["john@acme.com"]}`,
			expectedGetProjectResponse:    `{"id":"plan9-ID","name":"plan9","creationTimestamp":"2013-02-03T19:54:00Z","status":"Active","owners":[{"name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
			expectedCreateProjectResponse: `{"id":"%s","name":"my-first-project","creationTimestamp":"0001-01-01T00:00:00Z","status":"Inactive","owners":[{"name":"john","creationTimestamp":"0001-01-01T00:00:00Z","email":"john@acme.com"}]}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// set up
			apiSA := apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					Name: tc.existingSa.Name,
				},
				Email: tc.existingSa.Spec.Email,
			}

			tokenSecret, err := genToken(tc.existingSa, tc.projectToSync, "fake")
			if err != nil {
				t.Fatalf("can not generate token data %v", err)
			}
			token := tokenSecret.Data["token"]

			// act 1 - get the project using sa token
			getEp, err := test.CreateTestEndpoint(apiSA, []ctrlruntimeclient.Object{tokenSecret}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s", tc.projectToSync), strings.NewReader(""))
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
			res := httptest.NewRecorder()
			getEp.ServeHTTP(res, req)

			// validate
			if res.Code != tc.httpGetStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpGetStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.expectedGetProjectResponse)

			// act 2 - create a new project using sa token
			req = httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(tc.bodyCreate))
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
			res = httptest.NewRecorder()
			getEp.ServeHTTP(res, req)

			// validate
			if res.Code != tc.httpCreateStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpCreateStatus, res.Code, res.Body.String())
			}

			expectedResponse := tc.expectedCreateProjectResponse
			// since Project.ID is automatically generated by the system just rewrite it.
			if tc.rewriteProjectID {
				actualProject := &apiv1.Project{}
				err = json.Unmarshal(res.Body.Bytes(), actualProject)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.expectedCreateProjectResponse, actualProject.ID)
			}

			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}

func genToken(sa *kubermaticv1.User, projectID, tokenName string) (*v1.Secret, error) {

	tokenGenerator, err := serviceaccount.JWTTokenGenerator([]byte(test.TestServiceAccountHashKey))
	if err != nil {
		return nil, fmt.Errorf("can init token generator %v", err)
	}
	token, err := tokenGenerator.Generate(serviceaccount.Claims(sa.Spec.Email, projectID, tokenName))
	if err != nil {
		return nil, fmt.Errorf("can not generate token data %v", err)
	}

	secret := &v1.Secret{}
	secret.Name = fmt.Sprintf("%s%s", "sa-token-", tokenName)
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.UserKindName,
			UID:        sa.GetUID(),
			Name:       sa.Name,
		},
	}
	secret.Labels = map[string]string{
		kubermaticv1.ProjectIDLabelKey: projectID,
		"name":                         tokenName,
	}
	secret.Data = make(map[string][]byte)
	secret.Data["token"] = []byte(token)
	secret.Type = "Opaque"
	return secret, nil
}
