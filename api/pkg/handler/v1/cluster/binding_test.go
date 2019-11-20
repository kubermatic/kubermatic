package cluster_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		existingKubermaticObjs []runtime.Object
		existingKubernrtesObjs []runtime.Object
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
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultRole("role-1", "default"),
				genDefaultRole("role-1", "test"),
				genDefaultClusterRole("role-1"),
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
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultRole("role-1", "test"),
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
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultRole("role-1", "default"),
				genDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
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
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultRole("role-1", "default"),
				genDefaultRoleBinding("test", "default", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			kubeObj = append(kubeObj, tc.existingKubernrtesObjs...)
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles/%s/%s/bindings", test.ProjectName, tc.clusterToGet, tc.namespace, tc.roleName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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
		existingKubermaticObjs []runtime.Object
		existingKubernrtesObjs []runtime.Object
	}{
		{
			name:             "scenario 1: list bindings",
			expectedResponse: `[{"namespace":"default","subjects":[{"kind":"User","name":"test-1@example.com"}],"roleRefName":"role-1"},{"namespace":"default","subjects":[{"kind":"User","name":"test-2@example.com"}],"roleRefName":"role-2"},{"namespace":"test","subjects":[{"kind":"User","name":"test-10@example.com"}],"roleRefName":"role-10"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultRole("role-1", "default"),
				genDefaultRole("role-2", "default"),
				genDefaultClusterRole("role-1"),
				genDefaultRoleBinding("binding-1", "default", "role-1", "test-1@example.com"),
				genDefaultRoleBinding("binding-2", "default", "role-2", "test-2@example.com"),
				genDefaultRoleBinding("binding-1", "test", "role-10", "test-10@example.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			var kubermaticObj []runtime.Object
			kubeObj = append(kubeObj, tc.existingKubernrtesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/bindings", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func genDefaultRoleBinding(name, namespace, roleID, userEmail string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterBindingComponentValue},
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: userEmail,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: roleID,
		},
	}
}

func genDefaultClusterRoleBinding(name, roleID, userEmail string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterBindingComponentValue},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: userEmail,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: roleID,
		},
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
		existingKubermaticObjs []runtime.Object
		existingKubernrtesObjs []runtime.Object
	}{
		// scenario 1
		{
			name:             "scenario 1: bind user to role-1",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"subjects":[{"kind":"User","apiGroup":"rbac.authorization.k8s.io","name":"test@example.com"}],"roleRefName":"role-1"}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultClusterRole("role-1"),
				genDefaultClusterRole("role-2"),
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
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultClusterRole("role-2"),
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
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultClusterRole("role-1"),
				genDefaultClusterRole("role-2"),
				genDefaultClusterRoleBinding("test", "role-1", "bob@acme.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 4: bind existing user",
			roleName:         "role-1",
			body:             `{"userEmail":"test@example.com"}`,
			expectedResponse: `{"error":{"code":400,"message":"user test@example.com already connected to role role-1"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusBadRequest,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultClusterRole("role-1"),
				genDefaultClusterRoleBinding("test", "role-1", "test@example.com"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			var kubermaticObj []runtime.Object
			kubeObj = append(kubeObj, tc.existingKubernrtesObjs...)
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterroles/%s/clusterbindings", test.ProjectName, tc.clusterToGet, tc.roleName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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
		existingKubermaticObjs []runtime.Object
		existingKubernrtesObjs []runtime.Object
	}{
		// scenario 1
		{
			name:             "scenario 1: list cluster role bindings",
			expectedResponse: `[{"subjects":[{"kind":"User","name":"test-1"}],"roleRefName":"role-1"},{"subjects":[{"kind":"User","name":"test-2"}],"roleRefName":"role-1"},{"subjects":[{"kind":"User","name":"test-3"}],"roleRefName":"role-2"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				genDefaultClusterRole("role-1"),
				genDefaultClusterRole("role-2"),
				genDefaultClusterRoleBinding("binding-1-1", "role-1", "test-1"),
				genDefaultClusterRoleBinding("binding-1-2", "role-1", "test-2"),
				genDefaultClusterRoleBinding("binding-2-1", "role-2", "test-3"),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			var kubermaticObj []runtime.Object
			kubeObj = append(kubeObj, tc.existingKubernrtesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/clusterbindings", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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
