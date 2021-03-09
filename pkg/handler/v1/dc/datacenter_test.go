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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

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
			expectedResponse: `[{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"registry_mirrors":["http://127.0.0.1:5001"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pause_image":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should be able to list dc with email filtering",
			expectedResponse: `[{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"registry_mirrors":["http://127.0.0.1:5001"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pause_image":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}]`,
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
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
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
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
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
			expectedResponse: `{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
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
			expectedResponse: `[{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"registry_mirrors":["http://127.0.0.1:5001"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should be able to list dc per provider with email filtering",
			provider:         "fake",
			expectedResponse: `[{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"registry_mirrors":["http://127.0.0.1:5001"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}}]`,
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
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
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
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
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
			expectedResponse: `{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
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

func TestDatacenterListForSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		seed             string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to list dc per seed without email filtering",
			seed:             "us-central1",
			expectedResponse: `[{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"registry_mirrors":["http://127.0.0.1:5001"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pause_image":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomains":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should be able to list dc per seed with email filtering",
			seed:             "us-central1",
			expectedResponse: `[{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henriks basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"http_proxy":"HTTPProxy","insecure_registries":["incsecure-registry"],"registry_mirrors":["http://127.0.0.1:5001"],"pause_image":"pause-image","hyperkube_image":"hyperkube-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}},{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pause_image":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}},{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should receive 404 for non-existing seed",
			seed:             "idontexist",
			expectedResponse: `{"error":{"code":404,"message":"seed \"idontexist\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/seed/%s/dc", tc.seed), nil)
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

func TestDatacenterGetForSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		seed             string
		dc               string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to get email restricted dc",
			seed:             "us-central1",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should not be able to get restricted dc if his email domain is restricted",
			seed:             "us-central1",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"error":{"code":404,"message":"datacenter \"restricted-fake-dc\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "regular user should be able to get restricted dc if his email domain is allowed",
			seed:             "us-central1",
			dc:               "restricted-fake-dc",
			expectedResponse: `{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmailDomain":"example.com","enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			name:             "should get 404 for non-existent dc",
			seed:             "us-central1",
			dc:               "idontexist",
			expectedResponse: `{"error":{"code":404,"message":"datacenter \"idontexist\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should get 404 for non-existent provider",
			seed:             "idontexist",
			dc:               "regular-do1",
			expectedResponse: `{"error":{"code":404,"message":"seed \"idontexist\" not found"}}`,
			httpStatus:       404,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should find dc",
			seed:             "us-central1",
			dc:               "regular-do1",
			expectedResponse: `{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/seed/%s/dc/%s", tc.seed, tc.dc), nil)
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

func TestDatacenterCreateEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		dcSpec           apiv1.DatacenterSpec
		dcName           string
		seedName         string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name: "admin should be able to create dc",
			dcSpec: apiv1.DatacenterSpec{
				Seed:         "us-central1",
				Country:      "NL",
				Location:     "Amsterdam",
				Digitalocean: &v1.DatacenterSpecDigitalocean{},
			},
			dcName:           "do-correct",
			seedName:         "us-central1",
			expectedResponse: `{"metadata":{"name":"do-correct"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","digitalocean":{"region":""},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       201,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "non-admin should not be able to create dc",
			dcSpec: apiv1.DatacenterSpec{
				Seed:         "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{},
			},
			dcName:           "do-correct",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:       403,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name: "should not be able to create already existing dc",
			dcSpec: apiv1.DatacenterSpec{
				Seed:         "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{},
			},
			dcName:           "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: datacenter \"private-do1\" already exists"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should not be able to create a dc in non existing seed",
			dcSpec: apiv1.DatacenterSpec{
				Seed:         "idontexist",
				Digitalocean: &v1.DatacenterSpecDigitalocean{},
			},
			dcName:           "private-do1",
			seedName:         "idontexist",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: seed \"idontexist\" does not exist"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should not be able to create a dc with no specified provider",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "us-central1",
			},
			dcName:           "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Validation error: one DC provider should be specified, got: []"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should not be able to create a dc with multiple specified providers",
			dcSpec: apiv1.DatacenterSpec{
				Seed:         "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{},
				AWS:          &v1.DatacenterSpecAWS{},
			},
			dcName:           "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Validation error: one DC provider should be specified, got: [digitalocean aws]"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should receive a validation error when providing different seed name in path and request",
			dcSpec: apiv1.DatacenterSpec{
				Seed:         "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{},
			},
			dcName:           "private-do1",
			seedName:         "different",
			expectedResponse: `{"error":{"code":400,"message":"Validation error: path seed \"different\" and request seed \"us-central1\" not equal"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var reqBody struct {
				Name string               `json:"name"`
				Spec apiv1.DatacenterSpec `json:"spec"`
			}
			reqBody.Spec = tc.dcSpec
			reqBody.Name = tc.dcName

			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("error marshalling body into json: %v", err)
			}
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/seed/%s/dc", tc.seedName), bytes.NewBuffer(body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser), test.GenTestSeed()}, nil, nil, hack.NewTestRouting)
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

func TestDatacenterUpdateEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		dcSpec           apiv1.DatacenterSpec
		dcName           string
		dcPathName       string
		seedName         string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name: "admin should be able to update dc",
			dcSpec: apiv1.DatacenterSpec{
				Seed:     "us-central1",
				Country:  "NL",
				Location: "Amsterdam",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
				Node: v1.NodeSettings{
					PauseImage: "pause-image",
				},
				EnforceAuditLogging:      true,
				EnforcePodSecurityPolicy: false,
				RequiredEmailDomains:     []string{"example.com", "pleaseno.org"},
			},
			dcName:           "private-do1",
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","digitalocean":{"region":"EU"},"node":{"pause_image":"pause-image"},"requiredEmailDomains":["example.com","pleaseno.org"],"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "admin should be able to update dc name",
			dcSpec: apiv1.DatacenterSpec{
				Seed:     "us-central1",
				Country:  "NL",
				Location: "Amsterdam",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
				Node: v1.NodeSettings{
					PauseImage: "pause-image",
				},
				EnforceAuditLogging:      true,
				EnforcePodSecurityPolicy: false,
				RequiredEmailDomains:     []string{"example.com", "pleaseno.org"},
			},
			dcName:           "private-do1-updated",
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"metadata":{"name":"private-do1-updated"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","digitalocean":{"region":"EU"},"node":{"pause_image":"pause-image"},"requiredEmailDomains":["example.com","pleaseno.org"],"enforceAuditLogging":true,"enforcePodSecurityPolicy":false}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "non-admin should not be able to update dc",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
			},
			dcName:           "do-correct",
			dcPathName:       "do-correct",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:       403,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name: "should not be able to update non existing dc",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
			},
			dcName:           "idontexist",
			dcPathName:       "idontexist",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: datacenter \"idontexist\" does not exists"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should not be able to update dc name to existing dc",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
			},
			dcName:           "private-do1",
			dcPathName:       "regular-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: cannot change \"regular-do1\" datacenter name to \"private-do1\" as it already exists"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should not be able to update a dc in non existing seed",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "idontexist",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
			},
			dcName:           "private-do1",
			dcPathName:       "private-do1",
			seedName:         "idontexist",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: seed \"idontexist\" does not exist"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should not be able to update a dc with no specified provider",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "us-central1",
			},
			dcName:           "private-do1",
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Validation error: one DC provider should be specified, got: []"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should not be able to update a dc with multiple specified providers",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
				AWS: &v1.DatacenterSpecAWS{
					Region: "EU",
				},
			},
			dcName:           "private-do1",
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Validation error: one DC provider should be specified, got: [digitalocean aws]"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name: "should receive a validation error when providing different seed name in path and request",
			dcSpec: apiv1.DatacenterSpec{
				Seed: "us-central1",
				Digitalocean: &v1.DatacenterSpecDigitalocean{
					Region: "EU",
				},
			},
			dcName:           "private-do1",
			dcPathName:       "private-do1",
			seedName:         "different",
			expectedResponse: `{"error":{"code":400,"message":"Validation error: path seed \"different\" and request seed \"us-central1\" not equal"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var reqBody struct {
				Name string               `json:"name"`
				Spec apiv1.DatacenterSpec `json:"spec"`
			}
			reqBody.Spec = tc.dcSpec
			reqBody.Name = tc.dcName
			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("error marshalling body into json: %v", err)
			}
			req := httptest.NewRequest("PUT",
				fmt.Sprintf("/api/v1/seed/%s/dc/%s", tc.seedName, tc.dcPathName), bytes.NewBuffer(body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser), test.GenTestSeed()}, nil, nil, hack.NewTestRouting)
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

func TestDatacenterPatchEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		patch            string
		dcPathName       string
		seedName         string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to update dc",
			patch:            `{"metadata":{"name":"private-do1"},"spec":{"country":"NL","location":"Amsterdam","aws":{"region":"EU"},"digitalocean":null,"node":null,"enforceAuditLogging":false}}`,
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"aws","aws":{"region":"EU","images":null},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "admin should be able to update dc name",
			patch:            `{"metadata":{"name":"private-do1-updated"}}`,
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"metadata":{"name":"private-do1-updated"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pause_image":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true}}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "non-admin should not be able to update dc",
			patch:            `{"metadata": {"name": "private-do1-updated"}}`,
			dcPathName:       "do-correct",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:       403,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should not be able to update non existing dc",
			patch:            `{"metadata": {"name": "private-do1-updated"}}`,
			dcPathName:       "idontexist",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: datacenter \"idontexist\" does not exists"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "should not be able to update dc name to existing dc",
			patch:            `{"metadata": {"name": "private-do1"}}`,
			dcPathName:       "regular-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: cannot change \"regular-do1\" datacenter name to \"private-do1\" as it already exists"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "should not be able to update a dc in non existing seed",
			patch:            `{"metadata": {"name": "private-do1-updated"}}`,
			dcPathName:       "private-do1",
			seedName:         "idontexist",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: seed \"idontexist\" does not exist"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "should not be able to update a dc with no specified provider",
			patch:            `{"spec":{"digitalocean":null}}`,
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"patched dc validation failed: one DC provider should be specified, got: []"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "should not be able to update a dc with multiple specified providers",
			patch:            `{"spec":{"digitalocean":{"region":"EU"}, "aws":{"region":"US"}}}`,
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"patched dc validation failed: one DC provider should be specified, got: [digitalocean aws]"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "should receive a validation error when setting a different seed then the path seed in the patch request",
			patch:            `{"spec":{"seed":"different"}}`,
			dcPathName:       "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"patched dc validation failed: path seed name \"us-central1\" has to be equal to patch seed name \"different\""}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH",
				fmt.Sprintf("/api/v1/seed/%s/dc/%s", tc.seedName, tc.dcPathName), strings.NewReader(tc.patch))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser), test.GenTestSeed()}, nil, nil, hack.NewTestRouting)
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

func TestDatacenterDeleteEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		dcName           string
		seedName         string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to delete dc",
			dcName:           "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{}`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "non-admin should not be able to delete dc",
			dcName:           "private-do1",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:       403,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			name:             "should receive error when deleting non-existing dc",
			dcName:           "idontexist",
			seedName:         "us-central1",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: datacenter \"idontexist\" does not exists"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "should receive error when deleting dc in non-existing seed",
			dcName:           "private-do1",
			seedName:         "idontexist",
			expectedResponse: `{"error":{"code":400,"message":"Bad request: seed \"idontexist\" does not exist"}}`,
			httpStatus:       400,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE",
				fmt.Sprintf("/api/v1/seed/%s/dc/%s", tc.seedName, tc.dcName), nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []runtime.Object{},
				[]runtime.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser), test.GenTestSeed()}, nil, nil, hack.NewTestRouting)
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
