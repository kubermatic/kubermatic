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

package serviceaccount_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	serviceaccount "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateServiceAccountProject(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		expectedResponse       string
		expectedGroup          string
		expectedSAName         string
		projectToSync          string
		httpStatus             int
		existingAPIUser        apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:       "scenario 1: create service account 'test' for editors group by project owner john",
			body:       `{"name":"test", "group":"editors"}`,
			httpStatus: http.StatusCreated,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:   "plan9-ID",
			expectedSAName:  "test",
			expectedGroup:   "editors-plan9-ID",
		},
		{
			name:       "scenario 2: check forbidden owner group",
			body:       `{"name":"test", "group":"owners"}`,
			httpStatus: http.StatusBadRequest,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "my-first-project-ID",
			expectedResponse: `{"error":{"code":400,"message":"invalid group name owners"}}`,
		},
		{
			name:       "scenario 3: check name, group, project ID validator",
			body:       `{"name":"test"}`,
			httpStatus: http.StatusBadRequest,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "my-first-project-ID",
			expectedResponse: `{"error":{"code":400,"message":"the name, project ID and group cannot be empty"}}`,
		},
		{
			name:       "scenario 4: check when given name is already reserved",
			body:       `{"name":"test", "group":"editors"}`,
			httpStatus: http.StatusConflict,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenInactiveServiceAccount("", "test", "editors", "my-first-project-ID"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "my-first-project-ID",
			expectedResponse: `{"error":{"code":409,"message":"service account \"test\" already exists"}}`,
		},
		{
			name:       "scenario 5: the admin Bob can create service account 'test' for editors group",
			body:       `{"name":"test", "group":"editors"}`,
			httpStatus: http.StatusCreated,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
			},
			existingAPIUser: *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:   "plan9-ID",
			expectedSAName:  "test",
			expectedGroup:   "editors-plan9-ID",
		},
		{
			name:       "scenario 6: the user Bob can not create service account 'test' for editors group for John project",
			body:       `{"name":"test", "group":"editors"}`,
			httpStatus: http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "editors"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", false),
			},
			existingAPIUser:  *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:    "plan9-ID",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't belong to the given project = plan9-ID"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts", tc.projectToSync), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			ep, client, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if tc.httpStatus == http.StatusCreated {
				var sa apiv1.ServiceAccount
				err = json.Unmarshal(res.Body.Bytes(), &sa)
				if err != nil {
					t.Fatal(err.Error())
				}
				if sa.Group != tc.expectedGroup {
					t.Fatalf("expected group %s got %s", tc.expectedGroup, sa.Group)
				}
				if sa.Name != tc.expectedSAName {
					t.Fatalf("expected name %s got %s", tc.expectedSAName, sa.Name)
				}
				if sa.Status != apiv1.ServiceAccountInactive {
					t.Fatalf("expected Inactive state got %s", sa.Status)
				}

				saName := fmt.Sprintf("serviceaccount-%s", sa.ID)
				expectedSA := &kubermaticapiv1.User{}
				err = client.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: saName}, expectedSA)
				if err != nil {
					t.Fatalf("expected SA object got error %v", err)
				}

				if expectedSA.Spec.Name != tc.expectedSAName {
					t.Fatalf("expected name %s got %s", tc.expectedSAName, expectedSA.Spec.Name)
				}

			} else {
				test.CompareWithResult(t, res, tc.expectedResponse)
			}

		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedSA             []apiv1.ServiceAccount
		expectedError          string
		projectToSync          string
		httpStatus             int
		existingAPIUser        apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:          "scenario 1: list active service accounts",
			projectToSync: "plan9-ID",
			httpStatus:    http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			expectedSA: []apiv1.ServiceAccount{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "1",
						Name: "test-1",
					},
					Group:  "editors-plan9-ID",
					Status: "Active",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "3",
						Name: "test-3",
					},
					Group:  "viewers-plan9-ID",
					Status: "Active",
				},
			},
		},
		{
			name:          "scenario 2: list active 'test-3' and inactive 'test-1' service accounts",
			projectToSync: "plan9-ID",
			httpStatus:    http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenInactiveServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenInactiveServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			expectedSA: []apiv1.ServiceAccount{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "1",
						Name: "test-1",
					},
					Group:  "editors-plan9-ID",
					Status: "Inactive",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "3",
						Name: "test-3",
					},
					Group:  "viewers-plan9-ID",
					Status: "Active",
				},
			},
		},
		{
			name:          "scenario 3: the admin Bob can list active service accounts for any project",
			projectToSync: "plan9-ID",
			httpStatus:    http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				test.GenServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingAPIUser: *test.GenAPIUser("bob", "bob@acme.com"),
			expectedSA: []apiv1.ServiceAccount{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "1",
						Name: "test-1",
					},
					Group:  "editors-plan9-ID",
					Status: "Active",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "3",
						Name: "test-3",
					},
					Group:  "viewers-plan9-ID",
					Status: "Active",
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts", tc.projectToSync), nil)
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if tc.httpStatus == http.StatusOK {
				actualSA := test.NewServiceAccountV1SliceWrapper{}
				actualSA.DecodeOrDie(res.Body, t).Sort()

				wrappedExpectedSA := test.NewServiceAccountV1SliceWrapper(tc.expectedSA)
				wrappedExpectedSA.Sort()

				actualSA.EqualOrDie(wrappedExpectedSA, t)

			} else {
				test.CompareWithResult(t, res, tc.expectedError)
			}

		})
	}
}

