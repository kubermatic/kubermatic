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

package whitelistedregistry_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateWhitelistedRegistry(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		WRtoCreate       apiv2.WhitelistedRegistry
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
	}{
		{
			Name:             "scenario 1: admin can create whitelisted registry",
			WRtoCreate:       test.GenDefaultAPIWhitelistedRegistry("wr", "quay.io"),
			ExpectedResponse: `{"name":"wr","spec":{"registryPrefix":"quay.io"}}`,
			HTTPStatus:       http.StatusCreated,
			ExistingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			Name:             "scenario 2: non-admin can not create whitelisted registry",
			WRtoCreate:       test.GenDefaultAPIWhitelistedRegistry("wr", "quay.io"),
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var reqBody struct {
				Name string                               `json:"name"`
				Spec kubermaticv1.WhitelistedRegistrySpec `json:"spec"`
			}
			reqBody.Spec = tc.WRtoCreate.Spec
			reqBody.Name = tc.WRtoCreate.Name

			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("error marshalling body into json: %v", err)
			}
			req := httptest.NewRequest("POST", "/api/v2/whitelistedregistries", bytes.NewBuffer(body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestGetWhitelistedRegistry(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		WRName           string
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
		ExistingObjects  []ctrlruntimeclient.Object
	}{
		{
			Name:             "scenario 1: admin can get whitelisted registry",
			WRName:           "wr1",
			ExpectedResponse: `{"name":"wr1","spec":{"registryPrefix":"quay.io"}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: []ctrlruntimeclient.Object{
				test.GenWhitelistedRegistry("wr1", "quay.io"),
				test.GenWhitelistedRegistry("wr2", "docker.io"),
			},
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
		{
			Name:             "scenario 2: admin cannot get non-existing whitelisted registry",
			WRName:           "missing",
			ExpectedResponse: `{"error":{"code":404,"message":"whitelistedregistries.kubermatic.k8s.io \"missing\" not found"}} `,
			HTTPStatus:       http.StatusNotFound,
			ExistingObjects: []ctrlruntimeclient.Object{
				test.GenWhitelistedRegistry("wr1", "quay.io"),
				test.GenWhitelistedRegistry("wr2", "docker.io"),
			},
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
		{
			Name:             "scenario 3: non-admin can't get whitelisted registry",
			WRName:           "wr1",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingObjects: []ctrlruntimeclient.Object{
				test.GenWhitelistedRegistry("wr1", "quay.io"),
				test.GenWhitelistedRegistry("wr2", "docker.io"),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.ExistingObjects = append(tc.ExistingObjects, test.APIUserToKubermaticUser(*tc.ExistingAPIUser))

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/whitelistedregistries/%s", tc.WRName), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListWhitelistedRegistries(t *testing.T) {
	wr1 := test.GenDefaultAPIWhitelistedRegistry("wr1", "quay.io")
	wr2 := test.GenDefaultAPIWhitelistedRegistry("wr2", "docker.io")

	t.Parallel()
	testcases := []struct {
		Name                          string
		ExpectedWhitelistedRegistries []*apiv2.WhitelistedRegistry
		HTTPStatus                    int
		ExistingAPIUser               *apiv1.User
		ExistingObjects               []ctrlruntimeclient.Object
	}{
		{
			Name: "scenario 1: admin can list all default whitelisted registries",
			ExpectedWhitelistedRegistries: []*apiv2.WhitelistedRegistry{
				&wr1,
				&wr2,
			},
			HTTPStatus: http.StatusOK,
			ExistingObjects: []ctrlruntimeclient.Object{
				test.GenWhitelistedRegistry("wr1", "quay.io"),
				test.GenWhitelistedRegistry("wr2", "docker.io"),
			},
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.ExistingObjects = append(tc.ExistingObjects, test.APIUserToKubermaticUser(*tc.ExistingAPIUser))

			req := httptest.NewRequest("GET", "/api/v2/whitelistedregistries", strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualWRs := test.NewWhitelistedRegistrySliceWrapper{}
			actualWRs.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedCTs := test.NewWhitelistedRegistrySliceWrapper(tc.ExpectedWhitelistedRegistries)
			wrappedExpectedCTs.Sort()

			actualWRs.EqualOrDie(wrappedExpectedCTs, t)
		})
	}
}
