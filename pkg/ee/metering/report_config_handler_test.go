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

package metering_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetMeteringReportConfigEndpoint(t *testing.T) {
	t.Parallel()

	var retention uint32 = 14
	testSeed := test.GenTestSeed(func(seed *kubermaticv1.Seed) {
		seed.Spec.Metering = &kubermaticv1.MeteringConfiguration{
			Enabled:          true,
			StorageClassName: "test",
			StorageSize:      "10Gi",
			ReportConfigurations: map[string]*kubermaticv1.MeteringReportConfiguration{
				"weekly": {
					Schedule:  "0 1 * * 6",
					Interval:  7,
					Retention: &retention,
				},
			},
		}
	})

	testcases := []struct {
		name                   string
		reportName             string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingAPIUser        *apiv1.User
		httpStatus             int
		expectedResponse       string
	}{
		// scenario 1
		{
			name:                   "List metering report configurations.",
			reportName:             "",
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusOK,
			expectedResponse:       `[{"name":"weekly","schedule":"0 1 * * 6","interval":7,"retention":14}]`,
		},
		// scenario 2
		{
			name:                   "Fetch single metering report configuration.",
			reportName:             "weekly",
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusOK,
			expectedResponse:       `{"name":"weekly","schedule":"0 1 * * 6","interval":7,"retention":14}`,
		},
		// scenario 3
		{
			name:                   "Fetch non-existing metering report configuration.",
			reportName:             "non-existing",
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusNotFound,
			expectedResponse:       `{"error":{"code":404,"message":"MeteringReportConfiguration \"non-existing\" not found"}}`,
		},
		// scenario 4
		{
			name:                   "Fetch metering report configuration from seed without metering.",
			reportName:             "metering-does-not-exist",
			existingKubermaticObjs: []ctrlruntimeclient.Object{test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusNotFound,
			expectedResponse:       `{"error":{"code":404,"message":"MeteringReportConfiguration \"metering-does-not-exist\" not found"}}`,
		},
		// scenario 5
		{
			name:                   "List metering report configurations from seed without metering.",
			reportName:             "",
			existingKubermaticObjs: []ctrlruntimeclient.Object{test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusOK,
			expectedResponse:       `[]`,
		},
	}

	for _, tc := range testcases {
		reqURL := "/api/v1/admin/metering/configurations/reports"
		if tc.reportName != "" {
			reqURL += "/" + tc.reportName
		}
		req := httptest.NewRequest(http.MethodGet, reqURL, strings.NewReader(""))
		res := httptest.NewRecorder()

		router, err := test.CreateTestEndpoint(*tc.existingAPIUser, nil, tc.existingKubermaticObjs, nil, hack.NewTestRouting)
		if err != nil {
			t.Fatalf("failed to create test endpoint")
		}
		router.ServeHTTP(res, req)

		if res.Code != tc.httpStatus {
			t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
		}

		test.CompareWithResult(t, res, tc.expectedResponse)
	}
}

func TestCreateMeteringReportConfigEndpoint(t *testing.T) {
	t.Parallel()

	testSeed := test.GenTestSeed(func(seed *kubermaticv1.Seed) {
		seed.Spec.Metering = &kubermaticv1.MeteringConfiguration{
			Enabled:          true,
			StorageClassName: "test",
			StorageSize:      "10Gi",
			ReportConfigurations: map[string]*kubermaticv1.MeteringReportConfiguration{
				"weekly": {
					Schedule: "0 1 * * 6",
					Interval: 7,
				},
			},
		}
	})

	testcases := []struct {
		name                   string
		reportName             string
		body                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingAPIUser        *apiv1.User
		httpStatus             int
		expectedResponse       string
	}{
		// scenario 1
		{
			name:       "Create new metering report configuration.",
			reportName: "monthly",
			body: `{
				"interval": 30,
				"schedule": "1 1 1 * *",
				"retention": 60
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusCreated,
			expectedResponse:       `{}`,
		},
		// scenario 2
		{
			name:       "Create new metering report configuration. Missing name.",
			reportName: "",
			body: `{
				"interval": 30,
				"schedule": "1 1 1 * *",
				"retention": 60
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusMethodNotAllowed,
			expectedResponse:       ``,
		},
		// scenario 3
		{
			name:       "Create new metering report configuration. Missing interval.",
			reportName: "monthly",
			body: `{
				"schedule": "1 1 1 * *"
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"interval value cannot be smaller than 1."}}`,
		},
		// scenario 4
		{
			name:       "Create new metering report configuration. Incorrect schedule.",
			reportName: "monthly",
			body: `{
				"interval": 30,
				"schedule": "X 1 1 * *"
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"invalid cron expression format: X 1 1 * *"}}`,
		},
		// scenario 5
		{
			name:       "Create existing metering report configuration.",
			reportName: "weekly",
			body: `{
				"interval": 7,
				"schedule": "1 1 * * *"
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusConflict,
			expectedResponse:       `{"error":{"code":409,"message":"report configuration \"weekly\" already exists"}}`,
		},
		// scenario 6
		{
			name:       "Create new metering report configuration. Invalid name.",
			reportName: "invalid_name_",
			body: `{
				"interval": 30,
				"schedule": "1 1 1 * *"
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"metering report configuration name can contain only alphanumeric characters or '-'"}}`,
		},
		// scenario 7
		{
			name:       "Create new metering report configuration. Invalid retention.",
			reportName: "monthly",
			body: `{
				"interval": 30,
				"schedule": "1 1 1 * *",
				"retention": -1
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"retention value cannot be smaller than 1."}}`,
		},
	}

	for _, tc := range testcases {
		reqURL := "/api/v1/admin/metering/configurations/reports"
		if tc.reportName != "" {
			reqURL += "/" + tc.reportName
		}
		req := httptest.NewRequest(http.MethodPost, reqURL, strings.NewReader(tc.body))
		res := httptest.NewRecorder()

		router, err := test.CreateTestEndpoint(*tc.existingAPIUser, nil, tc.existingKubermaticObjs, nil, hack.NewTestRouting)
		if err != nil {
			t.Fatalf("failed to create test endpoint")
		}
		router.ServeHTTP(res, req)

		if res.Code != tc.httpStatus {
			t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
		}

		test.CompareWithResult(t, res, tc.expectedResponse)
	}
}

func TestUpdateMeteringReportConfigEndpoint(t *testing.T) {
	t.Parallel()

	var retention uint32 = 30
	testSeed := test.GenTestSeed(func(seed *kubermaticv1.Seed) {
		seed.Spec.Metering = &kubermaticv1.MeteringConfiguration{
			Enabled:          true,
			StorageClassName: "test",
			StorageSize:      "10Gi",
			ReportConfigurations: map[string]*kubermaticv1.MeteringReportConfiguration{
				"weekly": {
					Schedule:  "0 1 * * 6",
					Interval:  7,
					Retention: &retention,
				},
			},
		}
	})

	testcases := []struct {
		name                   string
		reportName             string
		body                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingAPIUser        *apiv1.User
		httpStatus             int
		expectedResponse       string
	}{
		// scenario 1
		{
			name:       "Update existing metering report configuration.",
			reportName: "weekly",
			body: `{
				"interval": 30,
				"schedule": "1 1 1 * *",
				"retention": 180
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusCreated,
			expectedResponse:       `{}`,
		},
		// scenario 2
		{
			name:       "Update existing metering report configuration. Invalid schedule.",
			reportName: "weekly",
			body: `{
				"schedule": "X 1 1 * *"
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"invalid cron expression format: X 1 1 * *"}}`,
		},
		// scenario 3
		{
			name:       "Update existing metering report configuration. Invalid interval.",
			reportName: "weekly",
			body: `{
				"interval": -1
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"interval value cannot be smaller than 1."}}`,
		},
		// scenario 4
		{
			name:       "Update existing metering report configuration. Invalid retention.",
			reportName: "weekly",
			body: `{
				"retention": -2
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"retention value cannot be smaller than 1."}}`,
		},
		// scenario 5
		{
			name:       "Update existing metering report configuration. Invalid name.",
			reportName: "invalid_name_",
			body: `{
				"interval": 30,
				"schedule": "1 1 1 * *"
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusBadRequest,
			expectedResponse:       `{"error":{"code":400,"message":"metering report configuration name can contain only alphanumeric characters or '-'"}}`,
		},
		// scenario 6
		{
			name:       "Update non-existing metering report configuration.",
			reportName: "monthly",
			body: `{
				"interval": 31
			}`,
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusNotFound,
			expectedResponse:       `{"error":{"code":404,"message":"report configuration \"monthly\" does not exists"}}`,
		},
	}

	for _, tc := range testcases {
		reqURL := fmt.Sprintf("/api/v1/admin/metering/configurations/reports/%s", tc.reportName)
		req := httptest.NewRequest(http.MethodPut, reqURL, strings.NewReader(tc.body))
		res := httptest.NewRecorder()

		router, err := test.CreateTestEndpoint(*tc.existingAPIUser, nil, tc.existingKubermaticObjs, nil, hack.NewTestRouting)
		if err != nil {
			t.Fatalf("failed to create test endpoint")
		}
		router.ServeHTTP(res, req)

		if res.Code != tc.httpStatus {
			t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
		}

		test.CompareWithResult(t, res, tc.expectedResponse)
	}
}

func TestDeleteMeteringReportConfigEndpoint(t *testing.T) {
	t.Parallel()

	testSeed := test.GenTestSeed(func(seed *kubermaticv1.Seed) {
		seed.Spec.Metering = &kubermaticv1.MeteringConfiguration{
			Enabled:          true,
			StorageClassName: "test",
			StorageSize:      "10Gi",
			ReportConfigurations: map[string]*kubermaticv1.MeteringReportConfiguration{
				"weekly": {
					Schedule: "0 1 * * 6",
					Interval: 7,
				},
			},
		}
	})

	testcases := []struct {
		name                   string
		reportName             string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingAPIUser        *apiv1.User
		httpStatus             int
		expectedResponse       string
	}{
		// scenario 1
		{
			name:                   "Delete existing metering report configuration.",
			reportName:             "weekly",
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusOK,
			expectedResponse:       `{}`,
		},
		// scenario 2
		{
			name:                   "Delete non-existing metering report configuration.",
			reportName:             "monthly",
			existingKubermaticObjs: []ctrlruntimeclient.Object{testSeed},
			existingAPIUser:        test.GenDefaultAdminAPIUser(),
			httpStatus:             http.StatusOK,
			expectedResponse:       `{}`,
		},
	}

	for _, tc := range testcases {
		reqURL := fmt.Sprintf("/api/v1/admin/metering/configurations/reports/%s", tc.reportName)
		req := httptest.NewRequest(http.MethodDelete, reqURL, strings.NewReader(""))
		res := httptest.NewRecorder()

		router, err := test.CreateTestEndpoint(*tc.existingAPIUser, nil, tc.existingKubermaticObjs, nil, hack.NewTestRouting)
		if err != nil {
			t.Fatalf("failed to create test endpoint")
		}
		router.ServeHTTP(res, req)

		if res.Code != tc.httpStatus {
			t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
		}

		test.CompareWithResult(t, res, tc.expectedResponse)
	}
}
