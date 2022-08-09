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
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestListSeedsEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user gets seeds",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user gets default list",
			expectedResponse:       `[{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"fake-dc":{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henrik's basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"node-dc":{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"httpProxy":"HTTPProxy","insecureRegistries":["incsecure-registry"],"registryMirrors":["http://127.0.0.1:5001"],"pauseImage":"pause-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"private-do1":{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pauseImage":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":true}},"psp-dc":{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":false}},"regular-do1":{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":true}},"restricted-fake-dc":{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["example.com"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"restricted-fake-dc2":{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}}}}}]`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/seeds", strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestGetSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		seedName               string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user gets seeds",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":404,"message":"Seed \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user gets seed",
			seedName:         "us-central1",
			expectedResponse: `{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"fake-dc":{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henrik's basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"node-dc":{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"httpProxy":"HTTPProxy","insecureRegistries":["incsecure-registry"],"registryMirrors":["http://127.0.0.1:5001"],"pauseImage":"pause-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"private-do1":{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pauseImage":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":true}},"psp-dc":{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":false}},"regular-do1":{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":true}},"restricted-fake-dc":{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["example.com"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"restricted-fake-dc2":{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}}}}}`,
			httpStatus:       http.StatusOK, existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/seeds/%s", tc.seedName), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestUpdateSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		body                   string
		seedName               string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user updates seeds",
			body:                   `{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henrik's basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["example.com"],"enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}`,
			seedName:               "us-central1",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: not found",
			body:                   `{"name":"test","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henrik's basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["example.com"],"enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}`,
			seedName:               "test",
			expectedResponse:       `{"error":{"code":404,"message":"Seed \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                   "scenario 3: seed name mismatch",
			body:                   `{"name":"central1","spec":{"country":"US","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true}},"fake-dc":{"country":"Germany","location":"Henrik's basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["example.com"],"enforceAuditLogging":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false}}}}}`,
			seedName:               "us-central1",
			expectedResponse:       `{"error":{"code":400,"message":"seed name mismatch, you requested to update Seed \"us-central1\" but body contains Seed \"central1\""}}`,
			httpStatus:             http.StatusBadRequest,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 4
		{
			name:                   "scenario 4: authorized user updates seed",
			body:                   `{"name":"us-central1","spec":{"country":"NL","kubeconfig":{},"datacenters":{"audited-dc":{"country":"Germany","location":"Finanzamt Castle","node":{},"spec":{"fake":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"fake-dc":{"country":"Germany","location":"Henrik's basement","node":{},"spec":{"fake":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"private-do1":{"country":"NL","location":"US ","node":{},"spec":{"digitalocean":{"region":"ams3"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":true}},"regular-do1":{"country":"NL","location":"Amsterdam","node":{},"spec":{"digitalocean":{"region":"ams2"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"restricted-fake-dc":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["example.com"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"restricted-fake-dc2":{"country":"NL","location":"Amsterdam","node":{},"spec":{"fake":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}}}}}`,
			seedName:               "us-central1",
			expectedResponse:       `{"name":"us-central1","spec":{"country":"NL","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"fake-dc":{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henrik's basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"node-dc":{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"httpProxy":"HTTPProxy","insecureRegistries":["incsecure-registry"],"registryMirrors":["http://127.0.0.1:5001"],"pauseImage":"pause-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"private-do1":{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams3"},"node":{"pauseImage":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":true}},"psp-dc":{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":false}},"regular-do1":{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":true}},"restricted-fake-dc":{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["example.com"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"restricted-fake-dc2":{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}}}}}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 6
		{
			name:                   "scenario 6: authorized user updates seed - just backup destinations",
			body:                   `{"name":"us-central1","spec":{"country":"NL","kubeconfig":{},"etcdBackupRestore":{"destinations":{"s3":{"bucketName":"bucket","endpoint":"endpoint"}},"defaultDestination":"s3"}}}}`,
			seedName:               "us-central1",
			expectedResponse:       `{"name":"us-central1","spec":{"country":"NL","location":"us-central","kubeconfig":{},"datacenters":{"audited-dc":{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"fake-dc":{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henrik's basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"node-dc":{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"httpProxy":"HTTPProxy","insecureRegistries":["incsecure-registry"],"registryMirrors":["http://127.0.0.1:5001"],"pauseImage":"pause-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"private-do1":{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pauseImage":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":true}},"psp-dc":{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":false}},"regular-do1":{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":true}},"restricted-fake-dc":{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["example.com"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"restricted-fake-dc2":{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}}},"etcdBackupRestore":{"destinations":{"s3":{"endpoint":"endpoint","bucketName":"bucket"}},"defaultDestination":"s3"}}}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 7
		{
			name:                   "scenario 7: authorized user updates kubeconfig",
			body:                   `{"name":"us-central1","raw_kubeconfig":"YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"}`,
			seedName:               "us-central1",
			expectedResponse:       `{"name":"us-central1","spec":{"country":"US","location":"us-central","kubeconfig":{"namespace":"kubermatic","name":"kubeconfig-us-central1","resourceVersion":"1"},"datacenters":{"audited-dc":{"metadata":{"name":"audited-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Finanzamt Castle","provider":"fake","fake":{},"node":{},"enforceAuditLogging":true,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"fake-dc":{"metadata":{"name":"fake-dc"},"spec":{"seed":"us-central1","country":"Germany","location":"Henrik's basement","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"node-dc":{"metadata":{"name":"node-dc"},"spec":{"seed":"us-central1","country":"Chile","location":"Santiago","provider":"fake","fake":{},"node":{"httpProxy":"HTTPProxy","insecureRegistries":["incsecure-registry"],"registryMirrors":["http://127.0.0.1:5001"],"pauseImage":"pause-image"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"private-do1":{"metadata":{"name":"private-do1"},"spec":{"seed":"us-central1","country":"NL","location":"US ","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{"pauseImage":"image-pause"},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":true}},"psp-dc":{"metadata":{"name":"psp-dc"},"spec":{"seed":"us-central1","country":"Egypt","location":"Alexandria","provider":"fake","fake":{},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":true,"ipv6Enabled":false}},"regular-do1":{"metadata":{"name":"regular-do1"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"digitalocean","digitalocean":{"region":"ams2"},"node":{},"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":true}},"restricted-fake-dc":{"metadata":{"name":"restricted-fake-dc"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["example.com"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}},"restricted-fake-dc2":{"metadata":{"name":"restricted-fake-dc2"},"spec":{"seed":"us-central1","country":"NL","location":"Amsterdam","provider":"fake","fake":{},"node":{},"requiredEmails":["23f67weuc.com","example.com","12noifsdsd.org"],"enforceAuditLogging":false,"enforcePodSecurityPolicy":false,"ipv6Enabled":false}}}}}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/seeds/%s", tc.seedName), strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestDeleteSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		seedName               string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user tries to delete seed cluster",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user tries to delete not existing seed cluster",
			seedName:               "test",
			expectedResponse:       `{"error":{"code":404,"message":"Seed \"test\" not found"}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                   "scenario 3: authorized user tries to delete seed cluster",
			seedName:               "us-central1",
			expectedResponse:       `{}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/seeds/%s", tc.seedName), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestCreateSeedEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		body                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user tries to create seed",
			body:                   `{"name":"test","spec":{"kubeconfig":"YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"}}`,
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", false)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: admin can create a new seed",
			body:                   `{"name":"test","spec":{"kubeconfig":"YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"}}`,
			expectedResponse:       `{"name":"test","spec":{"kubeconfig":{"namespace":"kubermatic","name":"kubeconfig-test","resourceVersion":"1"}}}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/seeds", strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}
