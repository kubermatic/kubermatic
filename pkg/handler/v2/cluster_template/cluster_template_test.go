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

package clustertemplate_test

import (
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

func TestCreateClusterRemplateEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ExistingProject        *kubermaticv1.Project
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		RewriteClusterID       bool
	}{
		// scenario 1
		{
			Name:             "scenario 1: create cluster template in user scope",
			Body:             `{"name":"test","scope":"user","cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"user","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{},"enableUserSSHKeyAgent":true},"status":{"version":"","url":""}}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: create cluster template in project scope",
			Body:             `{"name":"test","scope":"project","cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"project","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{},"enableUserSSHKeyAgent":true},"status":{"version":"","url":""}}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:             "scenario 3: create cluster template in global scope by admin",
			Body:             `{"name":"test","scope":"global","cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"john@acme.com","scope":"global","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{},"enableUserSSHKeyAgent":true},"status":{"version":"","url":""}}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		// scenario 4
		{
			Name:             "scenario 4: regular user can't create global cluster template",
			Body:             `{"name":"test","scope":"global","cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"error":{"code":500,"message":"the global scope is reserved only for admins"}}`,
			HTTPStatus:       http.StatusInternalServerError,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/projects/%s/clustertemplates", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, test.GenDefaultVersions(), nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			expectedResponse := tc.ExpectedResponse
			// since Cluster.Name is automatically generated by the system just rewrite it.
			if tc.RewriteClusterID {
				actualClusterTemplate := &apiv2.ClusterTemplate{}
				err = json.Unmarshal(res.Body.Bytes(), actualClusterTemplate)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualClusterTemplate.ID)
			}

			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}
