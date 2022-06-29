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

package resourcequota_test

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
	"k8c.io/kubermatic/v2/pkg/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestHandlerResourceQuotas(t *testing.T) {
	t.Parallel()

	existingResourceQuotas := []ctrlruntimeclient.Object{
		&kubermaticv1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("project-%s-1", projectName),
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
					kubermaticv1.ResourceQuotaSubjectNameLabelKey: fmt.Sprintf("%s-1", projectName),
				},
			},
			Spec: kubermaticv1.ResourceQuotaSpec{
				Subject: kubermaticv1.Subject{
					Name: fmt.Sprintf("%s-1", projectName),
					Kind: kubermaticv1.ProjectSubjectKind,
				},
			},
		},
		&kubermaticv1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("project-%s-2", projectName),
				Namespace: resources.KubermaticNamespace,
				Labels: map[string]string{
					kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
					kubermaticv1.ResourceQuotaSubjectNameLabelKey: fmt.Sprintf("%s-2", projectName),
				},
			},
			Spec: kubermaticv1.ResourceQuotaSpec{
				Subject: kubermaticv1.Subject{
					Name: fmt.Sprintf("%s-2", projectName),
					Kind: kubermaticv1.ProjectSubjectKind,
				},
			},
		},
	}

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
			name:            "scenario 1: list all resource quotas",
			method:          "GET",
			url:             "/api/v2/quotas",
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				resourceQuotaList := &[]apiv2.ResourceQuota{}
				err := json.Unmarshal(resp.Body.Bytes(), resourceQuotaList)
				if err != nil {
					return err
				}
				listLen := len(*resourceQuotaList)
				expectedListLen := len(existingResourceQuotas)
				if listLen != expectedListLen {
					return fmt.Errorf("expected list length %d, got %d", expectedListLen, listLen)
				}
				return nil
			},
		},
		{
			name:            "scenario 2: list filtered resource quotas",
			method:          "GET",
			url:             fmt.Sprintf("/api/v2/quotas?subjectName=%s-1", projectName),
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				resourceQuotaList := &[]apiv2.ResourceQuota{}
				err := json.Unmarshal(resp.Body.Bytes(), resourceQuotaList)
				if err != nil {
					return err
				}
				listLen := len(*resourceQuotaList)
				expectedListLen := 1
				if listLen != expectedListLen {
					return fmt.Errorf("expected list length %d, got %d", expectedListLen, listLen)
				}
				return nil
			},
		},
		{
			name:            "scenario 3: get a single resource quota",
			method:          "GET",
			url:             fmt.Sprintf("/api/v2/quotas/project-%s-1", projectName),
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				resourceQuota := &apiv2.ResourceQuota{}
				err := json.Unmarshal(resp.Body.Bytes(), resourceQuota)
				if err != nil {
					return err
				}
				expectedName := fmt.Sprintf("project-%s-1", projectName)
				if resourceQuota.Name != expectedName {
					return fmt.Errorf("expected name %s, got %s", expectedName, resourceQuota.Name)
				}
				return nil
			},
		},
		{
			name:            "scenario 4: get a non-existing single resource quota",
			method:          "GET",
			url:             "/api/v2/quotas/project-non-existing",
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      404,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 5: create an existing resource quota",
			method: "POST",
			url:    "/api/v2/quotas",
			body: `{
				"subject": {
					"kind": "project",
					"name": "` + fmt.Sprintf("%s-1", projectName) + `"
				}
			}`,
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      409,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 6: create a new resource quota",
			method: "POST",
			url:    "/api/v2/quotas",
			body: `{
				"subject": {
					"kind": "project",
					"name": "testproject"
				},
				"quota": {
					"cpu": 10,
					"memory": "64Gi",
					"storage": "256Gi"
				}
			}`,
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      201,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 7: update an existing resource quota",
			method: "PATCH",
			url:    fmt.Sprintf("/api/v2/quotas/project-%s-1", projectName),
			body: `{
				"cpu": 10,
				"memory": "64Gi",
				"storage": "256Gi"
			}`,
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 8: update a non-existing resource quota",
			method: "PATCH",
			url:    "/api/v2/quotas/project-non-existing",
			body: `{
				"cpu": 10,
				"memory": "64Gi",
				"storage": "256Gi"
			}`,
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      404,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 9: delete an existing resource quota",
			method:          "DELETE",
			url:             fmt.Sprintf("/api/v2/quotas/project-%s-1", projectName),
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      200,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 10: delete a non-existing resource quota",
			method:          "DELETE",
			url:             "/api/v2/quotas/project-non-existing",
			existingAPIUser: test.GenDefaultAdminAPIUser(),
			existingObjects: existingResourceQuotas,
			httpStatus:      404,
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
