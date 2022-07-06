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

package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kvapiv1 "kubevirt.io/api/core/v1"

	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	kubevirtv1 "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/kubevirtcli/client/versioned"
	kubevirtclifake "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/kubevirtcli/client/versioned/fake"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesclientset "k8s.io/client-go/kubernetes"
	fakerestclient "k8s.io/client-go/kubernetes/fake"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	presetDefaultSmall1 = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "preset-default-small-1",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{},
		},
		Spec: *NewKubevirtPresetSpec(123, 234, 345, 456),
	}

	presetDefaultSmall2 = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "preset-default-small-2",
			Namespace: "default",
		},
		Spec: *NewKubevirtPresetSpec(123, 234, 345, 456),
	}
	// Should not be returned, not in "default" namespace
	presetOtherSmall = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "preset-default-small",
			Namespace: "other",
		},
		Spec: *NewKubevirtPresetSpec(123, 234, 345, 456),
	}

	// Cluster settings
	clusterId    = "keen-snyder"
	clusterName  = "clusterAbc"
	fakeKvConfig = "eyJhcGlWZXJzaW9uIjoidjEiLCJjbHVzdGVycyI6W3siY2x1c3RlciI6eyJjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YSI6IiIsInNlcnZlciI6Imh0dHBzOi8vOTUuMjE2LjIwLjE0Njo2NDQzIn0sIm5hbWUiOiJrdWJlcm5ldGVzIn1dLCJjb250ZXh0cyI6W3siY29udGV4dCI6eyJjbHVzdGVyIjoia3ViZXJuZXRlcyIsIm5hbWVzcGFjZSI6Imt1YmUtc3lzdGVtIiwidXNlciI6Imt1YmVybmV0ZXMtYWRtaW4ifSwibmFtZSI6Imt1YmVybmV0ZXMtYWRtaW5Aa3ViZXJuZXRlcyJ9XSwiY3VycmVudC1jb250ZXh0Ijoia3ViZXJuZXRlcy1hZG1pbkBrdWJlcm5ldGVzIiwia2luZCI6IkNvbmZpZyIsInByZWZlcmVuY2VzIjp7fSwidXNlcnMiOlt7Im5hbWUiOiJrdWJlcm5ldGVzLWFkbWluIiwidXNlciI6eyJjbGllbnQtY2VydGlmaWNhdGUtZGF0YSI6IiIsImNsaWVudC1rZXktZGF0YSI6IiJ9fV19"
	// Credential ref name
	credentialref = "credentialref"
	credentialns  = "ns"
)

func getRuntimeObjects(objs ...ctrlruntimeclient.Object) []runtime.Object {
	runtimeObjects := []runtime.Object{}
	for _, obj := range objs {
		runtimeObjects = append(runtimeObjects, obj.(runtime.Object))
	}

	return runtimeObjects
}

type KeyValue struct {
	Key   string
	Value string
}

func NewKubevirtPresetSpec(memoryReq, cpuReq, memoryLimit, cpuLimit int64) *kvapiv1.VirtualMachineInstancePresetSpec {
	return &kvapiv1.VirtualMachineInstancePresetSpec{
		Domain: &kvapiv1.DomainSpec{
			CPU: &kvapiv1.CPU{
				Cores: 2,
			},
			Devices: kvapiv1.Devices{
				Disks: []kvapiv1.Disk{
					{
						Name:       "datavolumedisk",
						DiskDevice: kvapiv1.DiskDevice{Disk: &kvapiv1.DiskTarget{Bus: "virtio"}},
					},
					{
						Name:       "cloudinitdisk",
						DiskDevice: kvapiv1.DiskDevice{Disk: &kvapiv1.DiskTarget{Bus: "virtio"}},
					},
				},
			},
			Resources: kvapiv1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(memoryReq, resource.Format("BinarySI")),
					corev1.ResourceCPU:    *resource.NewQuantity(cpuReq, resource.Format("BinarySI")),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(memoryLimit, resource.Format("BinarySI")),
					corev1.ResourceCPU:    *resource.NewQuantity(cpuLimit, resource.Format("BinarySI")),
				},
			},
		}}
}

