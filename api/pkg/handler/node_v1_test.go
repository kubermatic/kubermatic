package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestDeleteNodeForCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                    string
		Body                    string
		HTTPStatus              int
		NodeIDToDelete          string
		ExistingProject         *kubermaticv1.Project
		ExistingKubermaticUser  *kubermaticv1.User
		ExistingAPIUser         *apiv1.User
		ExistingCluster         *kubermaticv1.Cluster
		ExistingNodes           []*corev1.Node
		ExistingMachines        []*clusterv1alpha1.Machine
		ExpectedActions         int
		ExpectedHTTPStatusOnGet int
		ExpectedResponseOnGet   string
	}{
		// scenario 1
		{
			Name:            "scenario 1: delete the node that belong to the given cluster",
			Body:            ``,
			HTTPStatus:      http.StatusOK,
			NodeIDToDelete:  "venus",
			ExistingProject: genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingNodes: []*corev1.Node{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "venus",
					},
				},

				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mars",
					},
				},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				&clusterv1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "venus",
						Namespace: "kube-system",
					},
					Spec: clusterv1alpha1.MachineSpec{
						ProviderConfig: clusterv1alpha1.ProviderConfig{
							Value: &runtime.RawExtension{
								Raw: []byte(`{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`),
							},
						},
						Versions: clusterv1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.6",
						},
					},
				},

				&clusterv1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mars",
						Namespace: "kube-system",
					},
					Spec: clusterv1alpha1.MachineSpec{
						ProviderConfig: clusterv1alpha1.ProviderConfig{
							Value: &runtime.RawExtension{
								Raw: []byte(`{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`),
							},
						},
						Versions: clusterv1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.9",
						},
					},
				},
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       testingProjectName,
						},
					},
				},
			},
			ExpectedActions: 2,
			//
			// even though the machine object was deleted the associated node object was not. When the client GETs the previously deleted "node" it will get a valid response.
			// That is only true for testing, but in a real cluster, the node object will get deleted by the garbage-collector as it has a ownerRef set.
			ExpectedHTTPStatusOnGet: http.StatusOK,
			ExpectedResponseOnGet:   `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{},"operatingSystem":{},"versions":{"kubelet":"","containerRuntime":{"name":"","version":""}}},"status":{"machineName":"","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"0","memory":"0"},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/abcd/nodes/%s", testingProjectName, tc.NodeIDToDelete), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingCluster != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingCluster)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			ep, clientsSets, err := createTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			fakeMachineClient := clientsSets.fakeMachineClient
			if len(fakeMachineClient.Actions()) != tc.ExpectedActions {
				t.Fatalf("expected to get %d but got %d actions = %v", tc.ExpectedActions, len(fakeMachineClient.Actions()), fakeMachineClient.Actions())
			}

			deletedActionFound := false
			for _, action := range fakeMachineClient.Actions() {
				if action.Matches("delete", "machines") {
					deletedActionFound = true
					deleteAction, ok := action.(clienttesting.DeleteAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					if tc.NodeIDToDelete != deleteAction.GetName() {
						t.Fatalf("expected that machine %s will be deleted, but machine %s was deleted", tc.NodeIDToDelete, deleteAction.GetName())
					}
				}
			}
			if !deletedActionFound {
				t.Fatal("delete action was not found")
			}

			//
			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/abcd/nodes/%s", testingProjectName, tc.NodeIDToDelete), strings.NewReader(""))
			res = httptest.NewRecorder()
			ep.ServeHTTP(res, req)
			if res.Code != tc.ExpectedHTTPStatusOnGet {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusOnGet, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponseOnGet)

		})
	}
}

func TestListNodesForCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       []apiv1.Node
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingNodes          []*corev1.Node
		ExistingMachines       []*clusterv1alpha1.Machine
	}{
		// scenario 1
		{
			Name:            "scenario 1: list nodes that belong to the given cluster",
			Body:            ``,
			HTTPStatus:      http.StatusOK,
			ExistingProject: genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingNodes: []*corev1.Node{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "venus",
					},
				},

				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mars",
					},
				},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				&clusterv1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "venus",
						Namespace: "kube-system",
					},
					Spec: clusterv1alpha1.MachineSpec{
						ProviderConfig: clusterv1alpha1.ProviderConfig{
							Value: &runtime.RawExtension{
								Raw: []byte(`{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`),
							},
						},
						Versions: clusterv1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.6",
						},
					},
				},

				&clusterv1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mars",
						Namespace: "kube-system",
					},
					Spec: clusterv1alpha1.MachineSpec{
						ProviderConfig: clusterv1alpha1.ProviderConfig{
							Value: &runtime.RawExtension{
								Raw: []byte(`{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`),
							},
						},
						Versions: clusterv1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.9",
						},
					},
				},
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       testingProjectName,
						},
					},
				},
			},
			ExpectedResponse: []apiv1.Node{
				apiv1.Node{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:   "venus",
						Name: "venus",
					},
					Spec: apiv2.NodeSpec{
						Cloud: apiv2.NodeCloudSpec{
							Digitalocean: &apiv2.DigitaloceanNodeSpec{
								Size: "2GB",
							},
						},
						OperatingSystem: apiv2.OperatingSystemSpec{
							Ubuntu: &apiv2.UbuntuSpec{
								DistUpgradeOnBoot: true,
							},
						},
						Versions: apiv2.NodeVersionInfo{
							Kubelet: "v1.9.6",
							ContainerRuntime: apiv2.NodeContainerRuntimeInfo{
								Name:    "docker",
								Version: "1.13",
							},
						},
					},
					Status: apiv2.NodeStatus{
						MachineName: "venus",
						Capacity: apiv2.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
						Allocatable: apiv2.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
					},
				},

				apiv1.Node{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:   "mars",
						Name: "mars",
					},
					Spec: apiv2.NodeSpec{
						Cloud: apiv2.NodeCloudSpec{
							AWS: &apiv2.AWSNodeSpec{
								InstanceType: "t2.micro",
								VolumeSize:   50,
							},
						},
						OperatingSystem: apiv2.OperatingSystemSpec{
							Ubuntu: &apiv2.UbuntuSpec{
								DistUpgradeOnBoot: false,
							},
						},
						Versions: apiv2.NodeVersionInfo{
							Kubelet: "v1.9.9",
							ContainerRuntime: apiv2.NodeContainerRuntimeInfo{
								Name:    "docker",
								Version: "1.12",
							},
						},
					},
					Status: apiv2.NodeStatus{
						MachineName: "mars",
						Capacity: apiv2.NodeResources{
							CPU:    "0",
							Memory: "0",
						},
						Allocatable: apiv2.NodeResources{
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
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/abcd/nodes", testingProjectName), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingCluster != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingCluster)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			ep, _, err := createTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualNodes := nodeV1SliceWrapper{}
			actualNodes.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedNodes := nodeV1SliceWrapper(tc.ExpectedResponse)
			wrappedExpectedNodes.Sort()

			actualNodes.EqualOrDie(wrappedExpectedNodes, t)
		})
	}
}

func TestGetNodeForCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		NodeIDToSync           string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingNodes          []*corev1.Node
		ExistingMachines       []*clusterv1alpha1.Machine
	}{
		// scenario 1
		{
			Name:             "scenario 1: get a node that belongs to the given cluster",
			Body:             ``,
			ExpectedResponse: `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{"digitalocean":{"size":"2GB","backups":false,"ipv6":false,"monitoring":false,"tags":null}},"operatingSystem":{},"versions":{"kubelet":"","containerRuntime":{"name":"","version":""}}},"status":{"machineName":"venus","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"0","memory":"0"},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
			HTTPStatus:       http.StatusOK,
			NodeIDToSync:     "venus",
			ExistingProject:  genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingNodes: []*corev1.Node{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "venus",
					},
				},
			},
			ExistingMachines: []*clusterv1alpha1.Machine{
				&clusterv1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "venus",
						Namespace: "kube-system",
					},
					Spec: clusterv1alpha1.MachineSpec{
						ProviderConfig: clusterv1alpha1.ProviderConfig{
							Value: &runtime.RawExtension{
								Raw: []byte(`{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}}`),
							},
						},
					},
				},
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       testingProjectName,
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/abcd/nodes/%s", testingProjectName, tc.NodeIDToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingCluster != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingCluster)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			for _, existingNode := range tc.ExistingNodes {
				kubernetesObj = append(kubernetesObj, existingNode)
			}
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			ep, _, err := createTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestCreateNodeForCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                               string
		Body                               string
		ExpectedResponse                   string
		HTTPStatus                         int
		RewriteClusterNameAndNamespaceName bool
		ExistingProject                    *kubermaticv1.Project
		ExistingKubermaticUser             *kubermaticv1.User
		ExistingAPIUser                    *apiv1.User
		ExistingCluster                    *kubermaticv1.Cluster
	}{
		// scenario 1
		{
			Name:                               "scenario 1: create a node that match the given spec",
			Body:                               `{"spec":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"containerRuntime":{"name":"docker"}}}}`,
			ExpectedResponse:                   `{"id":"%s","name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubermatic","kubermatic-cluster-abcd"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"","containerRuntime":{"name":"docker","version":""}}},"status":{"machineName":"%s","capacity":{"cpu":"","memory":""},"allocatable":{"cpu":"","memory":""},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
			HTTPStatus:                         http.StatusCreated,
			RewriteClusterNameAndNamespaceName: true,
			ExistingProject:                    genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingCluster: &kubermaticv1.Cluster{
				Status: kubermaticv1.ClusterStatus{
					Health: kubermaticv1.ClusterHealth{
						ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
							Apiserver:         true,
							Scheduler:         true,
							Controller:        true,
							MachineController: true,
							Etcd:              true,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       testingProjectName,
						},
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: "us-central1",
					},
				},
			},
		},
		// scenario 2
		{
			Name:                               "scenario 2: cluster components are not ready",
			Body:                               `{"spec":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"containerRuntime":{"name":"docker"}}}}`,
			ExpectedResponse:                   `{"error":{"code":503,"message":"Cluster components are not ready yet"}}`,
			HTTPStatus:                         http.StatusServiceUnavailable,
			RewriteClusterNameAndNamespaceName: true,
			ExistingProject:                    genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingCluster: &kubermaticv1.Cluster{
				Status: kubermaticv1.ClusterStatus{
					Health: kubermaticv1.ClusterHealth{
						ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
							Apiserver:         true,
							Scheduler:         true,
							Controller:        false,
							MachineController: true,
							Etcd:              true,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       testingProjectName,
						},
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: "us-central1",
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/abcd/nodes", testingProjectName), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}

			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingCluster != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingCluster)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
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
			compareWithResult(t, res, expectedResponse)
		})
	}
}
