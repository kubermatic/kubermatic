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

func TestCreateClusterRole(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToGet           string
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: create cluster role",
			Body:             `{"name":"test","rules": [{"apiGroups": [""],"resources": ["pods"],"verbs": ["get"]}]}`,
			ExpectedResponse: `{"id":"test","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","rules":[{"verbs":["get"],"apiGroups":[""],"resources":["pods"]}]}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterroles", test.ProjectName, tc.ClusterToGet), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestCreateRole(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToGet           string
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: create role test for kubermatic namespace",
			Body:             `{"name":"test","namespace":"kubermatic","rules": [{"apiGroups": [""],"resources": ["pods"],"verbs": ["get"]}]}`,
			ExpectedResponse: `{"id":"test","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"kubermatic","rules":[{"verbs":["get"],"apiGroups":[""],"resources":["pods"]}]}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: create role test when namespace is missing",
			Body:             `{"name":"test","rules": [{"apiGroups": [""],"resources": ["pods"],"verbs": ["get"]}]}`,
			ExpectedResponse: `{"error":{"code":400,"message":"invalid request: the request Body name and namespace cannot be empty"}}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusBadRequest,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles", test.ProjectName, tc.ClusterToGet), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListRole(t *testing.T) {
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
			name:             "scenario 1: list all roles",
			expectedResponse: `[{"id":"role-1","name":"role-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default","rules":[{"verbs":["get"],"apiGroups":[""],"resources":["pod"]}]},{"id":"role-2","name":"role-2","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"test","rules":[{"verbs":["get"],"apiGroups":[""],"resources":["pod"]}]}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-2", "test"),
				test.GenDefaultClusterRole("role-2"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
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

func TestListClusterRole(t *testing.T) {
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
			name:             "scenario 1: list all cluster roles",
			expectedResponse: `[{"id":"role-2","name":"role-2","creationTimestamp":"0001-01-01T00:00:00Z","rules":[{"verbs":["get","list"],"apiGroups":[""],"resources":["pod"]}]},{"id":"role-3","name":"role-3","creationTimestamp":"0001-01-01T00:00:00Z","rules":[{"verbs":["get","list"],"apiGroups":[""],"resources":["pod"]}]}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRole("role-3"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterroles", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
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

func TestGetRole(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
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
			name:             "scenario 1: get role with name role-1 and from the default namespace",
			roleName:         "role-1",
			namespace:        "default",
			expectedResponse: `{"id":"role-1","name":"role-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default","rules":[{"verbs":["get"],"apiGroups":[""],"resources":["pod"]}]}`,
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
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles/%s/%s", test.ProjectName, tc.clusterToGet, tc.namespace, tc.roleName), strings.NewReader(""))
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

func TestGetClusterRole(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
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
			name:             "scenario 1: get cluster role",
			roleName:         "role-2",
			expectedResponse: `{"id":"role-2","name":"role-2","creationTimestamp":"0001-01-01T00:00:00Z","rules":[{"verbs":["get","list"],"apiGroups":[""],"resources":["pod"]}]}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-2", "default"),
				test.GenDefaultClusterRole("role-2"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterroles/%s", test.ProjectName, tc.clusterToGet, tc.roleName), strings.NewReader(""))
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

func TestPatchRole(t *testing.T) {
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
			name:             "scenario 1: patch rules: role resources and api group",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"rules":[{"verbs":["get"],"apiGroups":["*"],"resources":["pod","node"]}]}`,
			expectedResponse: `{"id":"role-1","name":"role-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default","rules":[{"verbs":["get"],"apiGroups":["*"],"resources":["pod","node"]}]}`,
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
		// scenario 2
		{
			name:             "scenario 2: add new rule",
			roleName:         "role-1",
			namespace:        "test",
			body:             `{"rules":[{"verbs":["get"],"apiGroups":["*"],"resources":["pod","node"]},{"verbs":["*"],"apiGroups":["*"],"resources":["*"]}]}`,
			expectedResponse: `{"id":"role-1","name":"role-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"test","rules":[{"verbs":["get"],"apiGroups":["*"],"resources":["pod","node"]},{"verbs":["*"],"apiGroups":["*"],"resources":["*"]}]}`,
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
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles/%s/%s", test.ProjectName, tc.clusterToGet, tc.namespace, tc.roleName), strings.NewReader(tc.body))
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

func TestPatchClusterRole(t *testing.T) {
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
			name:             "scenario 1: patch rules: verbs",
			roleName:         "role-1",
			body:             `{"rules":[{"verbs":["get","delete"],"apiGroups":[""],"resources":["pod"]}]}`,
			expectedResponse: `{"id":"role-1","name":"role-1","creationTimestamp":"0001-01-01T00:00:00Z","rules":[{"verbs":["get","delete"],"apiGroups":[""],"resources":["pod"]}]}`,
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
		// scenario 2
		{
			name:             "scenario 2: add new rule",
			roleName:         "role-1",
			body:             `{"rules":[{"verbs":["get"],"apiGroups":[""],"resources":["pod"]},{"verbs":["delete"],"apiGroups":[""],"resources":["node"]}]}`,
			expectedResponse: `{"id":"role-1","name":"role-1","creationTimestamp":"0001-01-01T00:00:00Z","rules":[{"verbs":["get"],"apiGroups":[""],"resources":["pod"]},{"verbs":["delete"],"apiGroups":[""],"resources":["node"]}]}`,
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
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterroles/%s", test.ProjectName, tc.clusterToGet, tc.roleName), strings.NewReader(tc.body))
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

func TestListRoleNmaes(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       []apiv1.RoleName
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name: "scenario 1: list all role names",
			expectedResponse: []apiv1.RoleName{
				{
					Name:      "role-1",
					Namespace: []string{"default", "test"},
				},
				{
					Name:      "role-2",
					Namespace: []string{"default", "test-2"},
				},
			},
			clusterToGet: test.GenDefaultCluster().Name,
			httpStatus:   http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultRole("role-2", "default"),
				test.GenDefaultRole("role-1", "test"),
				test.GenDefaultRole("role-2", "test-2"),
				test.GenDefaultClusterRole("role-2"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/rolenames", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
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

			actualRoles := test.NewRoleNameSliceWrapper{}
			actualRoles.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedRoles := test.NewRoleNameSliceWrapper(tc.expectedResponse)
			wrappedExpectedRoles.Sort()
			actualRoles.EqualOrDie(wrappedExpectedRoles, t)

		})
	}
}

func TestListClusterRoleNames(t *testing.T) {
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
			name:             "scenario 1: list all cluster role names",
			expectedResponse: `[{"name":"role-2"},{"name":"role-3"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
				test.GenDefaultClusterRole("role-2"),
				test.GenDefaultClusterRole("role-3"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:             "scenario 2: no cluster roles",
			expectedResponse: `[]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				test.GenDefaultRole("role-1", "default"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterrolenames", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
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
