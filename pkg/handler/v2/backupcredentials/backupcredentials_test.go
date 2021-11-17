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
	v1 "k8s.io/api/core/v1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
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
		{
			Name:     "create backup credentials for given seed with destination",
			SeedName: test.GenTestSeed().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(func(seed *kubermaticv1.Seed) {
					seed.Spec.EtcdBackupRestore = &kubermaticv1.EtcdBackupRestore{
						Destinations: map[string]*kubermaticv1.BackupDestination{
							"s3": {
								Endpoint:   "aws.s3.com",
								BucketName: "testbucket",
							},
						},
					}
				}),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			BackupCredentials: func() *apiv2.BackupCredentials {
				cred := test.GenDefaultAPIBackupCredentials()
				cred.Destination = "s3"
				return cred
			}(),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
		{
			Name:     "update backup credentials for given seed with destination",
			SeedName: test.GenTestSeed().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(func(seed *kubermaticv1.Seed) {
					seed.Spec.EtcdBackupRestore = &kubermaticv1.EtcdBackupRestore{
						Destinations: map[string]*kubermaticv1.BackupDestination{
							"s3": {
								Endpoint:   "aws.s3.com",
								BucketName: "testbucket",
								Credentials: &v1.SecretReference{
									Name:      "secret",
									Namespace: "kubermatic",
								},
							},
						},
					}
				}),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			BackupCredentials: func() *apiv2.BackupCredentials {
				cred := test.GenDefaultAPIBackupCredentials()
				cred.Destination = "s3"
				return cred
			}(),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
		{
			Name:     "can't manage backup credentials for non-existing seed destination",
			SeedName: "nothere",
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			BackupCredentials: func() *apiv2.BackupCredentials {
				cred := test.GenDefaultAPIBackupCredentials()
				cred.Destination = "s3"
				return cred
			}(),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			creds := struct {
				BackupCredentials *apiv2.BackupCredentials `json:"backup_credentials"`
			}{
				BackupCredentials: tc.BackupCredentials,
			}
			requestURL := fmt.Sprintf("/api/v2/seeds/%s/backupcredentials", tc.SeedName)
			body, err := json.Marshal(creds)
			if err != nil {
				t.Fatalf("failed marshalling backupcredentials: %v", err)
			}
			req := httptest.NewRequest(http.MethodPut, requestURL, bytes.NewBuffer(body))
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, tc.ExistingKubeObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
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
