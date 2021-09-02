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

package seed_test

import (
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSeedNamesListEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		expectedResponse string
		httpStatus       int
		existingAPIUser  *apiv1.User
	}{
		{
			name:             "admin should be able to list seed names",
			expectedResponse: `["us-central1"]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			name:             "regular user should be able to list seed names",
			expectedResponse: `["us-central1"]`,
			httpStatus:       200,
			existingAPIUser:  test.GenDefaultAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/seed", nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.existingAPIUser, []ctrlruntimeclient.Object{},
				[]ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.existingAPIUser), test.GenTestSeed()}, nil, nil, nil, hack.NewTestRouting)
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
