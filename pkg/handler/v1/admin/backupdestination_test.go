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

package admin_test

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

func TestDeleteBackupDestinationEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		backupDestinationName  string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: not authorized user tries to delete seed backup destination",
			backupDestinationName:  "test",
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: []ctrlruntimeclient.Object{test.GenAdminUser("Bob", "bob@acme.com", false), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user tries to delete not existing backup destination",
			backupDestinationName:  "non-existing",
			expectedResponse:       `{"error":{"code":404,"message":"backup destination \"non-existing\" does not exist for seed \"us-central1\""}}`,
			httpStatus:             http.StatusNotFound,
			existingKubermaticObjs: []ctrlruntimeclient.Object{test.GenAdminUser("Bob", "bob@acme.com", true), test.GenTestSeed()},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:                  "scenario 3: authorized user tries to delete backup destination",
			backupDestinationName: "s3",
			expectedResponse:      `{}`,
			httpStatus:            http.StatusOK,
			existingKubermaticObjs: []ctrlruntimeclient.Object{test.GenAdminUser("Bob", "bob@acme.com", true),
				test.GenTestSeed(func(seed *kubermaticv1.Seed) {
					seed.Spec.EtcdBackupRestore = &kubermaticv1.EtcdBackupRestore{Destinations: map[string]*kubermaticv1.BackupDestination{
						"s3": {
							BucketName: "s3",
							Endpoint:   "s3.com",
						},
						"minio": {
							BucketName: "minio",
							Endpoint:   "min.io",
						},
					}}
				})},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/seeds/%s/backupdestinations/%s",
				test.GenTestSeed().Name, tc.backupDestinationName), strings.NewReader(""))
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
