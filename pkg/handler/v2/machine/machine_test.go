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

package machine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

func TestCreateMachineDeployment(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		ProjectID              string
		ClusterID              string
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:                   "scenario 1: create a machine deployment that match the given spec",
			Body:                   `{"spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":false,"dynamicConfig":false},"status":{}}`,
			HTTPStatus:             http.StatusCreated,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 2
		{
			Name:                   "scenario 2: cluster components are not ready",
			Body:                   `{"spec":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}`,
			ExpectedResponse:       `{"error":{"code":503,"message":"Cluster components are not ready yet"}}`,
			HTTPStatus:             http.StatusServiceUnavailable,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(false)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 3
		{
			Name:                   "scenario 3: kubelet version is too old",
			Body:                   `{"spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.6.0"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"node deployment validation failed: kubelet version 9.6.0 is not compatible with control plane version 9.9.9"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 4
		{
			Name:                   "scenario 4: kubelet version is too new",
			Body:                   `{"spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.10.0"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"node deployment validation failed: kubelet version 9.10.0 is not compatible with control plane version 9.9.9"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 5
		{
			Name:                   "scenario 5: set taints",
			Body:                   `{"spec":{"replicas":1,"template":{"taints": [{"key":"foo","value":"bar","effect":"NoExecute"}],"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"},"taints":[{"key":"foo","value":"bar","effect":"NoExecute"}]},"paused":false,"dynamicConfig":false},"status":{}}`,
			HTTPStatus:             http.StatusCreated,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 6
		{
			Name:                   "scenario 6: invalid taint",
			Body:                   `{"spec":{"replicas":1,"template":{"taints": [{"key":"foo","value":"bar","effect":"BAD_EFFECT"}],"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"node deployment validation failed: taint effect 'BAD_EFFECT' not allowed. Allowed: NoExecute, NoSchedule, PreferNoSchedule"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 7
		{
			Name:                   "scenario 7: create a machine deployment with dynamic config",
			Body:                   `{"spec":{"replicas":1,"dynamicConfig":true,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":false,"dynamicConfig":true},"status":{}}`,
			HTTPStatus:             http.StatusCreated,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/machinedeployments", tc.ProjectID, tc.ClusterID), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			// Since Node Deployment's ID, name and match labels are automatically generated by the system just rewrite them.
			nd := &apiv1.NodeDeployment{}
			var expectedResponse string
			err = json.Unmarshal(res.Body.Bytes(), nd)
			if err != nil {
				t.Fatal(err)
			}
			if tc.HTTPStatus > 399 {
				expectedResponse = tc.ExpectedResponse
			} else {
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, nd.Name, nd.Name)
			}

			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}

func TestDeleteMachineDeploymentNode(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                    string
		HTTPStatus              int
		NodeIDToDelete          string
		ClusterIDToSync         string
		ProjectIDToSync         string
		ExistingAPIUser         *apiv1.User
		ExistingNodes           []*corev1.Node
		ExistingMachines        []*clusterv1alpha1.Machine
		ExistingKubermaticObjs  []runtime.Object
		ExpectedHTTPStatusOnGet int
		ExpectedResponseOnGet   string
		ExpectedNodeCount       int
	}{
		// scenario 1
		{
			Name:                   "scenario 1: delete the machine node that belong to the given cluster",
			HTTPStatus:             http.StatusOK,
			NodeIDToDelete:         "venus",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				genTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			//
			// even though the machine object was deleted the associated node object was not. When the client GETs the previously deleted "node" it will get a valid response.
			// That is only true for testing, but in a real cluster, the node object will get deleted by the garbage-collector as it has a ownerRef set.
			ExpectedHTTPStatusOnGet: http.StatusOK,
			ExpectedResponseOnGet:   `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"0","memory":"0"},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
			ExpectedNodeCount:       1,
		},
		// scenario 2
		{
			Name:                   "scenario 2: the admin John can delete any cluster machine node",
			HTTPStatus:             http.StatusOK,
			NodeIDToDelete:         "venus",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster(), test.GenAdminUser("John", "john@acme.com", true)),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				genTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			//
			// even though the machine object was deleted the associated node object was not. When the client GETs the previously deleted "node" it will get a valid response.
			// That is only true for testing, but in a real cluster, the node object will get deleted by the garbage-collector as it has a ownerRef set.
			ExpectedHTTPStatusOnGet: http.StatusOK,
			ExpectedResponseOnGet:   `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"0","memory":"0"},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
			ExpectedNodeCount:       1,
		},
		// scenario 3
		{
			Name:                   "scenario 3: the user John can not delete Bob's cluster machine node",
			HTTPStatus:             http.StatusForbidden,
			NodeIDToDelete:         "venus",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster(), test.GenAdminUser("John", "john@acme.com", false)),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				genTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			ExpectedHTTPStatusOnGet: http.StatusForbidden,
			ExpectedResponseOnGet:   `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			ExpectedNodeCount:       2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/machinedeployments/nodes/%s", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeIDToDelete), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			machines := &clusterv1alpha1.MachineList{}
			if err := clientsSets.FakeClient.List(context.TODO(), machines); err != nil {
				t.Fatalf("failed to list machines from fake client: %v", err)
			}

			if machineCount := len(machines.Items); machineCount != tc.ExpectedNodeCount {
				t.Errorf("Expected %d machines to be gone but got %d", tc.ExpectedNodeCount, machineCount)
			}
		})
	}
}

func TestListMachineDeployments(t *testing.T) {
	t.Parallel()
	var replicas int32 = 1
	var paused = false
	testcases := []struct {
		Name                       string
		ExpectedResponse           []apiv1.NodeDeployment
		HTTPStatus                 int
		ProjectIDToSync            string
		ClusterIDToSync            string
		ExistingProject            *kubermaticv1.Project
		ExistingKubermaticUser     *kubermaticv1.User
		ExistingAPIUser            *apiv1.User
		ExistingCluster            *kubermaticv1.Cluster
		ExistingMachineDeployments []*clusterv1alpha1.MachineDeployment
		ExistingKubermaticObjs     []runtime.Object
	}{
		// scenario 1
		{
			Name:                   "scenario 1: list machine deployments that belong to the given cluster",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false),
				genTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
			},
			ExpectedResponse: []apiv1.NodeDeployment{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "venus",
						Name: "venus",
					},
					Spec: apiv1.NodeDeploymentSpec{
						Template: apiv1.NodeSpec{
							Cloud: apiv1.NodeCloudSpec{
								Digitalocean: &apiv1.DigitaloceanNodeSpec{
									Size: "2GB",
								},
							},
							OperatingSystem: apiv1.OperatingSystemSpec{
								Ubuntu: &apiv1.UbuntuSpec{
									DistUpgradeOnBoot: true,
								},
							},
							Versions: apiv1.NodeVersionInfo{
								Kubelet: "v9.9.9",
							},
						},
						Replicas:      replicas,
						Paused:        &paused,
						DynamicConfig: pointer.BoolPtr(false),
					},
					Status: clusterv1alpha1.MachineDeploymentStatus{},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "mars",
						Name: "mars",
					},
					Spec: apiv1.NodeDeploymentSpec{
						Template: apiv1.NodeSpec{
							Cloud: apiv1.NodeCloudSpec{
								AWS: &apiv1.AWSNodeSpec{
									InstanceType:     "t2.micro",
									VolumeSize:       50,
									AvailabilityZone: "eu-central-1a",
									SubnetID:         "subnet-2bff4f43",
								},
							},
							OperatingSystem: apiv1.OperatingSystemSpec{
								Ubuntu: &apiv1.UbuntuSpec{
									DistUpgradeOnBoot: false,
								},
							},
							Versions: apiv1.NodeVersionInfo{
								Kubelet: "v9.9.9",
							},
						},
						Replicas:      replicas,
						Paused:        &paused,
						DynamicConfig: pointer.BoolPtr(false),
					},
					Status: clusterv1alpha1.MachineDeploymentStatus{},
				},
			},
		},
		// scenario 2
		{
			Name:                   "scenario 2: the admin John can list machine deployments that belong to the given Bob's cluster",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster(), test.GenAdminUser("John", "john@acme.com", true)),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false),
				genTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
			},
			ExpectedResponse: []apiv1.NodeDeployment{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "venus",
						Name: "venus",
					},
					Spec: apiv1.NodeDeploymentSpec{
						Template: apiv1.NodeSpec{
							Cloud: apiv1.NodeCloudSpec{
								Digitalocean: &apiv1.DigitaloceanNodeSpec{
									Size: "2GB",
								},
							},
							OperatingSystem: apiv1.OperatingSystemSpec{
								Ubuntu: &apiv1.UbuntuSpec{
									DistUpgradeOnBoot: true,
								},
							},
							Versions: apiv1.NodeVersionInfo{
								Kubelet: "v9.9.9",
							},
						},
						Replicas:      replicas,
						Paused:        &paused,
						DynamicConfig: pointer.BoolPtr(false),
					},
					Status: clusterv1alpha1.MachineDeploymentStatus{},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "mars",
						Name: "mars",
					},
					Spec: apiv1.NodeDeploymentSpec{
						Template: apiv1.NodeSpec{
							Cloud: apiv1.NodeCloudSpec{
								AWS: &apiv1.AWSNodeSpec{
									InstanceType:     "t2.micro",
									VolumeSize:       50,
									AvailabilityZone: "eu-central-1a",
									SubnetID:         "subnet-2bff4f43",
								},
							},
							OperatingSystem: apiv1.OperatingSystemSpec{
								Ubuntu: &apiv1.UbuntuSpec{
									DistUpgradeOnBoot: false,
								},
							},
							Versions: apiv1.NodeVersionInfo{
								Kubelet: "v9.9.9",
							},
						},
						Replicas:      replicas,
						Paused:        &paused,
						DynamicConfig: pointer.BoolPtr(false),
					},
					Status: clusterv1alpha1.MachineDeploymentStatus{},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/machinedeployments",
				tc.ProjectIDToSync, tc.ClusterIDToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualNodeDeployments := test.NodeDeploymentSliceWrapper{}
			actualNodeDeployments.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedNodeDeployments := test.NodeDeploymentSliceWrapper(tc.ExpectedResponse)
			wrappedExpectedNodeDeployments.Sort()

			actualNodeDeployments.EqualOrDie(wrappedExpectedNodeDeployments, t)
		})
	}
}

func TestGetMachineDeployment(t *testing.T) {
	t.Parallel()
	var replicas int32 = 1
	var paused = false
	testcases := []struct {
		Name                       string
		ExpectedResponse           apiv1.NodeDeployment
		HTTPStatus                 int
		ProjectIDToSync            string
		ClusterIDToSync            string
		ExistingProject            *kubermaticv1.Project
		ExistingKubermaticUser     *kubermaticv1.User
		ExistingAPIUser            *apiv1.User
		ExistingCluster            *kubermaticv1.Cluster
		ExistingMachineDeployments []*clusterv1alpha1.MachineDeployment
		ExistingKubermaticObjs     []runtime.Object
	}{
		// scenario 1
		{
			Name:                   "scenario 1: get machine deployment that belong to the given cluster",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false),
			},
			ExpectedResponse: apiv1.NodeDeployment{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   "venus",
					Name: "venus",
				},
				Spec: apiv1.NodeDeploymentSpec{
					Template: apiv1.NodeSpec{
						Cloud: apiv1.NodeCloudSpec{
							Digitalocean: &apiv1.DigitaloceanNodeSpec{
								Size: "2GB",
							},
						},
						OperatingSystem: apiv1.OperatingSystemSpec{
							Ubuntu: &apiv1.UbuntuSpec{
								DistUpgradeOnBoot: true,
							},
						},
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Replicas:      replicas,
					Paused:        &paused,
					DynamicConfig: pointer.BoolPtr(false),
				},
				Status: clusterv1alpha1.MachineDeploymentStatus{},
			},
		},

		// scenario 2
		{
			Name:                   "scenario 2: get machine deployment that belong to the given cluster and has dynamic config set up",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, true),
			},
			ExpectedResponse: apiv1.NodeDeployment{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   "venus",
					Name: "venus",
				},
				Spec: apiv1.NodeDeploymentSpec{
					Template: apiv1.NodeSpec{
						Cloud: apiv1.NodeCloudSpec{
							Digitalocean: &apiv1.DigitaloceanNodeSpec{
								Size: "2GB",
							},
						},
						OperatingSystem: apiv1.OperatingSystemSpec{
							Ubuntu: &apiv1.UbuntuSpec{
								DistUpgradeOnBoot: true,
							},
						},
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Replicas:      replicas,
					Paused:        &paused,
					DynamicConfig: pointer.BoolPtr(true),
				},
				Status: clusterv1alpha1.MachineDeploymentStatus{},
			},
		},
		// scenario 3
		{
			Name:                   "scenario 1: the admin John can get any cluster machine deployment",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster(), test.GenAdminUser("John", "john@acme.com", true)),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false),
			},
			ExpectedResponse: apiv1.NodeDeployment{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   "venus",
					Name: "venus",
				},
				Spec: apiv1.NodeDeploymentSpec{
					Template: apiv1.NodeSpec{
						Cloud: apiv1.NodeCloudSpec{
							Digitalocean: &apiv1.DigitaloceanNodeSpec{
								Size: "2GB",
							},
						},
						OperatingSystem: apiv1.OperatingSystemSpec{
							Ubuntu: &apiv1.UbuntuSpec{
								DistUpgradeOnBoot: true,
							},
						},
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Replicas:      replicas,
					Paused:        &paused,
					DynamicConfig: pointer.BoolPtr(false),
				},
				Status: clusterv1alpha1.MachineDeploymentStatus{},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/machinedeployments/venus",
				tc.ProjectIDToSync, tc.ClusterIDToSync), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			bytes, err := json.Marshal(tc.ExpectedResponse)
			if err != nil {
				t.Fatalf("failed to marshall expected response %v", err)
			}

			test.CompareWithResult(t, res, string(bytes))
		})
	}
}

func genTestCluster(isControllerReady bool) *kubermaticv1.Cluster {
	controllerStatus := kubermaticv1.HealthStatusDown
	if isControllerReady {
		controllerStatus = kubermaticv1.HealthStatusUp
	}
	cluster := test.GenDefaultCluster()
	cluster.Status = kubermaticv1.ClusterStatus{
		ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
			Apiserver:                    kubermaticv1.HealthStatusUp,
			Scheduler:                    kubermaticv1.HealthStatusUp,
			Controller:                   controllerStatus,
			MachineController:            kubermaticv1.HealthStatusUp,
			Etcd:                         kubermaticv1.HealthStatusUp,
			CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
			UserClusterControllerManager: kubermaticv1.HealthStatusUp,
		},
	}
	cluster.Spec.Cloud = kubermaticv1.CloudSpec{
		DatacenterName: "regular-do1",
	}
	return cluster
}

func genTestMachine(name, rawProviderSpec string, labels map[string]string, ownerRef []metav1.OwnerReference) *clusterv1alpha1.Machine {
	return test.GenTestMachine(name, rawProviderSpec, labels, ownerRef)
}

func genTestMachineDeployment(name, rawProviderSpec string, selector map[string]string, dynamicConfig bool) *clusterv1alpha1.MachineDeployment {
	return test.GenTestMachineDeployment(name, rawProviderSpec, selector, dynamicConfig)
}