func TestEdit(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		expectedErrorResponse  string
		expectedGroup          string
		expectedSAName         string
		projectToSync          string
		saToUpdate             string
		httpStatus             int
		existingAPIUser        apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:       "scenario 1: update service account, change name and group",
			body:       `{"id":"19840801", "name":"newName", "group":"editors"}`,
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-19840801@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add service account*/
				test.GenServiceAccount("19840801", "test", "viewers", "plan9-ID"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:   "plan9-ID",
			expectedSAName:  "newName",
			expectedGroup:   "editors-plan9-ID",
			saToUpdate:      "19840801",
		},
		{
			name:       "scenario 2: change service account name for already existing in project",
			body:       `{"id":"19840801", "name":"test-2", "group":"viewers"}`,
			httpStatus: http.StatusConflict,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "viewers"),
				test.GenBinding("plan9-ID", "serviceaccount-2@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add service account*/
				test.GenServiceAccount("19840801", "test-1", "viewers", "plan9-ID"),
				test.GenServiceAccount("2", "test-2", "viewers", "plan9-ID"),
			},
			existingAPIUser:       *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:         "plan9-ID",
			saToUpdate:            "19840801",
			expectedErrorResponse: `{"error":{"code":409,"message":"service account \"test-2\" already exists"}}`,
		},
		{
			name:       "scenario 3: the admin Bob can update service account for any project",
			body:       `{"id":"19840801", "name":"newName", "group":"editors"}`,
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("my-third-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("my-third-project-ID", "john@acme.com", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-19840801@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				/*add service account*/
				test.GenServiceAccount("19840801", "test", "viewers", "plan9-ID"),
			},
			existingAPIUser: *test.GenAPIUser("bob", "bob@acme.com"),
			projectToSync:   "plan9-ID",
			expectedSAName:  "newName",
			expectedGroup:   "editors-plan9-ID",
			saToUpdate:      "19840801",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s", tc.projectToSync, tc.saToUpdate), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			ep, client, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			if tc.httpStatus == http.StatusOK {
				var sa apiv1.ServiceAccount
				err = json.Unmarshal(res.Body.Bytes(), &sa)
				if err != nil {
					t.Fatal(err.Error())
				}
				if sa.Group != tc.expectedGroup {
					t.Fatalf("expected group %s got %s", tc.expectedGroup, sa.Group)
				}
				if sa.Name != tc.expectedSAName {
					t.Fatalf("expected name %s got %s", tc.expectedSAName, sa.Name)
				}

				saName := fmt.Sprintf("serviceaccount-%s", sa.ID)
				expectedSA := &kubermaticapiv1.User{}
				err = client.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: saName}, expectedSA)
				if err != nil {
					t.Fatalf("expected SA object got error %v", err)
				}

				if expectedSA.Spec.Name != tc.expectedSAName {
					t.Fatalf("expected name %s got %s", tc.expectedSAName, expectedSA.Spec.Name)
				}

				group, ok := expectedSA.Labels[serviceaccount.ServiceAccountLabelGroup]
				if !ok {
					t.Fatalf("expected find label %s", serviceaccount.ServiceAccountLabelGroup)
				}

				if group != tc.expectedGroup {
					t.Fatalf("expected group from binding %s got %s", tc.expectedGroup, group)
				}

			} else {
				test.CompareWithResult(t, res, tc.expectedErrorResponse)
			}

		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		projectToSync          string
		saToDelete             string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:            "scenario 1: the owner of the project delete service account",
			httpStatus:      http.StatusOK,
			projectToSync:   test.GenDefaultProject().Name,
			existingAPIUser: test.GenDefaultAPIUser(),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				// add a project
				test.GenDefaultProject(),
				// add a user
				test.GenDefaultUser(),
				// make a user the owner of the default project
				test.GenDefaultOwnerBinding(),
				test.GenBinding("my-first-project-ID", "serviceaccount-1@sa.kubermatic.io", "viewers"),
				/*add service account*/
				test.GenServiceAccount("19840801", "test", "viewers", "my-first-project-ID"),
			},
			saToDelete: "19840801",
		},
		{
			name:            "scenario 2: the admin can delete any service account",
			httpStatus:      http.StatusOK,
			projectToSync:   test.GenDefaultProject().Name,
			existingAPIUser: test.GenAPIUser("john", "john@acme.com"),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				// add a project
				test.GenDefaultProject(),
				// add a user
				test.GenDefaultUser(),
				genUser("john", "john@acme.com", true),
				// make a user the owner of the default project
				test.GenDefaultOwnerBinding(),
				test.GenBinding("my-first-project-ID", "serviceaccount-1@sa.kubermatic.io", "viewers"),
				/*add service account*/
				test.GenServiceAccount("19840801", "test", "viewers", "my-first-project-ID"),
			},
			saToDelete: "19840801",
		},
		{
			name:            "scenario 2: the user John can delete Bob's service account",
			httpStatus:      http.StatusForbidden,
			projectToSync:   test.GenDefaultProject().Name,
			existingAPIUser: test.GenAPIUser("john", "john@acme.com"),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				// add a project
				test.GenDefaultProject(),
				// add a user
				test.GenDefaultUser(),
				genUser("john", "john@acme.com", false),
				// make a user the owner of the default project
				test.GenDefaultOwnerBinding(),
				test.GenBinding("my-first-project-ID", "serviceaccount-1@sa.kubermatic.io", "viewers"),
				/*add service account*/
				test.GenServiceAccount("19840801", "test", "viewers", "my-first-project-ID"),
			},
			saToDelete: "19840801",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts/%s", tc.projectToSync, tc.saToDelete), strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}
		})
	}
}

func genUser(name, email string, isAdmin bool) *kubermaticapiv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}
