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

	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateRoleBinding(t *testing.T) {
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
			name:             "scenario 1: create role binding with name test-1",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"name":"test-1","namespace":"default","roleRefName":"role-1","subjects":[{"kind":"User","name":"test@example.com"}]}`,
			expectedResponse: `{"id":"test-1","name":"test-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default","subjects":[{"kind":"User","name":"test@example.com"}],"roleRefName":"role-1"}`,
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
		// scenario 2
		{
			name:             "scenario 2: create role binding with incorrect namespace",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"name":"test-1","namespace":"test","roleRefName":"role-1","subjects":[{"kind":"User","name":"test@example.com"}]}`,
			expectedResponse: `{"error":{"code":400,"message":"invalid request: the request namespace must be the same as role binding namespace"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusBadRequest,
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
		// scenario 3
		{
			name:             "scenario 3: create role binding with incorrect API group",
			roleName:         "role-1",
			namespace:        "default",
			body:             `{"name":"test-1","namespace":"default","roleRefName":"role-1","subjects":[{"kind":"test","name":"test@example.com"}]}`,
			expectedResponse: `{"error":{"code":400,"message":"invalid request: the request Body subjects contain wrong kind name: 'test'. Should be 'Group' or 'User'"}}`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusBadRequest,
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
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			kubernetesObj := []runtime.Object{}
			kubeObj := []runtime.Object{}
			kubeObj = append(kubeObj, tc.existingKubernrtesObjs...)
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/roles/%s/%s/bindings", test.ProjectName, tc.clusterToGet, tc.namespace, tc.roleName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
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
