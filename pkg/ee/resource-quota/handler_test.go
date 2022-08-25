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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestHandlerResourceQuotas(t *testing.T) {
	t.Parallel()

	rq1 := &kubermaticv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("project-%s", projectName),
			Labels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: projectName,
			},
		},
		Spec: kubermaticv1.ResourceQuotaSpec{
			Subject: kubermaticv1.Subject{
				Name: projectName,
				Kind: kubermaticv1.ProjectSubjectKind,
			},
			Quota: genQuota(resource.MustParse("5"), resource.MustParse("1235M"), resource.MustParse("125Gi")),
		},
	}
	rq2 := &kubermaticv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("project-%s", anotherProjectName),
			Labels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: anotherProjectName,
			},
		},
		Spec: kubermaticv1.ResourceQuotaSpec{
			Subject: kubermaticv1.Subject{
				Name: anotherProjectName,
				Kind: kubermaticv1.ProjectSubjectKind,
			},
			Quota: genQuota(resource.MustParse("0"), resource.MustParse("1234M"), resource.MustParse("0")),
		},
	}

	admin := test.GenAdminUser("John", "john@acme.com", true)
	project2 := test.GenProject("my-second-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp())
	existingObjects := test.GenDefaultKubermaticObjects(rq1, rq2, admin, project2)

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
			name:            "scenario 1: list all resource quotas with proper quota conversion",
			method:          http.MethodGet,
			url:             "/api/v2/quotas",
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusOK,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				resourceQuotaList := &[]apiv2.ResourceQuota{}
				err := json.Unmarshal(resp.Body.Bytes(), resourceQuotaList)
				if err != nil {
					return err
				}
				listLen := len(*resourceQuotaList)
				if listLen != 2 {
					return fmt.Errorf("expected list length %d, got %d", 2, listLen)
				}
				for _, rq := range *resourceQuotaList {
					var expectedQuota apiv2.Quota
					if rq.Name == rq1.Name {
						expectedQuota = genAPIQuota(5, 1.24, 134.22)
					} else {
						expectedQuota = genAPIQuota(0, 1.23, 0)
					}
					if !diff.DeepEqual(expectedQuota, rq.Quota) {
						return fmt.Errorf("Objects differ:\n%v", diff.ObjectDiff(expectedQuota, rq.Quota))
					}
					if rq.SubjectHumanReadableName != strings.TrimSuffix(rq.SubjectName, "-ID") {
						return fmt.Errorf(
							"human-readable name is not correct: expected %s, got %s",
							projectName,
							rq.SubjectHumanReadableName,
						)
					}
				}
				return nil
			},
		},
		{
			name:            "scenario 2: list filtered resource quotas",
			method:          http.MethodGet,
			url:             fmt.Sprintf("/api/v2/quotas?subjectName=%s", projectName),
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusOK,
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
			method:          http.MethodGet,
			url:             fmt.Sprintf("/api/v2/quotas/project-%s", projectName),
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusOK,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				resourceQuota := &apiv2.ResourceQuota{}
				err := json.Unmarshal(resp.Body.Bytes(), resourceQuota)
				if err != nil {
					return err
				}
				expectedName := fmt.Sprintf("project-%s", projectName)
				if resourceQuota.Name != expectedName {
					return fmt.Errorf("expected name %s, got %s", expectedName, resourceQuota.Name)
				}
				expectedHumanReadableName := strings.TrimSuffix(resourceQuota.SubjectName, "-ID")
				if resourceQuota.SubjectHumanReadableName != expectedHumanReadableName {
					return fmt.Errorf(
						"expected name %s, got %s",
						expectedHumanReadableName,
						resourceQuota.SubjectHumanReadableName,
					)
				}
				return nil
			},
		},
		{
			name:            "scenario 4: get a non-existing single resource quota",
			method:          http.MethodGet,
			url:             "/api/v2/quotas/project-non-existing",
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusNotFound,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 5: create an existing resource quota",
			method: http.MethodPost,
			url:    "/api/v2/quotas",
			body: `{
		      "subjectKind": "project",
		      "subjectName": "` + projectName + `"
			}`,
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusConflict,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 6: create a new resource quota",
			method: http.MethodPost,
			url:    "/api/v2/quotas",
			body: `{
		      "subjectKind": "project",
		      "subjectName": "testproject",
				"quota": {
					"cpu": 10,
					"memory": 64,
					"storage": 256.5
				}
			}`,
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusCreated,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 7: update an existing resource quota",
			method: http.MethodPatch,
			url:    fmt.Sprintf("/api/v2/quotas/project-%s", projectName),
			body: `{
				"cpu": 10,
				"memory": 64,
				"storage": 256.5
			}`,
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusOK,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:   "scenario 8: update a non-existing resource quota",
			method: http.MethodPatch,
			url:    "/api/v2/quotas/project-non-existing",
			body: `{
				"cpu": 10,
				"memory": 64,
				"storage": 256.5
			}`,
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusNotFound,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 9: delete an existing resource quota",
			method:          http.MethodDelete,
			url:             fmt.Sprintf("/api/v2/quotas/project-%s", projectName),
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusOK,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 10: delete a non-existing resource quota",
			method:          http.MethodDelete,
			url:             "/api/v2/quotas/project-non-existing",
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			existingObjects: existingObjects,
			httpStatus:      http.StatusNotFound,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				return nil
			},
		},
		{
			name:            "scenario 11: get a project resource quota",
			method:          http.MethodGet,
			url:             fmt.Sprintf("/api/v2/projects/%s/quota", projectName),
			existingAPIUser: test.GenDefaultAPIUser(),
			existingObjects: existingObjects,
			httpStatus:      http.StatusOK,
			validateResp: func(resp *httptest.ResponseRecorder) error {
				resourceQuota := &apiv2.ResourceQuota{}
				err := json.Unmarshal(resp.Body.Bytes(), resourceQuota)
				if err != nil {
					return err
				}
				expectedName := fmt.Sprintf("project-%s", projectName)
				if resourceQuota.Name != expectedName {
					return fmt.Errorf("expected name %s, got %s", expectedName, resourceQuota.Name)
				}
				expectedHumanReadableName := strings.TrimSuffix(resourceQuota.SubjectName, "-ID")
				if resourceQuota.SubjectHumanReadableName != expectedHumanReadableName {
					return fmt.Errorf("expected name %s, got %s", expectedHumanReadableName, resourceQuota.Name)
				}
				return nil
			},
		},
		{
			name:            "scenario 12: user bob can't get a project resource quota from a project he doesn't belong to",
			method:          http.MethodGet,
			url:             fmt.Sprintf("/api/v2/projects/%s-2/quota", projectName),
			existingAPIUser: test.GenDefaultAPIUser(),
			existingObjects: append(existingObjects, func() *kubermaticv1.Project {
				p := test.GenDefaultProject()
				p.Name = fmt.Sprintf("%s-2", projectName)
				return p
			}()),
			httpStatus: http.StatusForbidden,
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

func genQuota(cpu resource.Quantity, mem resource.Quantity, storage resource.Quantity) kubermaticv1.ResourceDetails {
	return kubermaticv1.ResourceDetails{
		CPU:     &cpu,
		Memory:  &mem,
		Storage: &storage,
	}
}

func genAPIQuota(cpu int64, mem, storage float64) apiv2.Quota {
	quota := apiv2.Quota{
		CPU: &cpu,
	}
	if mem != 0 {
		quota.Memory = &mem
	}
	if storage != 0 {
		quota.Storage = &storage
	}
	return quota
}
