package node_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/go-test/deep"
	"k8s.io/apimachinery/pkg/types"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestDeleteNodeForCluster(t *testing.T) {
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
			Name:                   "scenario 1: delete the node that belong to the given cluster",
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
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes/%s", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeIDToDelete), strings.NewReader(""))
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

			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes/%s", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeIDToDelete), strings.NewReader(""))
			res = httptest.NewRecorder()
			ep.ServeHTTP(res, req)
			if res.Code != tc.ExpectedHTTPStatusOnGet {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusOnGet, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponseOnGet)

		})
	}
}

func TestListNodesForCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       []apiv1.Node
		HTTPStatus             int
		ProjectIDToSync        string
		ClusterIDToSync        string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingNodes          []*corev1.Node
		ExistingMachines       []*clusterv1alpha1.Machine
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:                   "scenario 1: list nodes that belong to the given cluster",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				genTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			ExpectedResponse: []apiv1.Node{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "venus",
						Name: "venus",
					},
					Spec: apiv1.NodeSpec{
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
						SSHUserName: "root",
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Status: apiv1.NodeStatus{
						MachineName: "venus",
						Capacity: apiv1.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
						Allocatable: apiv1.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "mars",
						Name: "mars",
					},
					Spec: apiv1.NodeSpec{
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
						SSHUserName: "ubuntu",
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Status: apiv1.NodeStatus{
						MachineName: "mars",
						Capacity: apiv1.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
						Allocatable: apiv1.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
					},
				},
			},
		},
		// scenario 2
		{
			Name:                   "scenario 2: list nodes that belong to the given cluster should skip controlled machines",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, []metav1.OwnerReference{{APIVersion: "", Kind: "", Name: "", UID: ""}}),
				genTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			ExpectedResponse: []apiv1.Node{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "mars",
						Name: "mars",
					},
					Spec: apiv1.NodeSpec{
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
						SSHUserName: "ubuntu",
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Status: apiv1.NodeStatus{
						MachineName: "mars",
						Capacity: apiv1.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
						Allocatable: apiv1.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes", tc.ProjectIDToSync, tc.ClusterIDToSync), strings.NewReader(""))
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
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualNodes := test.NodeV1SliceWrapper{}
			actualNodes.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedNodes := test.NodeV1SliceWrapper(tc.ExpectedResponse)
			wrappedExpectedNodes.Sort()

			actualNodes.EqualOrDie(wrappedExpectedNodes, t)
		})
	}
}

func TestGetNodeForCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		HTTPStatus             int
		NodeIDToSync           string
		ClusterIDToSync        string
		ProjectIDToSync        string
		ExistingAPIUser        *apiv1.User
		ExistingNodes          []*corev1.Node
		ExistingMachines       []*clusterv1alpha1.Machine
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:                   "scenario 1: get a node that belongs to the given cluster",
			ExpectedResponse:       `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{"digitalocean":{"size":"2GB","backups":false,"ipv6":false,"monitoring":false,"tags":null}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"sshUserName":"root","versions":{"kubelet":"v9.9.9"}},"status":{"machineName":"venus","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"0","memory":"0"},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
			HTTPStatus:             http.StatusOK,
			NodeIDToSync:           "venus",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingNodes:          []*corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "venus"}}},
			ExistingMachines:       []*clusterv1alpha1.Machine{genTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "xyz": "abc"}, nil)},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes/%s", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeIDToSync), strings.NewReader(""))
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
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func TestCreateNodeForClusterIsDeprecated(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                               string
		Body                               string
		ExpectedResponse                   string
		ProjectIDToSync                    string
		ClusterIDToSync                    string
		HTTPStatus                         int
		RewriteClusterNameAndNamespaceName bool
		ExistingProject                    *kubermaticv1.Project
		ExistingKubermaticUser             *kubermaticv1.User
		ExistingAPIUser                    *apiv1.User
		ExistingCluster                    *kubermaticv1.Cluster
		ExistingKubermaticObjs             []runtime.Object
	}{
		// scenario 1
		{
			Name:                               "scenario 1: create a node that match the given spec",
			Body:                               `{"spec":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}`,
			ExpectedResponse:                   `{"error":{"code":400,"message":"Creating Nodes is deprecated. Please create a Node Deployment instead","details":["If you are calling this API endpoint directly then use POST \"v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments\" instead"]}}`,
			HTTPStatus:                         http.StatusBadRequest,
			RewriteClusterNameAndNamespaceName: true,
			ProjectIDToSync:                    test.GenDefaultProject().Name,
			ClusterIDToSync:                    test.GenDefaultCluster().Name,
			ExistingKubermaticObjs:             test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:                    test.GenDefaultAPIUser(),
		},

		// scenario 2
		{
			Name:                               "scenario 2: cluster components are not ready",
			Body:                               `{"spec":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}`,
			ExpectedResponse:                   `{"error":{"code":400,"message":"Creating Nodes is deprecated. Please create a Node Deployment instead","details":["If you are calling this API endpoint directly then use POST \"v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments\" instead"]}}`,
			HTTPStatus:                         http.StatusBadRequest,
			RewriteClusterNameAndNamespaceName: true,
			ProjectIDToSync:                    test.GenDefaultProject().Name,
			ClusterIDToSync:                    test.GenDefaultCluster().Name,
			ExistingKubermaticObjs:             test.GenDefaultKubermaticObjects(genTestCluster(false)),
			ExistingAPIUser:                    test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes", tc.ProjectIDToSync, tc.ClusterIDToSync), strings.NewReader(tc.Body))
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

			expectedResponse := tc.ExpectedResponse
			// since Node.Name is automatically generated by the system just rewrite it.
			if tc.RewriteClusterNameAndNamespaceName {
				actualNode := &apiv1.Node{}
				err = json.Unmarshal(res.Body.Bytes(), actualNode)
				if err != nil {
					t.Fatal(err)
				}
				if tc.HTTPStatus > 399 {
					expectedResponse = tc.ExpectedResponse
				} else {
					expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualNode.ID, actualNode.Name, actualNode.Status.MachineName)
				}
			}
			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}

func TestCreateNodeDeployment(t *testing.T) {
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
			Name:                   "scenario 1: create a node deployment that match the given spec",
			Body:                   `{"spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}}`,
			ExpectedResponse:       `{"name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":false},"status":{}}`,
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
			ExpectedResponse:       `{"name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"},"taints":[{"key":"foo","value":"bar","effect":"NoExecute"}]},"paused":false},"status":{}}`,
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
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments", tc.ProjectID, tc.ClusterID), strings.NewReader(tc.Body))
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
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, nd.Name)
			}

			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}

func TestListNodeDeployments(t *testing.T) {
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
			Name:                   "scenario 1: list node deployments that belong to the given cluster",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil),
				genTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil),
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
						Replicas: replicas,
						Paused:   &paused,
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
						Replicas: replicas,
						Paused:   &paused,
					},
					Status: clusterv1alpha1.MachineDeploymentStatus{},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments",
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

			actualNodeDeployments := nodeDeploymentSliceWrapper{}
			actualNodeDeployments.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedNodeDeployments := nodeDeploymentSliceWrapper(tc.ExpectedResponse)
			wrappedExpectedNodeDeployments.Sort()

			actualNodeDeployments.EqualOrDie(wrappedExpectedNodeDeployments, t)
		})
	}
}

func TestGetNodeDeployment(t *testing.T) {
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
			Name:                   "scenario 1: get node deployment that belong to the given cluster",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil),
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
					Replicas: replicas,
					Paused:   &paused,
				},
				Status: clusterv1alpha1.MachineDeploymentStatus{},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments/venus",
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

func TestListNodeDeploymentNodes(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                       string
		ExpectedResponse           []apiv1.Node
		HTTPStatus                 int
		ProjectIDToSync            string
		ClusterIDToSync            string
		ExistingProject            *kubermaticv1.Project
		ExistingKubermaticUser     *kubermaticv1.User
		ExistingAPIUser            *apiv1.User
		ExistingCluster            *kubermaticv1.Cluster
		ExistingNodes              []*corev1.Node
		ExistingMachineDeployments []*clusterv1alpha1.MachineDeployment
		ExistingMachines           []*clusterv1alpha1.Machine
		ExistingKubermaticObjs     []runtime.Object
		NodeDeploymentID           string
	}{
		// scenario 1
		{
			Name:                   "scenario 1: list nodes that belong to the given node deployment",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123"}),
				genTestMachineDeployment("mars", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil),
			},
			NodeDeploymentID: "venus",
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus-1", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				genTestMachine("venus-2", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "xyz": "abc"}, nil),
				// TODO @alvaroaleman: Have to deactivate these because the ctrlruntimefakeclient doesn't have a release where a LabelSelector is supported yet
				// This is already done in https://github.com/kubernetes-sigs/controller-runtime/pull/311 but we have to wait for that release
				//				genTestMachine("mars-1", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "345", "xyz": "abc"}, nil),
				//				genTestMachine("mars-2", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, nil),
			},
			ExpectedResponse: []apiv1.Node{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "venus-1",
						Name: "",
					},
					Spec: apiv1.NodeSpec{
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
						SSHUserName: "root",
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Status: apiv1.NodeStatus{
						MachineName: "venus-1",
						Capacity:    apiv1.NodeResources{},
						Allocatable: apiv1.NodeResources{},
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:   "venus-2",
						Name: "",
					},
					Spec: apiv1.NodeSpec{
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
						SSHUserName: "root",
						Versions: apiv1.NodeVersionInfo{
							Kubelet: "v9.9.9",
						},
					},
					Status: apiv1.NodeStatus{
						MachineName: "venus-2",
						Capacity:    apiv1.NodeResources{},
						Allocatable: apiv1.NodeResources{},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments/%s/nodes", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeDeploymentID), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualNodes := test.NodeV1SliceWrapper{}
			actualNodes.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedNodes := test.NodeV1SliceWrapper(tc.ExpectedResponse)
			wrappedExpectedNodes.Sort()

			actualNodes.EqualOrDie(wrappedExpectedNodes, t)
		})
	}
}

