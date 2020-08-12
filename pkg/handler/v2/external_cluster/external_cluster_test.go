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

package externalcluster_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

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
			Name:                   "scenario 1: cluster is created",
			Body:                   `{"name":"test","kubeconfig":"YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"}`,
			ExpectedResponse:       `{"id":"%s","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"kubernetes","spec":{"cloud":{"dc":""},"version":"","oidc":{}},"status":{"version":"","url":""}}`,
			RewriteClusterID:       true,
			HTTPStatus:             http.StatusCreated,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:                   "scenario 2: unable to create a cluster when the user doesn't belong to the project",
			Body:                   `{"name":"test","kubeconfig":"YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"}`,
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
		// scenario 3
		{
			Name:             "scenario 3: unable to create a cluster when project is not ready",
			Body:             `{"name":"test","kubeconfig":"YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"}`,
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
		{
			Name:             "scenario 4: the admin user can create cluster for any project",
			Body:             `{"name":"test","kubeconfig":"YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"}`,
			ExpectedResponse: `{"id":"%s","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"kubernetes","spec":{"cloud":{"dc":""},"version":"","oidc":{}},"status":{"version":"","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters", tc.ProjectToSync), strings.NewReader(tc.Body))
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

func TestDeleteClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ClusterToSync          string
		ExistingKubermaticObjs []runtime.Object
		ExistingAPIUser        *apiv1.User
	}{
		{
			Name:                   "scenario 1: delete external cluster",
			ExpectedResponse:       `{}`,
			HTTPStatus:             http.StatusOK,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genExternalCluster("clusterAbcID")),
			ClusterToSync:          "clusterAbcID",
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: the admin John can delete Bob's cluster",
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", true),
				genExternalCluster("clusterAbcID"),
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: the user John can not delete Bob's cluster",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", false),
				genExternalCluster("clusterAbcID"),
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			// validate if deletion was successful
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func genUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}

func genExternalCluster(name string) *kubermaticv1.ExternalCluster {
	return &kubermaticv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ExternalClusterSpec{
			HumanReadableName: name,
		},
	}
}
