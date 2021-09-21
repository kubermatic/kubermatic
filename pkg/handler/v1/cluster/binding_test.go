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

package cluster_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBindUserToRole(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		roleName               string
		namespace              string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:             "scenario 1: bind user to role test-1",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"namespace":"default","subjects":[{"kind":"User","apiGroup":"rbac.authorization.k8s.io","name":"test@example.com"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-1", "test"),
				test.GenDefaultClusterRole("role-1"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 2: create role binding when cluster role doesn't exist",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"error":{"code":404,"message":"roles.rbac.authorization.k8s.io \"role-1\" not found"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusNotFound,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "test"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: update existing binding for the new user",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"namespace":"default","subjects":[{"kind":"User","name":"bob@acme.com"},{"kind":"User","apiGroup":"rbac.authorization.k8s.io","name":"test@example.com"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 4: bind existing user",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"bob@acme.com"}`,
			expectedResponse: `{"error":{"code":400,"message":"user bob@acme.com already connected to role role-1"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusBadRequest,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// group scenarios
		{
			name:             "scenario 5: bind group to role test-1",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"group":"test"}`,
			expectedResponse: `{"namespace":"default","subjects":[{"kind":"Group","apiGroup":"rbac.authorization.k8s.io","name":"test"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-1", "test"),
				test.GenDefaultClusterRole("role-1"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 6: create role binding when cluster role doesn't exist",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"group":"test"}`,
			expectedResponse: `{"error":{"code":404,"message":"roles.rbac.authorization.k8s.io \"role-1\" not found"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusNotFound,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "test"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 7
		{
			name:             "scenario 7: update existing binding for the new group",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"group":"test"}`,
			expectedResponse: `{"namespace":"default","subjects":[{"kind":"Group","name":"admins"},{"kind":"Group","apiGroup":"rbac.authorization.k8s.io","name":"test"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultGroupRoleBinding("test", "default", "role-1", "admins"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 8: update existing binding for the new group",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"group":"test"}`,
			expectedResponse: `{"namespace":"default","subjects":[{"kind":"User","name":"bob@acme.com"},{"kind":"Group","apiGroup":"rbac.authorization.k8s.io","name":"test"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 9: bind existing group",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"group":"test"}`,
			expectedResponse: `{"error":{"code":400,"message":"group test already connected to role role-1"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusBadRequest,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultGroupRoleBinding("test", "default", "role-1", "test"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 10: the admin John can bind user to role test-1 for any cluster",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"namespace":"default","subjects":[{"kind":"User","apiGroup":"rbac.authorization.k8s.io","name":"test@example.com"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-1", "test"),
				test.GenDefaultClusterRole("role-1"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 11: the user John can not bind user to role test-1 for Bob's cluster",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-1", "test"),
				test.GenDefaultClusterRole("role-1"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles/%s/%s/bindings", test.ProjectName, tc.clusterToGet, tc.namespace, tc.roleName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestUnbindUserFromRoleBinding(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		roleName               string
		namespace              string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
	}{
		{
			name:             "scenario 1: remove user from the binding",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"bob@acme.com"}`,
			expectedResponse: `{"namespace":"default","roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 2: remove group from the binding",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"group":"test"}`,
			expectedResponse: `{"namespace":"default","roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultGroupRoleBinding("test", "default", "role-1", "test"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 3: the admin John can remove user from the binding for the any cluster",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"bob@acme.com"}`,
			expectedResponse: `{"namespace":"default","roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 4: the user John can not remove user from the binding for the Bob's cluster",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"userEmail":"bob@acme.com"}`,
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles/%s/%s/bindings", test.ProjectName, tc.clusterToGet, tc.namespace, tc.roleName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestListRoleBinding(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
	}{
		{
			name:             "scenario 1: list bindings",
			expectedResponse: `[{"namespace":"default","subjects":[{"kind":"User","name":"test-1@example.com"}],"roleRefName":"role-1"},{"namespace":"default","subjects":[{"kind":"User","name":"test-2@example.com"}],"roleRefName":"role-2"},{"namespace":"default","subjects":[{"kind":"Group","name":"test"}],"roleRefName":"role-2"},{"namespace":"test","subjects":[{"kind":"User","name":"test-10@example.com"}],"roleRefName":"role-10"},{"namespace":"test","subjects":[{"kind":"Group","name":"test"}],"roleRefName":"role-10"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-2", "default"),
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultRoleBinding("binding-1", "default", "role-1", "test-1@example.com"),
				test.GenDefaultRoleBinding("binding-2", "default", "role-2", "test-2@example.com"),
				test.GenDefaultGroupRoleBinding("binding-3", "default", "role-2", "test"),
				test.GenDefaultRoleBinding("binding-1", "test", "role-10", "test-10@example.com"),
				test.GenDefaultGroupRoleBinding("binding-2", "test", "role-10", "test"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 2: the admin John can list bindings for any cluster",
			expectedResponse: `[{"namespace":"default","subjects":[{"kind":"User","name":"test-1@example.com"}],"roleRefName":"role-1"},{"namespace":"default","subjects":[{"kind":"User","name":"test-2@example.com"}],"roleRefName":"role-2"},{"namespace":"default","subjects":[{"kind":"Group","name":"test"}],"roleRefName":"role-2"},{"namespace":"test","subjects":[{"kind":"User","name":"test-10@example.com"}],"roleRefName":"role-10"},{"namespace":"test","subjects":[{"kind":"Group","name":"test"}],"roleRefName":"role-10"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-2", "default"),
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultRoleBinding("binding-1", "default", "role-1", "test-1@example.com"),
				test.GenDefaultRoleBinding("binding-2", "default", "role-2", "test-2@example.com"),
				test.GenDefaultGroupRoleBinding("binding-3", "default", "role-2", "test"),
				test.GenDefaultRoleBinding("binding-1", "test", "role-10", "test-10@example.com"),
				test.GenDefaultGroupRoleBinding("binding-2", "test", "role-10", "test"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 3: the user John can not list Bob's bindings",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-2", "default"),
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultRoleBinding("binding-1", "default", "role-1", "test-1@example.com"),
				test.GenDefaultRoleBinding("binding-2", "default", "role-2", "test-2@example.com"),
				test.GenDefaultGroupRoleBinding("binding-3", "default", "role-2", "test"),
				test.GenDefaultRoleBinding("binding-1", "test", "role-10", "test-10@example.com"),
				test.GenDefaultGroupRoleBinding("binding-2", "test", "role-10", "test"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/bindings", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestBindUserToClusterRole(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		roleName               string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:             "scenario 1: bind user to role-1, when cluster role binding doesn't exist",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"error":{"code":500,"message":"the cluster role binding not found"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusInternalServerError,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 2: create cluster role binding when cluster role doesn't exist",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"error":{"code":404,"message":"clusterroles.rbac.authorization.k8s.io \"role-1\" not found"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusNotFound,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-2"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: update existing binding for the new user",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"subjects":[{"kind":"User","name":"bob@acme.com"},{"kind":"User","apiGroup":"rbac.authorization.k8s.io","name":"test@example.com"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 4: bind existing user",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"error":{"code":400,"message":"user test@example.com already connected to the cluster role role-1"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusBadRequest,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "test@example.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// group scenarios
		// scenario 4
		{
			name:             "scenario 5: bind group to role-1, when cluster role binding doesn't exist",
			roleName:         "role-1",
			body:             `{"group":"test"}`,
			expectedResponse: `{"error":{"code":500,"message":"the cluster role binding not found"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusInternalServerError,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 6: create cluster role binding when cluster role doesn't exist",
			roleName:         "role-1",
			body:             `{"group":"test"}`,
			expectedResponse: `{"error":{"code":404,"message":"clusterroles.rbac.authorization.k8s.io \"role-1\" not found"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusNotFound,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-2"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 7
		{
			name:             "scenario 7: update existing binding for the new group",
			roleName:         "role-1",
			body:             `{"group":"test"}`,
			expectedResponse: `{"subjects":[{"kind":"User","name":"bob@acme.com"},{"kind":"Group","apiGroup":"rbac.authorization.k8s.io","name":"test"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 8: update existing binding for the new group",
			roleName:         "role-1",
			body:             `{"group":"test"}`,
			expectedResponse: `{"subjects":[{"kind":"Group","name":"admins"},{"kind":"Group","apiGroup":"rbac.authorization.k8s.io","name":"test"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultGroupClusterRoleBinding("test", "role-1", "admins"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 9: bind existing group",
			roleName:         "role-1",
			body:             `{"group":"test"}`,
			expectedResponse: `{"error":{"code":400,"message":"group test already connected to the cluster role role-1"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusBadRequest,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultGroupClusterRoleBinding("test", "role-1", "test"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 10: admin can update existing binding for the new user for any cluster",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"subjects":[{"kind":"User","name":"bob@acme.com"},{"kind":"User","apiGroup":"rbac.authorization.k8s.io","name":"test@example.com"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 11: user John can not update existing binding for the new user for Bob's cluster",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterroles/%s/clusterbindings", test.ProjectName, tc.clusterToGet, tc.roleName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestUnbindUserFromClusterRoleBinding(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		roleName               string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:             "scenario 1: remove user from existing cluster role binding",
			roleName:         "role-1",
			body:             `{"userEmail":"bob@acme.com"}`,
			expectedResponse: `{"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:             "scenario 2: remove group from existing cluster role binding",
			roleName:         "role-1",
			body:             `{"group":"test"}`,
			expectedResponse: `{"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultGroupClusterRoleBinding("test", "role-1", "test"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 3: the admin can remove user from existing cluster role binding for any cluster",
			roleName:         "role-1",
			body:             `{"userEmail":"bob@acme.com"}`,
			expectedResponse: `{"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 4: the user can not remove user from existing cluster role binding for Bob's cluster",
			roleName:         "role-1",
			body:             `{"userEmail":"bob@acme.com"}`,
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterroles/%s/clusterbindings", test.ProjectName, tc.clusterToGet, tc.roleName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestListClusterRoleBinding(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:             "scenario 1: list cluster role bindings",
			expectedResponse: `[{"subjects":[{"kind":"User","name":"test-1"}],"roleRefName":"role-1"},{"subjects":[{"kind":"User","name":"test-2"}],"roleRefName":"role-1"},{"subjects":[{"kind":"User","name":"test-3"}],"roleRefName":"role-2"},{"subjects":[{"kind":"Group","name":"test-4"}],"roleRefName":"role-2"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("binding-1-1", "role-1", "test-1"),
				test.GenDefaultClusterRoleBinding("binding-1-2", "role-1", "test-2"),
				test.GenDefaultClusterRoleBinding("binding-2-1", "role-2", "test-3"),
				test.GenDefaultGroupClusterRoleBinding("binding-2-2", "role-2", "test-4"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:             "scenario 2: the admin John can list cluster role bindings for any cluster",
			expectedResponse: `[{"subjects":[{"kind":"User","name":"test-1"}],"roleRefName":"role-1"},{"subjects":[{"kind":"User","name":"test-2"}],"roleRefName":"role-1"},{"subjects":[{"kind":"User","name":"test-3"}],"roleRefName":"role-2"},{"subjects":[{"kind":"Group","name":"test-4"}],"roleRefName":"role-2"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("binding-1-1", "role-1", "test-1"),
				test.GenDefaultClusterRoleBinding("binding-1-2", "role-1", "test-2"),
				test.GenDefaultClusterRoleBinding("binding-2-1", "role-2", "test-3"),
				test.GenDefaultGroupClusterRoleBinding("binding-2-2", "role-2", "test-4"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		// scenario 3
		{
			name:             "scenario 3: the user John can not list Bob's cluster role bindings",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultClusterRole("role-1"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRoleBinding("binding-1-1", "role-1", "test-1"),
				test.GenDefaultClusterRoleBinding("binding-1-2", "role-1", "test-2"),
				test.GenDefaultClusterRoleBinding("binding-2-1", "role-2", "test-3"),
				test.GenDefaultGroupClusterRoleBinding("binding-2-2", "role-2", "test-4"),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterbindings", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}
