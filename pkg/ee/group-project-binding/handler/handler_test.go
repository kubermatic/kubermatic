//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestHandlerGroupProjectBindings(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name             string
		method           string
		url              string
		body             string
		existingAPIUser  *apiv1.User
		existingObjects  []ctrlruntimeclient.Object
		httpStatus       int
		expectedResponse string
		validateResp     func(resp *httptest.ResponseRecorder) error
	}{
		{
			name:            "scenario 1: list GroupProjectBindings in a project",
			method:          "GET",
			url:             "/api/v2/projects/foo-ID/groupbindings",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("boo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("foo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("foo-ID", "TestGroup", "owners"),
				test.GenGroupBinding("boo-ID", "TestGroup", "owners"),
			},
			httpStatus: 200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				bindingList := &[]apiv2.GroupProjectBinding{}
				err := json.Unmarshal(resp.Body.Bytes(), bindingList)
				if err != nil {
					return err
				}
				listLen := len(*bindingList)
				expectedListLen := 1
				if expectedListLen != listLen {
					return fmt.Errorf("expected list length %d, got %d", expectedListLen, listLen)
				}
				return nil
			},
		},
		{
			name:            "scenario 2: list GroupProjectBindings in an illicit project",
			method:          "GET",
			url:             "/api/v2/projects/foo-ID/groupbindings",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("boo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("boo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("foo-ID", "TestGroup", "owners"),
				test.GenGroupBinding("boo-ID", "TestGroup", "owners"),
			},
			httpStatus: 403,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 3: get an existing GroupProjectBinding",
			method:          "GET",
			url:             "/api/v2/projects/boo-ID/groupbindings/boo-ID-xxxxxxxxxx",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("boo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("boo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("boo-ID", "TestGroup", "owners"),
			},
			httpStatus: 200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				binding := &apiv2.GroupProjectBinding{}
				err := json.Unmarshal(resp.Body.Bytes(), binding)
				if err != nil {
					return err
				}
				expectedName := "boo-ID-xxxxxxxxxx"
				if expectedName != binding.Name {
					return fmt.Errorf("expected name %s, got %s", expectedName, binding.Name)
				}
				return nil
			},
		},
		{
			name:            "scenario 4: get a non-existing GroupProjectBinding",
			method:          "GET",
			url:             "/api/v2/projects/boo-ID/groupbindings/boo-ID-DoesNotExist",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("boo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("boo-ID", "bob@acme.com", "editors"),
			},
			httpStatus: 404,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 5: get an illicit GroupProjectBinding",
			method:          "GET",
			url:             "/api/v2/projects/foo-ID/groupbindings/foo-ID-xxxxxxxxxx",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenProject("boo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("boo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("boo-ID", "TestGroup", "owners"),
			},
			httpStatus: 403,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 6: create a new GroupProjectBinding",
			method: "POST",
			url:    "/api/v2/projects/foo-ID/groupbindings",
			body: `{
				"role": "viewers",
				"group": "viewers-test"
			}`,
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("foo-ID", "bob@acme.com", "editors"),
			},
			httpStatus: 201,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 7: create a new GroupProjectBinding with invalid role name",
			method: "POST",
			url:    "/api/v2/projects/foo-ID/groupbindings",
			body: `{
				"role": "invalid",
				"group": "viewers-test"
			}`,
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("foo-ID", "bob@acme.com", "editors"),
			},
			httpStatus: 400,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 8: delete an existing GroupProjectBinding",
			method:          "DELETE",
			url:             "/api/v2/projects/foo-ID/groupbindings/foo-ID-xxxxxxxxxx",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("foo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("foo-ID", "viewers-test", "viewers"),
			},
			httpStatus: 200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 9: patch an existing GroupProjectBinding",
			method: "PATCH",
			body: `{
				"role": "owners"
			}`,
			url:             "/api/v2/projects/foo-ID/groupbindings/foo-ID-xxxxxxxxxx",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("foo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("foo-ID", "viewers-test", "viewers"),
			},
			httpStatus: 200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 9: patch a non-existing GroupProjectBinding",
			method: "PATCH",
			body: `{
				"role": "owners"
			}`,
			url:             "/api/v2/projects/foo-ID/groupbindings/foo-ID-nonexisting",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("foo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("foo-ID", "viewers-test", "viewers"),
			},
			httpStatus: 404,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 10: patch an existing GroupProjectBinding with illicit role",
			method: "PATCH",
			body: `{
				"role": "invalid"
			}`,
			url:             "/api/v2/projects/foo-ID/groupbindings/foo-ID-xxxxxxxxxx",
			existingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
			existingObjects: []ctrlruntimeclient.Object{
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				test.GenBinding("foo-ID", "bob@acme.com", "editors"),
				test.GenGroupBinding("foo-ID", "viewers-test", "viewers"),
			},
			httpStatus: 400,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			router, err := test.CreateTestEndpoint(*tc.existingAPIUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint")
			}
			router.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			err = tc.validateResp(res)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
