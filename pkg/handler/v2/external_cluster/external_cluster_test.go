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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
		ExistingKubermaticObjs []ctrlruntimeclient.Object
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
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
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
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingAPIUser        *apiv1.User
	}{
		{
			Name:                   "scenario 1: delete external cluster",
			ExpectedResponse:       `{}`,
			HTTPStatus:             http.StatusOK,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID")),
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
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
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
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
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

func TestListClusters(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedClusters       []apiv1.Cluster
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name: "scenario 1: list clusters that belong to the given project",
			ExpectedClusters: []apiv1.Cluster{
				{
					ObjectMeta: apiv1.ObjectMeta{
						Name: "clusterAbcID",
						ID:   "clusterAbcID",
					},
					Type:   "kubernetes",
					Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: test.GenDefaultProject().Name},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						Name: "clusterDefID",
						ID:   "clusterDefID",
					},
					Type:   "kubernetes",
					Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: test.GenDefaultProject().Name},
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
				genExternalCluster(test.GenDefaultProject().Name, "clusterDefID"),
				genExternalCluster("fakeID", "clusterFakeID"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name: "scenario 2: the admin John can list Bob's clusters",
			ExpectedClusters: []apiv1.Cluster{
				{
					ObjectMeta: apiv1.ObjectMeta{
						Name: "clusterAbcID",
						ID:   "clusterAbcID",
					},
					Type:   "kubernetes",
					Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: test.GenDefaultProject().Name},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						Name: "clusterDefID",
						ID:   "clusterDefID",
					},
					Type:   "kubernetes",
					Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: test.GenDefaultProject().Name},
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", true),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
				genExternalCluster(test.GenDefaultProject().Name, "clusterDefID"),
				genExternalCluster("fakeID", "clusterFakeID"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters", test.ProjectName), strings.NewReader(""))
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

			actualClusters := test.NewClusterV1SliceWrapper{}
			actualClusters.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedClusters := test.NewClusterV1SliceWrapper(tc.ExpectedClusters)
			wrappedExpectedClusters.Sort()

			actualClusters.EqualOrDie(wrappedExpectedClusters, t)
		})
	}
}

func TestGetClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ClusterToSync          string
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingAPIUser        *apiv1.User
	}{
		{
			Name:                   "scenario 1: get external cluster",
			ExpectedResponse:       `{"id":"clusterAbcID","name":"clusterAbcID","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"kubernetes","spec":{"cloud":{"dc":""},"version":"1.17.9","oidc":{}},"status":{"version":"","url":""}}`,
			HTTPStatus:             http.StatusOK,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID")),
			ClusterToSync:          "clusterAbcID",
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: the admin John can get Bob's cluster",
			ExpectedResponse: `{"id":"clusterAbcID","name":"clusterAbcID","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"kubernetes","spec":{"cloud":{"dc":""},"version":"1.17.9","oidc":{}},"status":{"version":"","url":""}}`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", true),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: the user John can not get Bob's cluster",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", false),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			// validate if deletion was successful
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(""))
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

func TestUpdateClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ClusterToSync          string
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingAPIUser        *apiv1.User
	}{
		{
			Name:                   "scenario 1: update external cluster",
			Body:                   `{"name":"test"}`,
			ExpectedResponse:       `{"id":"clusterAbcID","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"kubernetes","spec":{"cloud":{"dc":""},"version":"","oidc":{}},"status":{"version":"","url":""}}`,
			HTTPStatus:             http.StatusOK,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID")),
			ClusterToSync:          "clusterAbcID",
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: the admin John can update Bob's cluster",
			Body:             `{"name":"test"}`,
			ExpectedResponse: `{"id":"clusterAbcID","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","labels":{"project-id":"my-first-project-ID"},"type":"kubernetes","spec":{"cloud":{"dc":""},"version":"","oidc":{}},"status":{"version":"","url":""}}`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", true),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: the user John can not update Bob's cluster",
			Body:             `{"name":"test"}`,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", false),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			// validate if deletion was successful
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(tc.Body))
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

func TestGetClusterMetrics(t *testing.T) {
	t.Parallel()
	cpuQuantity, err := resource.ParseQuantity("290")
	if err != nil {
		t.Fatal(err)
	}
	memoryQuantity, err := resource.ParseQuantity("687202304")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToGet           string
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingNodes          []*corev1.Node
		ExistingPodMetrics     []*v1beta1.PodMetrics
		ExistingNodeMetrics    []*v1beta1.NodeMetrics
	}{
		// scenario 1
		{
			Name:             "scenario 1: gets cluster metrics",
			ExpectedResponse: `{"name":"clusterAbcID","controlPlane":{"memoryTotalBytes":1310,"cpuTotalMillicores":580000},"nodes":{"memoryTotalBytes":1310,"memoryAvailableBytes":1310,"memoryUsedPercentage":100,"cpuTotalMillicores":580000,"cpuAvailableMillicores":580000,"cpuUsedPercentage":100}}`,
			ClusterToGet:     "clusterAbcID",
			HTTPStatus:       http.StatusOK,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ExistingPodMetrics: []*v1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "kube-system"},
					Containers: []v1beta1.ContainerMetrics{
						{
							Name:  "c1-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
						{
							Name:  "c2-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
					},
				},
			},
			ExistingNodeMetrics: []*v1beta1.NodeMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "venus"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "mars"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
			},
		},
		// scenario 2
		{
			Name:             "scenario 2: the admin John can get any cluster metrics",
			ExpectedResponse: `{"name":"clusterAbcID","controlPlane":{"memoryTotalBytes":1310,"cpuTotalMillicores":580000},"nodes":{"memoryTotalBytes":1310,"memoryAvailableBytes":1310,"memoryUsedPercentage":100,"cpuTotalMillicores":580000,"cpuAvailableMillicores":580000,"cpuUsedPercentage":100}}`,
			ClusterToGet:     "clusterAbcID",
			HTTPStatus:       http.StatusOK,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", true),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			ExistingPodMetrics: []*v1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "kube-system"},
					Containers: []v1beta1.ContainerMetrics{
						{
							Name:  "c1-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
						{
							Name:  "c2-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
					},
				},
			},
			ExistingNodeMetrics: []*v1beta1.NodeMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "venus"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "mars"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
			},
		},
		// scenario 3
		{
			Name:             "scenario 3: the user John can not get Bob's cluster metrics",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			ClusterToGet:     "clusterAbcID",
			HTTPStatus:       http.StatusForbidden,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", false),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			ExistingPodMetrics: []*v1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "cluster-defClusterID"},
					Containers: []v1beta1.ContainerMetrics{
						{
							Name:  "c1-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
						{
							Name:  "c2-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
					},
				},
			},
			ExistingNodeMetrics: []*v1beta1.NodeMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "venus"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "mars"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			for _, existingMetric := range tc.ExistingPodMetrics {
				kubernetesObj = append(kubernetesObj, existingMetric)
			}
			for _, existingMetric := range tc.ExistingNodeMetrics {
				kubernetesObj = append(kubernetesObj, existingMetric)
			}
			for _, node := range tc.ExistingNodes {
				kubeObj = append(kubeObj, node)
			}
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/metrics", test.ProjectName, tc.ClusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func TestGetClusterEvents(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToGet           string
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingNodes          []*corev1.Node
		ExistingNodeEvents     []*corev1.Event
		QueryParams            string
	}{
		// scenario 1
		{
			Name:             "scenario 1: gets all cluster events",
			ExpectedResponse: `[{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
			ClusterToGet:     "clusterAbcID",
			HTTPStatus:       http.StatusOK,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus", UID: "venus-1-machine"}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingNodeEvents: []*corev1.Event{
				test.GenTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Node", "venus-1-machine"),
				test.GenTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Node", "venus-1-machine"),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: gets only warning events",
			ExpectedResponse: `[{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
			QueryParams:      "?type=warning",
			ClusterToGet:     "clusterAbcID",
			HTTPStatus:       http.StatusOK,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus", UID: "venus-1-machine"}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingNodeEvents: []*corev1.Event{
				test.GenTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Node", "venus-1-machine"),
				test.GenTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Node", "venus-1-machine"),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:             "scenario 3: the admin John can get any cluster events",
			ExpectedResponse: `[{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
			ClusterToGet:     "clusterAbcID",
			HTTPStatus:       http.StatusOK,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus", UID: "venus-1-machine"}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", true),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			ExistingNodeEvents: []*corev1.Event{
				test.GenTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Node", "venus-1-machine"),
				test.GenTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Node", "venus-1-machine"),
			},
		},
		// scenario 4
		{
			Name:             "scenario 4: the user John can not get Bob's cluster events",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			ClusterToGet:     "clusterAbcID",
			HTTPStatus:       http.StatusForbidden,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus", UID: "venus-1-machine"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars", UID: "mars-1-machine"}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", false),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			ExistingNodeEvents: []*corev1.Event{
				test.GenTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Node", "venus-1-machine"),
				test.GenTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Node", "venus-1-machine"),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			for _, existingEvent := range tc.ExistingNodeEvents {
				kubernetesObj = append(kubernetesObj, existingEvent)
			}
			for _, node := range tc.ExistingNodes {
				kubeObj = append(kubeObj, node)
			}
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/events%s", test.ProjectName, tc.ClusterToGet, tc.QueryParams), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func genExternalCluster(projectName, clusterName string) *kubermaticv1.ExternalCluster {
	return &kubermaticv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterName,
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectName},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					Name:       projectName,
				},
			},
		},
		Spec: kubermaticv1.ExternalClusterSpec{
			HumanReadableName: clusterName,
		},
	}
}
