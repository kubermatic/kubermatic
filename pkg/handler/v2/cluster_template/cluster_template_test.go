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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			Body:             `{"name":"test","scope":"user","cluster":{"name":"keen-snyder","spec":{"version":"1.22.2","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"user","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.22.2","oidc":{},"enableUserSSHKeyAgent":true,"containerRuntime":"containerd"},"status":{"version":"","url":"","externalCCMMigration":""}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			Body:             `{"name":"test","scope":"project","cluster":{"name":"keen-snyder","spec":{"version":"1.22.2","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"project","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.22.2","oidc":{},"enableUserSSHKeyAgent":true,"containerRuntime":"containerd"},"status":{"version":"","url":"","externalCCMMigration":""}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			Body:             `{"name":"test","scope":"global","cluster":{"name":"keen-snyder","spec":{"version":"1.22.2","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"john@acme.com","scope":"global","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.22.2","oidc":{},"enableUserSSHKeyAgent":true,"containerRuntime":"containerd"},"status":{"version":"","url":"","externalCCMMigration":""}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			Body:             `{"name":"test","scope":"global","cluster":{"name":"keen-snyder","spec":{"version":"1.22.2","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"error":{"code":500,"message":"the global scope is reserved only for admins"}}`,
			HTTPStatus:       http.StatusInternalServerError,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 5
		{
			Name:             "scenario 5: create cluster template in project scope with SSH key",
			Body:             `{"name":"test","scope":"project","userSshKeys":[{"id":"key-c08aa5c7abf34504f18552846485267d-yafn","name":"test"}],"cluster":{"name":"keen-snyder","spec":{"version":"1.22.2","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"name":"test","id":"%s","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"project","userSshKeys":[{"name":"test","id":"key-c08aa5c7abf34504f18552846485267d-yafn"}],"cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.22.2","oidc":{},"enableUserSSHKeyAgent":true,"containerRuntime":"containerd"},"status":{"version":"","url":"","externalCCMMigration":""}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				// add an ssh key
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
				},
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 6
		{
			Name:             "scenario 6: create cluster template in project scope with wrong SSH key",
			Body:             `{"name":"test","scope":"project","userSshKeys":[{"id":"key-c08aa5c7abf34504f18552846485267d-yafn","name":"test"},{"id":"wrong-key","name":"wrong-key"}],"cluster":{"name":"keen-snyder","spec":{"version":"1.22.2","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"error":{"code":500,"message":"the given ssh key wrong-key does not belong to the given project my-first-project (my-first-project-ID)"}}`,
			HTTPStatus:       http.StatusInternalServerError,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				// add an ssh key
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
				},
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

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, test.GenDefaultVersions(), nil, nil, hack.NewTestRouting)
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
					Name:      "ct1",
					ID:        "ctID1",
					ProjectID: test.GenDefaultProject().Name,
					User:      test.GenDefaultAPIUser().Email,
					Scope:     kubermaticv1.UserClusterTemplateScope,
					Cluster: &apiv1.Cluster{
						Spec: apiv1.ClusterSpec{
							Cloud: kubermaticv1.CloudSpec{
								DatacenterName: "fake-dc",
								Fake:           &kubermaticv1.FakeCloudSpec{},
							},
						},
					},
					NodeDeployment: &apiv1.NodeDeployment{},
				},
				{
					Name:      "ct2",
					ID:        "ctID2",
					ProjectID: "",
					User:      "john@acme.com",
					Scope:     kubermaticv1.GlobalClusterTemplateScope,
					Cluster: &apiv1.Cluster{
						Spec: apiv1.ClusterSpec{
							Cloud: kubermaticv1.CloudSpec{
								DatacenterName: "fake-dc",
								Fake:           &kubermaticv1.FakeCloudSpec{},
							},
						},
					},
					NodeDeployment: &apiv1.NodeDeployment{},
				},
				{
					Name:      "ct4",
					ID:        "ctID4",
					ProjectID: test.GenDefaultProject().Name,
					User:      "john@acme.com",
					Scope:     kubermaticv1.ProjectClusterTemplateScope,
					Cluster: &apiv1.Cluster{
						Spec: apiv1.ClusterSpec{
							Cloud: kubermaticv1.CloudSpec{
								DatacenterName: "fake-dc",
								Fake:           &kubermaticv1.FakeCloudSpec{},
							},
						},
					},
					NodeDeployment: &apiv1.NodeDeployment{},
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
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, nil, hack.NewTestRouting)
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
			ExpectedResponse: `{"name":"ct2","id":"ctID2","user":"john@acme.com","scope":"global","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"","oidc":{}},"status":{"version":"","url":"","externalCCMMigration":""}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			ExpectedResponse: `{"name":"ct1","id":"ctID1","projectID":"my-first-project-ID","user":"bob@acme.com","scope":"user","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"","oidc":{}},"status":{"version":"","url":"","externalCCMMigration":""}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			ExpectedResponse: `{"name":"ct4","id":"ctID4","projectID":"my-first-project-ID","user":"john@acme.com","scope":"project","cluster":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","type":"","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"","oidc":{}},"status":{"version":"","url":"","externalCCMMigration":""}},"nodeDeployment":{"name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"template":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}}},"status":{}}}`,
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
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, nil, hack.NewTestRouting)
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
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, nil, hack.NewTestRouting)
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
			ExpectedResponse: `{"name":"%s","spec":{"projectID":"my-first-project-ID","clusterTemplateID":"ctID2","clusterTemplateName":"ct2","replicas":1}}`,
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
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualClusterTemplateInstance := &apiv2.ClusterTemplateInstance{}
			err = json.Unmarshal(res.Body.Bytes(), actualClusterTemplateInstance)
			if err != nil {
				t.Fatal(err)
			}
			expectedResponse := fmt.Sprintf(tc.ExpectedResponse, actualClusterTemplateInstance.Name)

			test.CompareWithResult(t, res, expectedResponse)

		})
	}
}
