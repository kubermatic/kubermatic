package serviceaccount_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		existingKubermaticObjs []runtime.Object
	}{
		{
			name:       "scenario 1: create service account 'test' for editors group by project owner john",
			body:       `{"name":"test", "group":"editors"}`,
			httpStatus: http.StatusCreated,
			existingKubermaticObjs: []runtime.Object{
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
			existingKubermaticObjs: []runtime.Object{
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
			existingKubermaticObjs: []runtime.Object{
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
			existingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenServiceAccount("", "test", "editors", "my-first-project-ID"),
			},
			existingAPIUser:  *test.GenAPIUser("john", "john@acme.com"),
			projectToSync:    "my-first-project-ID",
			expectedResponse: `{"error":{"code":409,"message":"service account \"test\" already exists"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/serviceaccounts", tc.projectToSync), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			ep, client, err := test.CreateTestEndpointAndGetClients(tc.existingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
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

				expectedSA, err := client.FakeKubermaticClient.KubermaticV1().Users().Get(sa.ID, metav1.GetOptions{})
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
