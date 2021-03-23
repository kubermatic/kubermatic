/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	serviceaccount "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateMainServiceAccount(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		expectedResponse       string
		expectedGroup          string
		expectedSAName         string
		httpStatus             int
		existingAPIUser        apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:       "scenario 1: create service account 'test' for editors group",
			body:       `{"name":"test", "group":"editors"}`,
			httpStatus: http.StatusCreated,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			expectedSAName:  "test",
			expectedGroup:   "editors",
		},
		{
			name:       "scenario 2: check forbidden owner group",
			body:       `{"name":"test", "group":"fake-group"}`,
			httpStatus: http.StatusBadRequest,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			expectedResponse: `{"error":{"code":400,"message":"invalid group name fake-group"}}`,
		},
		{
			name:       "scenario 3: check when given name is already reserved",
			body:       `{"name":"test", "group":"editors"}`,
			httpStatus: http.StatusConflict,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenMainServiceAccount("", "test", "editors", "john@acme.com"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			expectedResponse: `{"error":{"code":409,"message":"service account \"test\" already exists"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v2/serviceaccounts", strings.NewReader(tc.body))
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

				saName := fmt.Sprintf("main-serviceaccount-%s", sa.ID)
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

func TestListMainServiceAccounts(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedSA             []apiv1.ServiceAccount
		expectedError          string
		httpStatus             int
		existingAPIUser        apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:       "scenario 1: list main service accounts",
			httpStatus: http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("plan9", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("plan9-ID", "john@acme.com", "owners"),
				test.GenBinding("plan9-ID", "serviceaccount-1@sa.kubermatic.io", "editors"),
				test.GenBinding("plan9-ID", "serviceaccount-3@sa.kubermatic.io", "viewers"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenMainServiceAccount("4", "test-4", "editors", "john@acme.com"),
				test.GenMainServiceAccount("5", "test-5", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("6", "test-5", "viewers", "bob@acme.com"),
				test.GenProjectServiceAccount("1", "test-1", "editors", "plan9-ID"),
				test.GenProjectServiceAccount("2", "test-2", "editors", "test-ID"),
				test.GenProjectServiceAccount("3", "test-3", "viewers", "plan9-ID"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			expectedSA: []apiv1.ServiceAccount{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "4",
						Name: "test-4",
					},
					Group:  "editors",
					Status: "Active",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "5",
						Name: "test-5",
					},
					Group:  "viewers",
					Status: "Active",
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v2/serviceaccounts", nil)
			res := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(tc.existingAPIUser, []ctrlruntimeclient.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
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

func TestEditMainServiceAccounts(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		expectedErrorResponse  string
		expectedGroup          string
		expectedSAName         string
		saToUpdate             string
		httpStatus             int
		existingAPIUser        apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:       "scenario 1: update service account, change name and group",
			body:       `{"id":"4", "name":"newName", "group":"editors"}`,
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
				test.GenProjectServiceAccount("19840801", "test", "viewers", "plan9-ID"),
				test.GenMainServiceAccount("4", "test-4", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("5", "test-5", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("6", "test-5", "viewers", "bob@acme.com"),
			},
			existingAPIUser: *test.GenAPIUser("john", "john@acme.com"),
			expectedSAName:  "newName",
			expectedGroup:   "editors",
			saToUpdate:      "4",
		},
		{
			name:       "scenario 2: change service account name for already existing",
			body:       `{"id":"4", "name":"test-5", "group":"owners"}`,
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
				test.GenProjectServiceAccount("19840801", "test-1", "viewers", "plan9-ID"),
				test.GenProjectServiceAccount("2", "test-2", "viewers", "plan9-ID"),
				test.GenMainServiceAccount("4", "test-4", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("5", "test-5", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("6", "test-5", "viewers", "bob@acme.com"),
			},
			existingAPIUser:       *test.GenAPIUser("john", "john@acme.com"),
			saToUpdate:            "4",
			expectedErrorResponse: `{"error":{"code":409,"message":"service account \"test-5\" already exists"}}`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v2/serviceaccounts/%s", tc.saToUpdate), strings.NewReader(tc.body))
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

				saName := fmt.Sprintf("main-serviceaccount-%s", sa.ID)
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
		saToDelete             string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:            "scenario 1: owner can delete service account",
			httpStatus:      http.StatusOK,
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
				test.GenProjectServiceAccount("19840801", "test", "viewers", "my-first-project-ID"),
				test.GenMainServiceAccount("4", "test-4", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("5", "test-5", "viewers", "john@acme.com"),
				test.GenMainServiceAccount("6", "test-5", "viewers", "bob@acme.com"),
			},
			saToDelete: "6",
		},
		{
			name:            "scenario 2: the user John can't delete Bob's main service account",
			httpStatus:      http.StatusForbidden,
			existingAPIUser: test.GenAPIUser("john", "john@acme.com"),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				// add a project
				test.GenDefaultProject(),
				// add a user
				test.GenDefaultUser(),
				test.GenAdminUser("john", "john@acme.com", false),
				// make a user the owner of the default project
				test.GenDefaultOwnerBinding(),
				test.GenBinding("my-first-project-ID", "serviceaccount-1@sa.kubermatic.io", "viewers"),
				/*add service account*/
				test.GenProjectServiceAccount("19840801", "test", "viewers", "my-first-project-ID"),
				test.GenMainServiceAccount("6", "test-5", "viewers", "bob@acme.com"),
			},
			saToDelete: "6",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v2/serviceaccounts/%s", tc.saToDelete), strings.NewReader(""))
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
