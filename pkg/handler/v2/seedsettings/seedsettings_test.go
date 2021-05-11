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

package seedsettings_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetSeedSettingsEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name                   string
		seedName               string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []ctrlruntimeclient.Object
		expectedHTTPStatus     int
		expectedResponse       apiv2.SeedSettings
	}{
		{
			name:     "scenario 1: user can get seed settings with mla disabled",
			seedName: "us-central1",
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenDefaultUser(),
				test.GenTestSeed(),
			},
			existingAPIUser:    test.GenDefaultAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResponse: apiv2.SeedSettings{
				MLA: apiv2.SeedMLASettings{
					UserClusterMLAEnabled: false,
				},
			},
		},
		{
			name:     "scenario 2: user can get seed settings with mla enabled",
			seedName: "us-central1",
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenDefaultUser(),
				genMLASeed(true),
			},
			existingAPIUser:    test.GenDefaultAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResponse: apiv2.SeedSettings{
				MLA: apiv2.SeedMLASettings{
					UserClusterMLAEnabled: true,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, requestURL(tc.seedName), nil)
			resp := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, nil, tc.existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.expectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.expectedHTTPStatus, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusOK {
				b, err := json.Marshal(tc.expectedResponse)
				if err != nil {
					t.Fatalf("failed to marshall expected response %v", err)
				}
				fmt.Println(string(b))

				test.CompareWithResult(t, resp, string(b))
			}
		})
	}
}

func genMLASeed(mlaEnabled bool) *kubermaticv1.Seed {
	seed := test.GenTestSeed()
	seed.Spec.MLA = &kubermaticv1.SeedMLASettings{
		UserClusterMLAEnabled: mlaEnabled,
	}
	return seed
}

func requestURL(seedName string) string {
	return fmt.Sprintf("/api/v2/seeds/%s/settings", seedName)
}
