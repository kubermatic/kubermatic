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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestListNodesEndpoint(t *testing.T) {
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
			Name:                   "scenario 1: get external cluster nodes",
			ExpectedResponse:       `[{"id":"node1","name":"node1","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}},{"id":"node2","name":"node2","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}}]`,
			HTTPStatus:             http.StatusOK,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID")),
			ClusterToSync:          "clusterAbcID",
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: the admin John can get Bob's cluster nodes",
			ExpectedResponse: `[{"id":"node1","name":"node1","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}},{"id":"node2","name":"node2","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}}]`,
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
			Name:             "scenario 3: the user John can not get Bob's cluster nodes",
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
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/nodes", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(""))
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

func TestGetNodeEndpoint(t *testing.T) {
	t.Parallel()
	defaultNode, err := test.GenDefaultExternalClusterNode()
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ClusterToSync          string
		NodeToSync             string
		ExistingKubermaticObjs []runtime.Object
		ExistingKubeObjs       []runtime.Object
		ExistingAPIUser        *apiv1.User
	}{
		{
			Name:             "scenario 1: get external cluster node",
			ExpectedResponse: `{"id":"node1","name":"node1","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}}`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genExternalCluster(
				test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingKubeObjs: []runtime.Object{defaultNode},
			ClusterToSync:    "clusterAbcID",
			NodeToSync:       defaultNode.Name,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: the admin John can get Bob's cluster nodes",
			ExpectedResponse: `{"id":"node1","name":"node1","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}}`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", true),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingKubeObjs: []runtime.Object{defaultNode},
			ClusterToSync:    "clusterAbcID",
			NodeToSync:       defaultNode.Name,
			ExistingAPIUser:  test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: the user John can not get Bob's cluster nodes",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add admin user
				genUser("John", "john@acme.com", false),
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID"),
			),
			ExistingKubeObjs: []runtime.Object{defaultNode},
			ClusterToSync:    "clusterAbcID",
			NodeToSync:       defaultNode.Name,
			ExistingAPIUser:  test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			// validate if deletion was successful
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/nodes/%s", tc.ProjectToSync, tc.ClusterToSync, tc.NodeToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, tc.ExistingKubeObjs, kubermaticObj, nil, nil, hack.NewTestRouting)
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
