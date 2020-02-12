package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestListAdmissionPluginEndpoint(t *testing.T) {
	t.Parallel()
	version113, err := semver.NewSemver("v1.13")
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user gets plugins",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user gets empty list",
			expectedResponse:       `[]`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user gets plugin list",
			expectedResponse: `[{"name":"defaultTolerationSeconds","plugin":"DefaultTolerationSeconds"},{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}]`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true),
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				}},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("GET", "/api/v1/admin/admission/plugins", strings.NewReader(""))
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

func TestGetAdmissionPluginEndpoint(t *testing.T) {
	t.Parallel()
	version113, err := semver.NewSemver("v1.13")
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		name                   string
		plugin                 string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user gets plugins",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":404,"message":" \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user gets plugin",
			plugin:           "eventRateLimit",
			expectedResponse: ` {"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true),
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				}},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/admission/plugins/%s", tc.plugin), strings.NewReader(""))
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

func TestDeleteAdmissionPluginEndpoint(t *testing.T) {
	t.Parallel()
	version113, err := semver.NewSemver("v1.13")
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		name                   string
		plugin                 string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user delete plugin",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":404,"message":" \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user delete plugin",
			plugin:           "eventRateLimit",
			expectedResponse: ` {}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true),
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				}},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/admission/plugins/%s", tc.plugin), strings.NewReader(""))
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

func TestUpdateAdmissionPluginEndpoint(t *testing.T) {
	t.Parallel()
	version113, err := semver.NewSemver("v1.13")
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		name                   string
		plugin                 string
		body                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user update plugin",
			plugin:                 "eventRateLimit",
			body:                   `{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}`,
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []runtime.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			plugin:                 "eventRateLimit",
			body:                   `{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}`,
			expectedResponse:       `{"error":{"code":404,"message":" \"eventRateLimit\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user updates plugin name",
			plugin:           "eventRateLimit",
			body:             `{"name":"eventRateLimit","plugin":"NewEventRateLimit","fromVersion":"1.13.0"}`,
			expectedResponse: `{"name":"eventRateLimit","plugin":"NewEventRateLimit","fromVersion":"1.13.0"}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true),
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				}},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 4
		{
			name:             "scenario 4: authorized user updates plugin version",
			plugin:           "eventRateLimit",
			body:             `{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.15.2"}`,
			expectedResponse: `{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.15.2"}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true),
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				}},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/admin/admission/plugins/%s", tc.plugin), strings.NewReader(tc.body))
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
