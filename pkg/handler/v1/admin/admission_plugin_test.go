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

package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/semver"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user gets plugins",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user gets empty list",
			expectedResponse:       `[]`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user gets plugin list",
			expectedResponse: `[{"name":"defaultTolerationSeconds","plugin":"DefaultTolerationSeconds"},{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}]`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true),
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
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("GET", "/api/v1/admin/admission/plugins", strings.NewReader(""))
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
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user gets plugins",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":404,"message":" \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user gets plugin",
			plugin:           "eventRateLimit",
			expectedResponse: ` {"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true),
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
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/admission/plugins/%s", tc.plugin), strings.NewReader(""))
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
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user delete plugin",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			plugin:                 "test",
			expectedResponse:       `{"error":{"code":404,"message":" \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user delete plugin",
			plugin:           "eventRateLimit",
			expectedResponse: ` {}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true),
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
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/admission/plugins/%s", tc.plugin), strings.NewReader(""))
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
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user update plugin",
			plugin:                 "eventRateLimit",
			body:                   `{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}`,
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			plugin:                 "eventRateLimit",
			body:                   `{"name":"eventRateLimit","plugin":"EventRateLimit","fromVersion":"1.13.0"}`,
			expectedResponse:       `{"error":{"code":404,"message":" \"eventRateLimit\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user updates plugin name",
			plugin:           "eventRateLimit",
			body:             `{"name":"eventRateLimit","plugin":"NewEventRateLimit","fromVersion":"1.13.0"}`,
			expectedResponse: `{"name":"eventRateLimit","plugin":"NewEventRateLimit","fromVersion":"1.13.0"}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true),
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
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true),
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
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/admin/admission/plugins/%s", tc.plugin), strings.NewReader(tc.body))
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