func NewCredentialSecret(name, namespace string) *corev1.Secret {
	data := map[string][]byte{
		"kubeConfig": []byte(fakeKvConfig),
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func GenKubeVirtKubermaticPreset() *kubermaticv1.Preset {
	return &kubermaticv1.Preset{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubermatic-preset",
		},
		Spec: kubermaticv1.PresetSpec{
			Kubevirt: &kubermaticv1.Kubevirt{
				Kubeconfig: fakeKvConfig,
			},
			Fake: &kubermaticv1.Fake{Token: "dummy_pluton_token"},
		},
	}
}

var (
	presetListResponse = `[{"name":"preset-default-small-1","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"selector":{},"domain":{"resources":{"limits":{"cpu":"456","memory":"345"},"requests":{"cpu":"234","memory":"123"}},"cpu":{"cores":2},"devices":{"disks":[{"name":"datavolumedisk","disk":{"bus":"virtio"}},{"name":"cloudinitdisk","disk":{"bus":"virtio"}}]}}}},{"name":"preset-default-small-2","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"selector":{},"domain":{"resources":{"limits":{"cpu":"456","memory":"345"},"requests":{"cpu":"234","memory":"123"}},"cpu":{"cores":2},"devices":{"disks":[{"name":"datavolumedisk","disk":{"bus":"virtio"}},{"name":"cloudinitdisk","disk":{"bus":"virtio"}}]}}}}]`
)

func TestListPresetEndpoint(t *testing.T) {
	testcases := []struct {
		Name                       string
		HTTPRequestMethod          string
		HTTPRequestURL             string
		HTTPRequestHeaders         []KeyValue
		Body                       string
		ExpectedResponse           string
		HTTPStatus                 int
		ExistingKubermaticObjects  []ctrlruntimeclient.Object
		ExistingKubevirtObjects    []ctrlruntimeclient.Object
		ExistingKubevirtK8sObjects []ctrlruntimeclient.Object
		ExistingK8sObjects         []ctrlruntimeclient.Object
		ExistingAPIUser            apiv1.User
	}{
		// KUBEVIRT PRESET LIST
		{
			Name:               "scenario 1: preset list- kubevirt kubeconfig provided",
			HTTPRequestMethod:  "GET",
			HTTPRequestURL:     "/api/v2/providers/kubevirt/vmflavors",
			HTTPRequestHeaders: []KeyValue{{Key: "Kubeconfig", Value: fakeKvConfig}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
		{
			Name:               "scenario 2: preset list- kubevirt kubeconfig from kubermatic preset",
			HTTPRequestMethod:  "GET",
			HTTPRequestURL:     "/api/v2/providers/kubevirt/vmflavors",
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
				GenKubeVirtKubermaticPreset(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			providercommon.NewKubeVirtClientSet = func(kubeconfig string) (kubevirtv1.Interface, kubernetesclientset.Interface, error) {
				return kubevirtclifake.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtObjects...)...), fakerestclient.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtK8sObjects...)...), nil
			}

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListPresetNoCredentialsEndpoint(t *testing.T) {
	testcases := []struct {
		Name                       string
		HTTPRequestMethod          string
		HTTPRequestURL             string
		HTTPRequestHeaders         []KeyValue
		Body                       string
		ExpectedResponse           string
		HTTPStatus                 int
		ExistingKubermaticObjects  []ctrlruntimeclient.Object
		ExistingKubevirtObjects    []ctrlruntimeclient.Object
		ExistingKubevirtK8sObjects []ctrlruntimeclient.Object
		ExistingK8sObjects         []ctrlruntimeclient.Object
		ExistingAPIUser            apiv1.User
	}{
		// KUBEVIRT PRESET LIST No Credentials
		{
			Name:              "scenario 1: preset list- kubevirt kubeconfig from cluster",
			HTTPRequestMethod: "GET",
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/vmflavors", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: "KubevirtDC",
						Kubevirt: &kubermaticv1.KubevirtCloudSpec{
							Kubeconfig: fakeKvConfig,
						},
					}
					return cluster
				}(),
			),
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
		{
			Name:              "scenario 2: preset list- kubevirt kubeconfig from credential reference (secret)",
			HTTPRequestMethod: "GET",
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/vmflavors", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: "KubevirtDC",
						Kubevirt: &kubermaticv1.KubevirtCloudSpec{
							CredentialsReference: &types.GlobalSecretKeySelector{
								ObjectReference: corev1.ObjectReference{Name: credentialref, Namespace: credentialns},
							},
						},
					}
					return cluster
				}(),
			),
			ExistingK8sObjects:      []ctrlruntimeclient.Object{NewCredentialSecret(credentialref, credentialns)},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			providercommon.NewKubeVirtClientSet = func(kubeconfig string) (kubevirtv1.Interface, kubernetesclientset.Interface, error) {
				return kubevirtclifake.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtObjects...)...), fakerestclient.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtK8sObjects...)...), nil
			}

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

var (
	reclaimPolicy = v1.PersistentVolumeReclaimDelete
	storageClass1 = storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "storageclass-1",
		},
		ReclaimPolicy: &reclaimPolicy,
	}
	storageClass2 = storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "storageclass-2",
		},
	}

	storageClassListResponse = ` [{"name":"storageclass-1","creationTimestamp":"0001-01-01T00:00:00Z","provisioner":"","reclaimPolicy":"Delete"},{"name":"storageclass-2","creationTimestamp":"0001-01-01T00:00:00Z","provisioner":""}]`
)

