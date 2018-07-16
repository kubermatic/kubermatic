package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetNodeForCluster(t *testing.T) {
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
			ep, _, err := createTestEndpointAndGetClients(*tc.ExistingAPIUser, kubernetesObj, machineObj, kubermaticObj, nil, nil)
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
			Name:                               "scenario 1: create a node",
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
