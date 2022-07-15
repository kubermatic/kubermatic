/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package applicationdefinition_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestListApplicationDefinitions(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name               string
		ExistingAPIUser    *apiv1.User
		ExistingObjects    []ctrlruntimeclient.Object
		ExpectedHTTPStatus int
		ExpectedAppDefs    []apiv2.ApplicationDefinition
	}{
		{
			Name:            "admin can list all applicationdefinitions",
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenApplicationDefinition("appdef1"),
				test.GenApplicationDefinition("appdef2"),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExpectedHTTPStatus: http.StatusOK,
			ExpectedAppDefs: []apiv2.ApplicationDefinition{
				test.GenApiApplicationDefinition("appdef1"),
				test.GenApiApplicationDefinition("appdef2"),
			},
		},
		{
			Name:            "user can list all applicationdefinitions",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenApplicationDefinition("appdef1"),
				test.GenApplicationDefinition("appdef2"),
			),
			ExpectedHTTPStatus: http.StatusOK,
			ExpectedAppDefs: []apiv2.ApplicationDefinition{
				test.GenApiApplicationDefinition("appdef1"),
				test.GenApiApplicationDefinition("appdef2"),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v2/applicationdefinitions", nil)
			res := httptest.NewRecorder()

			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, nil, nil, tc.ExistingObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatus, res.Code, res.Body.String())
			}

			wrapAppDef := test.NewApplicationDefinitionWrapper{}
			wrapAppDef.DecodeOrDie(res.Body, t).Sort()

			wrapExpAppDef := test.NewApplicationDefinitionWrapper(tc.ExpectedAppDefs)
			wrapAppDef.Sort()

			wrapAppDef.EqualOrDie(wrapExpAppDef, t)
		})
	}
}

func genKubermaticUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}