func TestListStorageClassEndpoint(t *testing.T) {
	testcases := []struct {
		Name                       string
		HTTPRequestMethod          string
		HTTPRequestURL             string
		HTTPRequestHeaders         []KeyValue
		Body                       string
		ExpectedResponse           string
		HTTPStatus                 int
		ExistingKubermaticObjects  []ctrlruntimeclient.Object
		ExistingKubevirtObjects    []ctrlruntimeclient.Object
		ExistingKubevirtK8sObjects []ctrlruntimeclient.Object
		ExistingK8sObjects         []ctrlruntimeclient.Object
		ExistingAPIUser            apiv1.User
	}{
		// LIST Storage classes
		{
			Name:               "scenario 1: list storage classes- kubevirt kubeconfig provided",
			HTTPRequestMethod:  "GET",
			HTTPRequestURL:     "/api/v2/providers/kubevirt/storageclasses",
			HTTPRequestHeaders: []KeyValue{{Key: "Kubeconfig", Value: fakeKvConfig}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
			},
			ExistingKubevirtK8sObjects: []ctrlruntimeclient.Object{&storageClass1, &storageClass2},
			ExistingAPIUser:            *test.GenDefaultAPIUser(),
			ExpectedResponse:           storageClassListResponse,
		},
		{
			Name:               "scenario 2: list storage classes- kubevirt from kubermatic preset",
			HTTPRequestMethod:  "GET",
			HTTPRequestURL:     "/api/v2/providers/kubevirt/storageclasses",
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
				GenKubeVirtKubermaticPreset(),
			},
			ExistingKubevirtK8sObjects: []ctrlruntimeclient.Object{&storageClass1, &storageClass2},
			ExistingAPIUser:            *test.GenDefaultAPIUser(),
			ExpectedResponse:           storageClassListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			providercommon.NewKubeVirtClientSet = func(kubeconfig string) (kubevirtv1.Interface, kubernetesclientset.Interface, error) {
				return kubevirtclifake.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtObjects...)...), fakerestclient.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtK8sObjects...)...), nil
			}

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListStorageClassNoCredentialsEndpoint(t *testing.T) {
	testcases := []struct {
		Name                       string
		HTTPRequestMethod          string
		HTTPRequestURL             string
		HTTPRequestHeaders         []KeyValue
		Body                       string
		ExpectedResponse           string
		HTTPStatus                 int
		ExistingKubermaticObjects  []ctrlruntimeclient.Object
		ExistingKubevirtObjects    []ctrlruntimeclient.Object
		ExistingKubevirtK8sObjects []ctrlruntimeclient.Object
		ExistingK8sObjects         []ctrlruntimeclient.Object
		ExistingAPIUser            apiv1.User
	}{
		// LIST Storage classes No Credentials
		{
			Name:               "scenario 1: list storage classes- kubevirt kubeconfig from cluster",
			HTTPRequestMethod:  "GET",
			HTTPRequestURL:     fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/storageclasses", test.GenDefaultProject().Name, clusterId),
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: "KubevirtDC",
						Kubevirt: &kubermaticv1.KubevirtCloudSpec{
							Kubeconfig: fakeKvConfig,
						},
					}
					return cluster
				}(),
			),
			ExistingKubevirtK8sObjects: []ctrlruntimeclient.Object{&storageClass1, &storageClass2},
			ExistingAPIUser:            *test.GenDefaultAPIUser(),
			ExpectedResponse:           storageClassListResponse,
		},
		{
			Name:               "scenario 2: list storage classes- kubevirt kubeconfig from credential reference (secret)",
			HTTPRequestMethod:  "GET",
			HTTPRequestURL:     fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/storageclasses", test.GenDefaultProject().Name, clusterId),
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: "KubevirtDC",
						Kubevirt: &kubermaticv1.KubevirtCloudSpec{
							CredentialsReference: &types.GlobalSecretKeySelector{
								ObjectReference: corev1.ObjectReference{Name: credentialref, Namespace: credentialns},
							},
						},
					}
					return cluster
				}(),
			),
			ExistingKubevirtK8sObjects: []ctrlruntimeclient.Object{&storageClass1, &storageClass2},
			ExistingK8sObjects:         []ctrlruntimeclient.Object{NewCredentialSecret(credentialref, credentialns)},
			ExistingAPIUser:            *test.GenDefaultAPIUser(),
			ExpectedResponse:           storageClassListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			providercommon.NewKubeVirtClientSet = func(kubeconfig string) (kubevirtv1.Interface, kubernetesclientset.Interface, error) {
				return kubevirtclifake.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtObjects...)...), fakerestclient.NewSimpleClientset(getRuntimeObjects(tc.ExistingKubevirtK8sObjects...)...), nil
			}

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}
