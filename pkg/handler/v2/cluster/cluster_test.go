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

package cluster_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/handler/test"
	"github.com/kubermatic/kubermatic/pkg/handler/test/hack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ExistingProject        *kubermaticv1.Project
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
		RewriteClusterID       bool
	}{
		// scenario 1
		{
			Name:                   "scenario 1: a cluster with invalid spec is rejected",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}, "version":""}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"invalid cluster: invalid cloud spec \"Version\" is required but was not specified"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: cluster is created when valid spec and ssh key are passed",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{}},"status":{"version":"1.15.0","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
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
		// scenario 3
		{
			Name:                   "scenario 3: unable to create a cluster when the user doesn't belong to the project",
			Body:                   `{"cluster":{"humanReadableName":"keen-snyder","pause":false,"spec":{"version":"1.15.0","cloud":{"version":"1.15.0","fake":{},"dc":"fake-dc"}}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse:       `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:             http.StatusForbidden,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenUser("", "John", "john@acme.com")),
			ExistingAPIUser: func() *apiv1.User {
				defaultUser := test.GenDefaultAPIUser()
				defaultUser.Email = "john@acme.com"
				return defaultUser
			}(),
		},
		// scenario 4
		{
			Name:             "scenario 4: unable to create a cluster when project is not ready",
			Body:             `{"cluster":{"humanReadableName":"keen-snyder","pause":false,"spec":{"version":"1.15.0","cloud":{"fake":{},"dc":"fake-dc"}}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `{"error":{"code":503,"message":"Project is not initialized yet"}}`,
			HTTPStatus:       http.StatusServiceUnavailable,
			ExistingProject: func() *kubermaticv1.Project {
				project := test.GenDefaultProject()
				project.Status.Phase = kubermaticv1.ProjectInactive
				return project
			}(),
			ProjectToSync: test.GenDefaultProject().Name,
			ExistingKubermaticObjs: []runtime.Object{
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 5
		{
			Name:                   "scenario 5: openShift cluster is created",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","spec":{"version":"4.1.0","openshift":{"imagePullSecret": "some-secret"},"cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"openshift","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"4.1.0","oidc":{}},"status":{"version":"4.1.0","url":""}}`,
			RewriteClusterID:       true,
			HTTPStatus:             http.StatusCreated,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultSettings()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 6
		{
			Name:                   "scenario 6: openShift cluster is created with existing custom credential",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","credential":"fake","spec":{"version":"4.1.0","openshift":{"imagePullSecret": "some-secret"},"cloud":{"fake":{},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"openshift","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"4.1.0","oidc":{}},"status":{"version":"4.1.0","url":""}}`,
			RewriteClusterID:       true,
			HTTPStatus:             http.StatusCreated,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultSettings()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 7
		{
			Name:                   "scenario 7: custom credential doesn't exist for Fake cloud provider",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","credential":"default","spec":{"version":"4.1.0","cloud":{"fake":{},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"invalid credentials: missing preset 'default' for the user 'bob@acme.com'"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultSettings()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:                   "scenario 8: openShift cluster creation fails without imagePullSecret",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","credential":"fake","spec":{"version":"4.1.0","cloud":{"fake":{},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"openshift clusters must be configured with an imagePullSecret"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultSettings()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:                   "scenario 9a: rejected an attempt to create a cluster in email-restricted datacenter - legacy single domain restriction with requiredEmailDomains",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc"}}}}`,
			ExpectedResponse:       `{"error":{"code":404,"message":"datacenter \"restricted-fake-dc\" not found"}}`,
			RewriteClusterID:       false,
			HTTPStatus:             http.StatusNotFound,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:                   "scenario 9b: rejected an attempt to create a cluster in email-restricted datacenter - domain array restriction with `requiredEmailDomains`",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc2"}}}}`,
			ExpectedResponse:       `{"error":{"code":404,"message":"datacenter \"restricted-fake-dc2\" not found"}}`,
			RewriteClusterID:       false,
			HTTPStatus:             http.StatusNotFound,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 10a: create a cluster in email-restricted datacenter, to which the user does have access - legacy single domain restriction with requiredEmailDomains",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"restricted-fake-dc","fake":{}},"version":"1.15.0","oidc":{}},"status":{"version":"1.15.0","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenUser(test.UserID2, test.UserName2, test.UserEmail2),
				test.GenBinding(test.GenDefaultProject().Name, test.UserEmail2, "editors"),
			),
			ExistingAPIUser: test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			Name:             "scenario 10b: create a cluster in email-restricted datacenter, to which the user does have access - domain array restriction with `requiredEmailDomains`",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc2"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"restricted-fake-dc2","fake":{}},"version":"1.15.0","oidc":{}},"status":{"version":"1.15.0","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenUser(test.UserID2, test.UserName2, test.UserEmail2),
				test.GenBinding(test.GenDefaultProject().Name, test.UserEmail2, "editors"),
			),
			ExistingAPIUser: test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			Name:             "scenario 11: create a cluster in audit-logging-enforced datacenter, without explicitly enabling audit logging",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"audited-dc"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"audited-dc","fake":{}},"version":"1.15.0","oidc":{},"auditLogging":{"enabled":true}},"status":{"version":"1.15.0","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenUser(test.UserID2, test.UserName2, test.UserEmail2),
				test.GenBinding(test.GenDefaultProject().Name, test.UserEmail2, "editors"),
			),
			ExistingAPIUser: test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			Name:             "scenario 12: the admin user can create cluster for any project",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.15.0","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.15.0","oidc":{}},"status":{"version":"1.15.0","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", true),
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
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		// scenario 13
		{
			Name:                   "scenario 13: a cluster with invalid version",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}, "version":"1.2.3"}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"invalid cluster: invalid cloud spec: unsupported version 1.2.3"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 14
		{
			Name:                   "scenario 14: a cluster without version",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"invalid cluster: invalid cloud spec \"Version\" is required but was not specified"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/projects/%s/clusters", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, test.GenDefaultVersions(), nil, hack.NewTestRouting)
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
				actualCluster := &apiv1.Cluster{}
				err = json.Unmarshal(res.Body.Bytes(), actualCluster)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualCluster.ID)
			}

			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}

func genUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}
