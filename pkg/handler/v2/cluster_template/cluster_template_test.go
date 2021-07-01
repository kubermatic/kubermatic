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

func TestCreateClusterTemplateEndpoint(t *testing.T) {
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
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"user","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{},"enableUserSSHKeyAgent":true},"status":{"version":"","url":"","ccm":{"externalCCM":false}}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"project","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{},"enableUserSSHKeyAgent":true},"status":{"version":"","url":"","ccm":{"externalCCM":false}}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"john@acme.com","scope":"global","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{},"enableUserSSHKeyAgent":true},"status":{"version":"","url":"","ccm":{"externalCCM":false}}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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

func TestListClusterTemplates(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                     string
		ExpectedClusterTemplates []apiv2.ClusterTemplate
		HTTPStatus               int
		ExistingAPIUser          *apiv1.User
		ExistingKubermaticObjs   []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name: "scenario 1: list cluster templates",
			ExpectedClusterTemplates: []apiv2.ClusterTemplate{
				{
					Name:           "ct1",
					ID:             "ctID1",
					ProjectID:      test.GenDefaultProject().Name,
					User:           test.GenDefaultAPIUser().Email,
					Scope:          kubermaticv1.UserClusterTemplateScope,
					Cluster:        nil,
					NodeDeployment: nil,
				},
				{
					Name:           "ct2",
					ID:             "ctID2",
					ProjectID:      "",
					User:           "john@acme.com",
					Scope:          kubermaticv1.GlobalClusterTemplateScope,
					Cluster:        nil,
					NodeDeployment: nil,
				},
				{
					Name:           "ct4",
					ID:             "ctID4",
					ProjectID:      test.GenDefaultProject().Name,
					User:           "john@acme.com",
					Scope:          kubermaticv1.ProjectClusterTemplateScope,
					Cluster:        nil,
					NodeDeployment: nil,
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clustertemplates", test.ProjectName), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualClusterTemplates := test.NewClusterTemplateSliceWrapper{}
			actualClusterTemplates.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedClusterTemplates := test.NewClusterTemplateSliceWrapper(tc.ExpectedClusterTemplates)
			wrappedExpectedClusterTemplates.Sort()

			actualClusterTemplates.EqualOrDie(wrappedExpectedClusterTemplates, t)
		})
	}
}

func TestGetClusterTemplates(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		TemplateID             string
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: get global template",
			TemplateID:       "ctID2",
			ExpectedResponse: `{"name":"","id":"ctID2","user":"john@acme.com","scope":"global","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"","oidc":{}},"status":{"version":"","url":"","ccm":{"externalCCM":false}}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: get other user template",
			TemplateID:       "ctID3",
			ExpectedResponse: `{"error":{"code":403,"message":"user bob@acme.com can't access template ctID3"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:             "scenario 3: get user scope template",
			TemplateID:       "ctID1",
			ExpectedResponse: `{"name":"","id":"ctID1","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"user","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"","oidc":{}},"status":{"version":"","url":"","ccm":{"externalCCM":false}}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 4
		{
			Name:             "scenario 4: get project scope template",
			TemplateID:       "ctID4",
			ExpectedResponse: `{"name":"","id":"ctID4","projectID":"my-first-project-ID","user":"john@acme.com","scope":"project","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"","oidc":{}},"status":{"version":"","url":"","ccm":{"externalCCM":false}}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 5
		{
			Name:             "scenario 5: get template for different project",
			TemplateID:       "ctID5",
			ExpectedResponse: `{"error":{"code":403,"message":"cluster template doesn't belong to the project my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct5", "ctID5", "someProjectID", kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clustertemplates/%s", test.ProjectName, tc.TemplateID), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func TestDeleteClusterTemplates(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		TemplateID             string
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: regular user can't delete global template",
			TemplateID:       "ctID2",
			ExpectedResponse: `{"error":{"code":403,"message":"user bob@acme.com can't delete template ctID2"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: delete other user template",
			TemplateID:       "ctID3",
			ExpectedResponse: `{"error":{"code":403,"message":"user bob@acme.com can't access template ctID3"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:             "scenario 3: delete user scope template",
			TemplateID:       "ctID1",
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 4
		{
			Name:             "scenario 4: delete project scope template",
			TemplateID:       "ctID4",
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 5
		{
			Name:             "scenario 5: delete template for different project",
			TemplateID:       "ctID5",
			ExpectedResponse: `{"error":{"code":403,"message":"cluster template doesn't belong to the project my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct5", "ctID5", "someProjectID", kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v2/projects/%s/clustertemplates/%s", test.ProjectName, tc.TemplateID), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func TestCreateClusterTemplateInstance(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		TemplateToSync         string
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: create cluster template instance from global template",
			Body:             `{"replicas":1}`,
			TemplateToSync:   "ctID2",
			ExpectedResponse: `{"name":"my-first-project-ID-ctID2","spec":{"projectID":"my-first-project-ID","clusterTemplateID":"ctID2","clusterTemplateName":"ct2","replicas":1}}`,
			HTTPStatus:       http.StatusCreated,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/projects/%s/clustertemplates/%s/instances", test.ProjectName, tc.TemplateToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func TestGetClusterTemplateInstance(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		TemplateID             string
		InstanceID             string
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: get template instance for global template",
			TemplateID:       "ctID2",
			InstanceID:       "my-first-project-ID-ctID2",
			ExpectedResponse: `{"name":"my-first-project-ID-ctID2","spec":{"projectID":"my-first-project-ID","clusterTemplateID":"ctID2","clusterTemplateName":"","replicas":10}}`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID1", 10),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID2", 10),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID3", 10),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: get instance for other template",
			TemplateID:       "ctID2",
			InstanceID:       "my-first-project-ID-ctID1",
			ExpectedResponse: `{"error":{"code":500,"message":"cluster template instance doesn't belong to the template ctID2"}}`,
			HTTPStatus:       http.StatusInternalServerError,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID1", 10),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID2", 10),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID3", 10),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		}}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clustertemplates/%s/instances/%s", test.ProjectName, tc.TemplateID, tc.InstanceID), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func TestListClusterTemplateInstances(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		TemplateID             string
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: get template instance for global template",
			TemplateID:       "ctID2",
			ExpectedResponse: `[{"name":"my-first-project-ID-ctID2","spec":{"projectID":"my-first-project-ID","clusterTemplateID":"ctID2","clusterTemplateName":"","replicas":10}}]`,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenAdminUser("admin", "john@acme.com", true),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
				test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct3", "ctID3", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplate("ct4", "ctID4", test.GenDefaultProject().Name, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID1", 10),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID2", 10),
				test.GenClusterTemplateInstance(test.GenDefaultProject().Name, "ctID3", 10),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clustertemplates/%s/instances", test.ProjectName, tc.TemplateID), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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