func TestListNodeDeploymentNodesEvents(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                       string
		HTTPStatus                 int
		ExpectedResult             string
		ProjectIDToSync            string
		ClusterIDToSync            string
		ExistingProject            *kubermaticv1.Project
		ExistingKubermaticUser     *kubermaticv1.User
		ExistingAPIUser            *apiv1.User
		ExistingCluster            *kubermaticv1.Cluster
		ExistingNodes              []*corev1.Node
		ExistingMachineDeployments []*clusterv1alpha1.MachineDeployment
		ExistingMachines           []*clusterv1alpha1.Machine
		ExistingKubermaticObjs     []runtime.Object
		ExistingEvents             []*corev1.Event
		NodeDeploymentID           string
		QueryParams                string
	}{
		// scenario 1
		{
			Name:                   "scenario 1: list all events",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123"}),
			},
			NodeDeploymentID: "venus",
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus-1", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			ExistingEvents: []*corev1.Event{
				genTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Machine", "venus-1-machine"),
				genTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Machine", "venus-1-machine"),
			},
			ExpectedResult: `[{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
		},
		// scenario 2
		{
			Name:                   "scenario 2: list all warning events",
			QueryParams:            "?type=warning",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123"}),
			},
			NodeDeploymentID: "venus",
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus-1", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			ExistingEvents: []*corev1.Event{
				genTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Machine", "venus-1-machine"),
				genTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Machine", "venus-1-machine"),
			},
			ExpectedResult: `[{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
		},
		// scenario 3
		{
			Name:                   "scenario 3: list all normal events",
			QueryParams:            "?type=normal",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123"}),
			},
			NodeDeploymentID: "venus",
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus-1", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
			ExistingEvents: []*corev1.Event{
				genTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Machine", "venus-1-machine"),
				genTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Machine", "venus-1-machine"),
			},
			ExpectedResult: `[{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Node","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments/%s/nodes/events%s", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeDeploymentID, tc.QueryParams), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingEvents := range tc.ExistingEvents {
				kubernetesObj = append(kubernetesObj, existingEvents)
			}
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)

			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResult)
		})
	}
}

func TestPatchNodeDeployment(t *testing.T) {
	t.Parallel()

	var replicas int32 = 1
	var replicasUpdated int32 = 3
	var kubeletVerUpdated = "v9.8.0"

	testcases := []struct {
		Name                       string
		Body                       string
		ExpectedResponse           string
		HTTPStatus                 int
		cluster                    string
		project                    string
		ExistingAPIUser            *apiv1.User
		NodeDeploymentID           string
		ExistingMachineDeployments []*clusterv1alpha1.MachineDeployment
		ExistingKubermaticObjs     []runtime.Object
	}{
		// Scenario 1: Update replicas count.
		{
			Name:                       "Scenario 1: Update replicas count",
			Body:                       fmt.Sprintf(`{"spec":{"replicas":%v}}`, replicasUpdated),
			ExpectedResponse:           fmt.Sprintf(`{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":%v,"template":{"cloud":{"digitalocean":{"size":"2GB","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":true}},"versions":{"kubelet":"v9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":false},"status":{}}`, replicasUpdated),
			cluster:                    "keen-snyder",
			HTTPStatus:                 http.StatusOK,
			project:                    test.GenDefaultProject().Name,
			ExistingAPIUser:            test.GenDefaultAPIUser(),
			NodeDeploymentID:           "venus",
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil)},
			ExistingKubermaticObjs:     test.GenDefaultKubermaticObjects(genTestCluster(true)),
		},
		// Scenario 2: Update kubelet version.
		{
			Name:                       "Scenario 2: Update kubelet version",
			Body:                       fmt.Sprintf(`{"spec":{"template":{"versions":{"kubelet":"%v"}}}}`, kubeletVerUpdated),
			ExpectedResponse:           fmt.Sprintf(`{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":%v,"template":{"cloud":{"digitalocean":{"size":"2GB","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":true}},"versions":{"kubelet":"%v"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":false},"status":{}}`, replicas, kubeletVerUpdated),
			cluster:                    "keen-snyder",
			HTTPStatus:                 http.StatusOK,
			project:                    test.GenDefaultProject().Name,
			ExistingAPIUser:            test.GenDefaultAPIUser(),
			NodeDeploymentID:           "venus",
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil)},
			ExistingKubermaticObjs:     test.GenDefaultKubermaticObjects(genTestCluster(true)),
		},
		// Scenario 3: Change to paused.
		{
			Name:                       "Scenario 3: Change to paused",
			Body:                       `{"spec":{"paused":true}}`,
			ExpectedResponse:           `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"2GB","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":true}},"versions":{"kubelet":"v9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":true},"status":{}}`,
			cluster:                    "keen-snyder",
			HTTPStatus:                 http.StatusOK,
			project:                    test.GenDefaultProject().Name,
			ExistingAPIUser:            test.GenDefaultAPIUser(),
			NodeDeploymentID:           "venus",
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil)},
			ExistingKubermaticObjs:     test.GenDefaultKubermaticObjects(genTestCluster(true)),
		},
		// Scenario 4: Downgrade to too old kubelet version
		{
			Name:                       "Scenario 4: Downgrade kubelet to too old",
			Body:                       fmt.Sprintf(`{"spec":{"template":{"versions":{"kubelet":"9.6.0"}}}}`),
			ExpectedResponse:           fmt.Sprintf(`{"error":{"code":400,"message":"kubelet version 9.6.0 is not compatible with control plane version 9.9.9"}}`),
			cluster:                    "keen-snyder",
			HTTPStatus:                 http.StatusBadRequest,
			project:                    test.GenDefaultProject().Name,
			ExistingAPIUser:            test.GenDefaultAPIUser(),
			NodeDeploymentID:           "venus",
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil)},
			ExistingKubermaticObjs:     test.GenDefaultKubermaticObjects(genTestCluster(true)),
		},
		// Scenario 5: Upgrade kubelet to a too new version
		{
			Name:                       "Scenario 5: Upgrade kubelet to too new",
			Body:                       fmt.Sprintf(`{"spec":{"template":{"versions":{"kubelet":"9.10.0"}}}}`),
			ExpectedResponse:           fmt.Sprintf(`{"error":{"code":400,"message":"kubelet version 9.10.0 is not compatible with control plane version 9.9.9"}}`),
			cluster:                    "keen-snyder",
			HTTPStatus:                 http.StatusBadRequest,
			project:                    test.GenDefaultProject().Name,
			ExistingAPIUser:            test.GenDefaultAPIUser(),
			NodeDeploymentID:           "venus",
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil)},
			ExistingKubermaticObjs:     test.GenDefaultKubermaticObjects(genTestCluster(true)),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments/%s",
				test.GenDefaultProject().Name, test.GenDefaultCluster().Name, tc.NodeDeploymentID), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineDeploymentObjets := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineDeploymentObjets = append(machineDeploymentObjets, existingMachineDeployment)
			}
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineDeploymentObjets, kubermaticObj, nil, nil, hack.NewTestRouting)
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

