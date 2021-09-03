/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package backupcredentials_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateOrUpdateEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		SeedName                  string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingKubeObjects       []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		BackupCredentials         *apiv2.BackupCredentials
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:     "create backup credentials for given seed",
			SeedName: test.GenTestSeed().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			BackupCredentials:      test.GenDefaultAPIBackupCredentials(),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
		{
			Name:     "non-admin user cannot create backup credentials",
			SeedName: test.GenTestSeed().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			BackupCredentials:      test.GenDefaultAPIBackupCredentials(),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:     "can't manage backup credentials for non-existing seed",
			SeedName: "nothere",
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			BackupCredentials:      test.GenDefaultAPIBackupCredentials(),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
		{
			Name:     "Update backup credentials for given seed",
			SeedName: test.GenTestSeed().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingKubeObjects: []ctrlruntimeclient.Object{
				test.GenDefaultBackupCredentials(),
			},
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			BackupCredentials:      test.GenDefaultAPIBackupCredentials(),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/seeds/%s/backupcredentials", tc.SeedName)
			body, err := json.Marshal(tc.BackupCredentials)
			if err != nil {
				t.Fatalf("failed marshalling backupcredentials: %v", err)
			}
			req := httptest.NewRequest(http.MethodPut, requestURL, bytes.NewBuffer(body))
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, tc.ExistingKubeObjects, tc.ExistingKubermaticObjects, nil, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
		})
	}
}
