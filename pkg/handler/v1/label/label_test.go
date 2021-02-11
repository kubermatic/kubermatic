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

package label_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/handler/v1/label"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestListSystemLabels(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		ExpectedLabels  apiv1.ResourceLabelMap
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:            "scenario 1: list system labels",
			ExpectedLabels:  label.GetSystemLabels(),
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/labels/system", strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []ctrlruntimeclient.Object{}
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actual := decodeResourceLabelMap(res.Body, t)
			expected := tc.ExpectedLabels

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("actual map is different that the expected one. Expected: %v, got %v", expected, actual)
			}
		})
	}
}

func decodeResourceLabelMap(r io.Reader, t *testing.T) apiv1.ResourceLabelMap {
	var res apiv1.ResourceLabelMap
	dec := json.NewDecoder(r)
	err := dec.Decode(&res)
	if err != nil {
		t.Fatal(err)
	}

	return res
}