func TestDeleteNodeDeployment(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                        string
		HTTPStatus                  int
		NodeIDToDelete              string
		ClusterIDToSync             string
		ProjectIDToSync             string
		ExistingAPIUser             *apiv1.User
		ExistingNodes               []*corev1.Node
		ExistingMachineDeployments  []*clusterv1alpha1.MachineDeployment
		ExistingKubermaticObjs      []runtime.Object
		ExpectedHTTPStatusOnGet     int
		ExpectedResponseOnGet       string
		EpxectedNodeDeploymentCount int
	}{
		// scenario 1
		{
			Name:                   "scenario 1: delete the node that belong to the given cluster",
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
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil),
				genTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil),
			},
			// Even though the machine deployment object was deleted the associated node object was not.
			// When the client GETs the previously deleted "node" it will get a valid response.
			// That is only true for testing, but in a real cluster, the node object will get deleted by the garbage-collector as it has a ownerRef set.
			ExpectedHTTPStatusOnGet:     http.StatusOK,
			ExpectedResponseOnGet:       `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":""}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"0","memory":"0"},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
			EpxectedNodeDeploymentCount: 1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments/%s",
				tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeIDToDelete), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineDeploymentObjets := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineDeploymentObjets = append(machineDeploymentObjets, existingMachineDeployment)
			}
			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineDeploymentObjets, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
			if err := clientsSets.FakeClient.List(context.TODO(), machineDeployments); err != nil {
				t.Fatalf("failed to list MachineDeployments: %v", err)
			}

			if machineDeploymentCount := len(machineDeployments.Items); machineDeploymentCount != tc.EpxectedNodeDeploymentCount {
				t.Errorf("Expected to find %d  machineDeployments but got %d", tc.EpxectedNodeDeploymentCount, machineDeploymentCount)
			}

			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes/%s",
				tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeIDToDelete), strings.NewReader(""))
			res = httptest.NewRecorder()
			ep.ServeHTTP(res, req)
			if res.Code != tc.ExpectedHTTPStatusOnGet {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusOnGet, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponseOnGet)
		})
	}
}

func TestNodeDeploymentMetrics(t *testing.T) {
	t.Parallel()

	cpuQuantity, err := resource.ParseQuantity("290104582")
	if err != nil {
		t.Fatal(err)
	}
	memoryQuantity, err := resource.ParseQuantity("687202304")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		Name                       string
		ExpectedResponse           string
		HTTPStatus                 int
		ProjectIDToSync            string
		ClusterIDToSync            string
		ExistingProject            *kubermaticv1.Project
		ExistingKubermaticUser     *kubermaticv1.User
		ExistingAPIUser            *apiv1.User
		ExistingCluster            *kubermaticv1.Cluster
		ExistingNodes              []*corev1.Node
		ExistingMachineDeployments []*clusterv1alpha1.MachineDeployment
		ExistingMachines           []*clusterv1alpha1.Machine
		ExistingKubermaticObjs     []runtime.Object
		ExistingMetrics            []*v1beta1.NodeMetrics
		NodeDeploymentID           string
	}{
		// scenario 1
		{
			Name:                   "scenario 1: get metrics for the node deployment nodes",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus-1"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
			},
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123"}),
				genTestMachineDeployment("mars", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil),
			},
			NodeDeploymentID: "venus",
			ExistingMachines: []*clusterv1alpha1.Machine{
				genTestMachine("venus-1", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				genTestMachine("venus-2", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "xyz": "abc"}, nil),
			},
			ExistingMetrics: []*v1beta1.NodeMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "venus-1"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
			},
			ExpectedResponse: `[{"name":"venus-1","memoryTotalBytes":655,"memoryAvailableBytes":655,"memoryUsedPercentage":100,"cpuTotalMillicores":290104582000,"cpuAvailableMillicores":290104582000,"cpuUsedPercentage":100}]`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodedeployments/%s/nodes/metrics", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.NodeDeploymentID), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingMetric := range tc.ExistingMetrics {
				machineObj = append(machineObj, existingMetric)
			}
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
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

// nodeDeploymentSliceWrapper wraps []apiv1.NodeDeployment
// to provide convenient methods for tests
type nodeDeploymentSliceWrapper []apiv1.NodeDeployment

// Sort sorts the collection by CreationTimestamp
func (k nodeDeploymentSliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *nodeDeploymentSliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *nodeDeploymentSliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k nodeDeploymentSliceWrapper) EqualOrDie(expected nodeDeploymentSliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

func genTestMachine(name, rawProviderSpec string, labels map[string]string, ownerRef []metav1.OwnerReference) *clusterv1alpha1.Machine {
	return test.GenTestMachine(name, rawProviderSpec, labels, ownerRef)
}

func genTestMachineDeployment(name, rawProviderSpec string, selector map[string]string) *clusterv1alpha1.MachineDeployment {
	return test.GenTestMachineDeployment(name, rawProviderSpec, selector)
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

func genTestEvent(eventName, eventType, eventReason, eventMessage, kind, uid string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventName,
			Namespace: metav1.NamespaceSystem,
		},
		InvolvedObject: corev1.ObjectReference{
			UID:       types.UID(uid),
			Name:      "testMachine",
			Namespace: metav1.NamespaceSystem,
			Kind:      kind,
		},
		Reason:  eventReason,
		Message: eventMessage,
		Source:  corev1.EventSource{Component: "eventTest"},
		Count:   1,
		Type:    eventType,
	}
}
