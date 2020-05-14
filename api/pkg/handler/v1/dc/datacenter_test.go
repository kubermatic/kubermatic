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

package dc_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestDatacentersListEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to list dc without email filtering",
			expectedResponse: `[{"metadata":{"name":"audited-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","node":{},"enforceAuditLogging":true}},{"metadata":{"name":"fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","node":{},"enforceAuditLogging":false}},{"metadata":{"name":"node-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false}},{"metadata":{"name":"private-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false}},{"metadata":{"name":"regular-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false}},{"metadata":{"name":"restricted-fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},{"metadata":{"name":"restricted-fake-dc2","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}},{"metadata":{"name":"us-central1","resourceVersion":"1"},"spec":{"seed":"","node":{},"enforceAuditLogging":false},"seed":true}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should be able to list dc with email filtering",
			expectedResponse: `[{"metadata":{"name":"audited-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","node":{},"enforceAuditLogging":true}},{"metadata":{"name":"fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","node":{},"enforceAuditLogging":false}},{"metadata":{"name":"node-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false}},{"metadata":{"name":"private-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false}},{"metadata":{"name":"regular-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false}},{"metadata":{"name":"us-central1","resourceVersion":"1"},"spec":{"seed":"","node":{},"enforceAuditLogging":false},"seed":true}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/dc", nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected route to return code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestDatacenterGetEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		dc               string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to get email restricted dc",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should not be able to get restricted dc if his email domain is restricted",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"error":{"code":404,"message":"datacenter \"restricted-fake-dc\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "regular user should be able to get restricted dc if his email domain is allowed",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			name:             "should get 404 for non-existent dc",
			dc:               "idontexist",
			expectedResponse: `{"error":{"code":404,"message":"datacenter \"idontexist\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should find dc",
			dc:               "regular-do1",
			expectedResponse: `{"metadata":{"name":"regular-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/dc/%s", tc.dc), nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected route to return code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestDatacenterListForProviderEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		provider         string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to list dc per provider without email filtering",
			provider:         "fake",
			expectedResponse: `[{"metadata":{"name":"audited-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","node":{},"enforceAuditLogging":true}},{"metadata":{"name":"fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","node":{},"enforceAuditLogging":false}},{"metadata":{"name":"node-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false}},{"metadata":{"name":"restricted-fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false}},{"metadata":{"name":"restricted-fake-dc2","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should be able to list dc per provider with email filtering",
			provider:         "fake",
			expectedResponse: `[{"metadata":{"name":"audited-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","node":{},"enforceAuditLogging":true}},{"metadata":{"name":"fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","node":{},"enforceAuditLogging":false}},{"metadata":{"name":"node-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false}}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should receive empty list for non-existent provider",
			provider:         "idontexist",
			expectedResponse: `[]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/providers/%s/dc", tc.provider), nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected route to return code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestDatacenterGetForProviderEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		provider         string
		dc               string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to get email restricted dc",
			provider:         "fake",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should not be able to get restricted dc if his email domain is restricted",
			provider:         "fake",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"error":{"code":404,"message":"datacenter \"restricted-fake-dc\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "regular user should be able to get restricted dc if his email domain is allowed",
			provider:         "fake",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			name:             "should get 404 for non-existent dc",
			provider:         "fake",
			dc:               "idontexist",
			expectedResponse: `{"error":{"code":404,"message":"datacenter \"idontexist\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should get 404 for non-existent provider",
			provider:         "idontexist",
			dc:               "regular-do1",
			expectedResponse: `{"error":{"code":404,"message":"datacenter \"regular-do1\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should find dc",
			provider:         "digitalocean",
			dc:               "regular-do1",
			expectedResponse: `{"metadata":{"name":"regular-do1","resourceVersion":"1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/providers/%s/dc/%s", tc.provider, tc.dc), nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected route to return code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}
