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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestListNodesEndpoint(t *testing.T) {
	t.Parallel()
	defaultNodes, err := test.GenDefaultExternalClusterNodes()
	if err != nil {
		t.Fatal(err)
	}
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ClusterToSync          string
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingKubeObjs       []ctrlruntimeclient.Object
		ExistingAPIUser        *apiv1.User
	}{
		{
			Name:             "scenario 1: get external cluster nodes",
			ExpectedResponse: `[{"id":"node1","name":"node1","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}},{"id":"node2","name":"node2","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"v1.15.12-gke.2"}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"290","memory":"687202304"},"nodeInfo":{"kernelVersion":"4.14","containerRuntime":"","containerRuntimeVersion":"containerd://1.2.8","kubeletVersion":"v1.15.12-gke.2","operatingSystem":"linux","architecture":"amd64"}}}]`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genExternalCluster(test.GenDefaultProject().Name, "clusterAbcID")),
			ExistingKubeObjs: defaultNodes,
			ClusterToSync:    "clusterAbcID",
			ExistingAPIUser:  test.GenDefaultAPIUser(),
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
			ExistingKubeObjs: defaultNodes,
			ClusterToSync:    "clusterAbcID",
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
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			// validate if deletion was successful
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/nodes", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, tc.ExistingKubeObjs, kubermaticObj, nil, hack.NewTestRouting)
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
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingKubeObjs       []ctrlruntimeclient.Object
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
			ExistingKubeObjs: []ctrlruntimeclient.Object{defaultNode},
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
			ExistingKubeObjs: []ctrlruntimeclient.Object{defaultNode},
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
			ExistingKubeObjs: []ctrlruntimeclient.Object{defaultNode},
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
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, tc.ExistingKubeObjs, kubermaticObj, nil, hack.NewTestRouting)
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

func TestGetClusterNodesMetrics(t *testing.T) {
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
		ExistingNodeMetrics    []*v1beta1.NodeMetrics
	}{
		// scenario 1
		{
			Name:             "scenario 1: gets cluster nodes metrics",
			ExpectedResponse: `[{"name":"mars","memoryTotalBytes":655,"memoryAvailableBytes":655,"memoryUsedPercentage":100,"cpuTotalMillicores":290000,"cpuAvailableMillicores":290000,"cpuUsedPercentage":100},{"name":"venus","memoryTotalBytes":655,"memoryAvailableBytes":655,"memoryUsedPercentage":100,"cpuTotalMillicores":290000,"cpuAvailableMillicores":290000,"cpuUsedPercentage":100}]`,
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
			Name:             "scenario 2: the admin John can get any cluster nodes metrics",
			ExpectedResponse: `[{"name":"mars","memoryTotalBytes":655,"memoryAvailableBytes":655,"memoryUsedPercentage":100,"cpuTotalMillicores":290000,"cpuAvailableMillicores":290000,"cpuUsedPercentage":100},{"name":"venus","memoryTotalBytes":655,"memoryAvailableBytes":655,"memoryUsedPercentage":100,"cpuTotalMillicores":290000,"cpuAvailableMillicores":290000,"cpuUsedPercentage":100}]`,
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
			Name:             "scenario 3: the user John can not get Bob's cluster nodes metrics",
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
			for _, existingMetric := range tc.ExistingNodeMetrics {
				kubernetesObj = append(kubernetesObj, existingMetric)
			}
			for _, node := range tc.ExistingNodes {
				kubeObj = append(kubeObj, node)
			}
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/nodesmetrics", test.ProjectName, tc.ClusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
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
