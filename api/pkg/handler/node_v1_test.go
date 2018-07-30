package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	clienttesting "k8s.io/client-go/testing"
)

func TestDeleteNodeForCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                    string
		Body                    string
		HTTPStatus              int
		NodeNameToDelete        string
		ExistingProject         *kubermaticv1.Project
		ExistingKubermaticUser  *kubermaticv1.User
		ExistingAPIUser         *apiv1.User
		ExistingCluster         *kubermaticv1.Cluster
		ExistingNodes           []*corev1.Node
		ExistingMachines        []*v1alpha1.Machine
		ExpectedActions         int
		ExpectedHTTPStatusOnGet int
		ExpectedResponseOnGet   string
	}{
		// scenario 1
		{
			Name:             "scenario 1: delete the node that belong to the given cluster",
			Body:             ``,
			HTTPStatus:       http.StatusOK,
			NodeNameToDelete: "venus",
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "John",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: testEmail,
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
			ExistingMachines: []*v1alpha1.Machine{
				&v1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "venus",
					},
					Spec: v1alpha1.MachineSpec{
						ProviderConfig: runtime.RawExtension{
							Raw: []byte(`{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`),
						},
						Versions: v1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.6",
							ContainerRuntime: v1alpha1.ContainerRuntimeInfo{
								Name:    "docker",
								Version: "1.13",
							},
						},
					},
				},

				&v1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mars",
					},
					Spec: v1alpha1.MachineSpec{
						ProviderConfig: runtime.RawExtension{
							Raw: []byte(`{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`),
						},
						Versions: v1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.9",
							ContainerRuntime: v1alpha1.ContainerRuntimeInfo{
								Name:    "docker",
								Version: "1.12",
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
							Name:       "myProjectInternalName",
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
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/myProjectInternalName/dc/us-central1/clusters/abcd/nodes/%s", tc.NodeNameToDelete), strings.NewReader(tc.Body))
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
					if tc.NodeNameToDelete != deleteAction.GetName() {
						t.Fatalf("expected that machine %s will be deleted, but machine %s was deleted", tc.NodeNameToDelete, deleteAction.GetName())
					}
				}
			}
			if !deletedActionFound {
				t.Fatal("delete action was not found")
			}

			//
			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/myProjectInternalName/dc/us-central1/clusters/abcd/nodes/%s", tc.NodeNameToDelete), strings.NewReader(""))
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
		ExistingMachines       []*v1alpha1.Machine
	}{
		// scenario 1
		{
			Name:       "scenario 1: list nodes that belong to the given cluster",
			Body:       ``,
			HTTPStatus: http.StatusOK,
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "John",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: testEmail,
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
			ExistingMachines: []*v1alpha1.Machine{
				&v1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "venus",
					},
					Spec: v1alpha1.MachineSpec{
						ProviderConfig: runtime.RawExtension{
							Raw: []byte(`{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`),
						},
						Versions: v1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.6",
							ContainerRuntime: v1alpha1.ContainerRuntimeInfo{
								Name:    "docker",
								Version: "1.13",
							},
						},
					},
				},

				&v1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mars",
					},
					Spec: v1alpha1.MachineSpec{
						ProviderConfig: runtime.RawExtension{
							Raw: []byte(`{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`),
						},
						Versions: v1alpha1.MachineVersionInfo{
							Kubelet: "v1.9.9",
							ContainerRuntime: v1alpha1.ContainerRuntimeInfo{
								Name:    "docker",
								Version: "1.12",
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
							Name:       "myProjectInternalName",
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
			req := httptest.NewRequest("GET", "/api/v1/projects/myProjectInternalName/dc/us-central1/clusters/abcd/nodes", strings.NewReader(tc.Body))
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

			{
				nodesFromResponse := []apiv1.Node{}
				rawBody, err := ioutil.ReadAll(res.Body)
				if err != nil {
					t.Fatal(err)
				}

				err = json.Unmarshal(rawBody, &nodesFromResponse)
				if err != nil {
					t.Fatal(err)
				}

				if len(nodesFromResponse) != len(tc.ExpectedResponse) {
					t.Fatalf("expected to get %d keys but got %d", len(tc.ExpectedResponse), len(nodesFromResponse))
				}

				for _, expectedNode := range tc.ExpectedResponse {
					found := false
					for _, actualNode := range nodesFromResponse {
						if actualNode.ID == expectedNode.ID {
							if !areEqualOrDie(t, actualNode, expectedNode) {
								t.Fatalf("actual node != expected node, diff = %v", diff.ObjectDiff(actualNode, expectedNode))
							}
							found = true
						}
					}
					if !found {
						t.Fatalf("the node with the name = %s was not found in the returned output", expectedNode.Name)
					}
				}

			}
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
		NodeNameToSync         string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingNodes          []*corev1.Node
		ExistingMachines       []*v1alpha1.Machine
	}{
		// scenario 1
		{
			Name:             "scenario 1: get a node that belongs to the given cluster",
			Body:             ``,
			ExpectedResponse: `{"id":"venus","name":"venus","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{"digitalocean":{"size":"2GB","backups":false,"ipv6":false,"monitoring":false,"tags":null}},"operatingSystem":{},"versions":{"kubelet":"","containerRuntime":{"name":"","version":""}}},"status":{"machineName":"venus","capacity":{"cpu":"0","memory":"0"},"allocatable":{"cpu":"0","memory":"0"},"nodeInfo":{"kernelVersion":"","containerRuntime":"","containerRuntimeVersion":"","kubeletVersion":"","operatingSystem":"","architecture":""}}}`,
			HTTPStatus:       http.StatusOK,
			NodeNameToSync:   "venus",
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "John",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: testEmail,
			},
			ExistingNodes: []*corev1.Node{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "venus",
					},
				},
			},
			ExistingMachines: []*v1alpha1.Machine{
				&v1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "venus",
					},
					Spec: v1alpha1.MachineSpec{
						ProviderConfig: runtime.RawExtension{
							Raw: []byte(`{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}}`),
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
							Name:       "myProjectInternalName",
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/myProjectInternalName/dc/us-central1/clusters/abcd/nodes/%s", tc.NodeNameToSync), strings.NewReader(tc.Body))
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
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "John",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: testEmail,
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       "myProjectInternalName",
						},
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "us-central1",
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/projects/myProjectInternalName/dc/us-central1/clusters/abcd/nodes", strings.NewReader(tc.Body))
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
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualNode.ID, actualNode.Name, actualNode.Status.MachineName)
			}
			compareWithResult(t, res, expectedResponse)
		})
	}
}
