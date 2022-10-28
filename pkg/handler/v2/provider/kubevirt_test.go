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
	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	kubevirtDatacenterName = "KubevirtDC"
)

func init() {
	utilruntime.Must(kvapiv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(kvinstancetypev1alpha1.AddToScheme(scheme.Scheme))
}

var (
	presetDefaultSmall1 = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "preset-default-small-1",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{},
		},
		Spec: *NewKubevirtPresetSpec(2, 2, 128, 2),
	}

	presetDefaultSmall2 = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "preset-default-small-2",
			Namespace: "default",
		},
		Spec: *NewKubevirtPresetSpec(2, 2, 128, 2),
	}
	// Should not be returned, not in "default" namespace.
	presetOtherSmall = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "preset-default-small",
			Namespace: "other",
		},
		Spec: *NewKubevirtPresetSpec(2, 2, 128, 2),
	}

	// Should not be returned, more than default resource-quota.
	presetNotInQuotaLimit = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "preset-not-in-quota-limit",
			Namespace: "default",
		},
		Spec: *NewKubevirtPresetSpec(256, 2, 512, 2),
	}

	// Should not be returned, requests.cpu and limits.cpu are unequal.
	presetWithInvalidCPU = kvapiv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "preset-invalid-cpu",
			Namespace: "default",
		},
		Spec: *NewKubevirtPresetSpec(256, 2, 512, 4),
	}

	// Cluster settings.
	clusterId    = "keen-snyder"
	clusterName  = "clusterAbc"
	fakeKvConfig = "eyJhcGlWZXJzaW9uIjoidjEiLCJjbHVzdGVycyI6W3siY2x1c3RlciI6eyJjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YSI6IiIsInNlcnZlciI6Imh0dHBzOi8vOTUuMjE2LjIwLjE0Njo2NDQzIn0sIm5hbWUiOiJrdWJlcm5ldGVzIn1dLCJjb250ZXh0cyI6W3siY29udGV4dCI6eyJjbHVzdGVyIjoia3ViZXJuZXRlcyIsIm5hbWVzcGFjZSI6Imt1YmUtc3lzdGVtIiwidXNlciI6Imt1YmVybmV0ZXMtYWRtaW4ifSwibmFtZSI6Imt1YmVybmV0ZXMtYWRtaW5Aa3ViZXJuZXRlcyJ9XSwiY3VycmVudC1jb250ZXh0Ijoia3ViZXJuZXRlcy1hZG1pbkBrdWJlcm5ldGVzIiwia2luZCI6IkNvbmZpZyIsInByZWZlcmVuY2VzIjp7fSwidXNlcnMiOlt7Im5hbWUiOiJrdWJlcm5ldGVzLWFkbWluIiwidXNlciI6eyJjbGllbnQtY2VydGlmaWNhdGUtZGF0YSI6IiIsImNsaWVudC1rZXktZGF0YSI6IiJ9fV19"
	// Credential ref name.
	credentialref = "credentialref"
	credentialns  = "ns"
)

type KeyValue struct {
	Key   string
	Value string
}

func NewKubevirtPresetSpec(memoryReq, cpuReq, memoryLimit, cpuLimit int64) *kvapiv1.VirtualMachineInstancePresetSpec {
	return &kvapiv1.VirtualMachineInstancePresetSpec{
		Domain: &kvapiv1.DomainSpec{
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
					corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dGi", memoryReq)),
					corev1.ResourceCPU:    *resource.NewQuantity(cpuReq, resource.Format("BinarySI")),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dGi", memoryLimit)),
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
	presetListResponse = `[{"name":"preset-default-small-1","namespace":"default","spec":"{\"selector\":{},\"domain\":{\"resources\":{\"requests\":{\"cpu\":\"2\",\"memory\":\"2Gi\"},\"limits\":{\"cpu\":\"2\",\"memory\":\"128Gi\"}},\"devices\":{\"disks\":[{\"name\":\"datavolumedisk\",\"disk\":{\"bus\":\"virtio\"}},{\"name\":\"cloudinitdisk\",\"disk\":{\"bus\":\"virtio\"}}]}}}"},{"name":"preset-default-small-2","namespace":"default","spec":"{\"selector\":{},\"domain\":{\"resources\":{\"requests\":{\"cpu\":\"2\",\"memory\":\"2Gi\"},\"limits\":{\"cpu\":\"2\",\"memory\":\"128Gi\"}},\"devices\":{\"disks\":[{\"name\":\"datavolumedisk\",\"disk\":{\"bus\":\"virtio\"}},{\"name\":\"cloudinitdisk\",\"disk\":{\"bus\":\"virtio\"}}]}}}"},{"name":"kubermatic-standard","spec":"{\"selector\":{\"matchLabels\":{\"kubevirt.io/flavor\":\"kubermatic-standard\"}},\"domain\":{\"resources\":{\"requests\":{\"cpu\":\"2\",\"memory\":\"8Gi\"},\"limits\":{\"cpu\":\"2\",\"memory\":\"8Gi\"}},\"devices\":{}}}"}]
`
)

func setFakeNewKubeVirtClient(objects []ctrlruntimeclient.Object) {
	providercommon.NewKubeVirtClient = func(kubeconfig string, options kubevirt.ClientOptions) (*kubevirt.Client, error) {
		return &kubevirt.Client{
			Client: fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build(),
		}, nil
	}
}

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
			Name:               "scenario 1: preset list - kubevirt kubeconfig provided",
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     "/api/v2/providers/kubevirt/vmflavors",
			HTTPRequestHeaders: []KeyValue{{Key: "Kubeconfig", Value: fakeKvConfig}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall, &presetNotInQuotaLimit, &presetWithInvalidCPU},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
		{
			Name:               "scenario 2: preset list - kubevirt kubeconfig from kubermatic preset",
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     "/api/v2/providers/kubevirt/vmflavors",
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
				GenKubeVirtKubermaticPreset(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall, &presetNotInQuotaLimit, &presetWithInvalidCPU},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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
			HTTPRequestMethod: http.MethodGet,
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/vmflavors", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
						Kubevirt: &kubermaticv1.KubevirtCloudSpec{
							Kubeconfig: fakeKvConfig,
						},
					}
					return cluster
				}(),
			),
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall, &presetNotInQuotaLimit, &presetWithInvalidCPU},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
		{
			Name:              "scenario 2: preset list- kubevirt kubeconfig from credential reference (secret)",
			HTTPRequestMethod: http.MethodGet,
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/vmflavors", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
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
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{&presetDefaultSmall1, &presetDefaultSmall2, &presetOtherSmall, &presetNotInQuotaLimit, &presetWithInvalidCPU},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        presetListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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
	instancetypeOk1             = newClusterInstancetype(2, "4Gi")
	instancetypeOk2             = newClusterInstancetype(4, "8Gi")
	instancetypeNotInQuotaLimit = newClusterInstancetype(35, "256Gi")

	instancetypeListResponse = "{\"instancetypes\":" +
		"{\"custom\":[" +
		"{\"name\":\"cpu-2-memory-4Gi\",\"spec\":\"{\\\"cpu\\\":{\\\"guest\\\":2},\\\"memory\\\":{\\\"guest\\\":\\\"4Gi\\\"}}\"}," +
		"{\"name\":\"cpu-4-memory-8Gi\",\"spec\":\"{\\\"cpu\\\":{\\\"guest\\\":4},\\\"memory\\\":{\\\"guest\\\":\\\"8Gi\\\"}}\"}" +
		"]," +
		"\"kubermatic\":[" +
		"{\"name\":\"standard-2\",\"spec\":\"{\\\"cpu\\\":{\\\"guest\\\":2},\\\"memory\\\":{\\\"guest\\\":\\\"8Gi\\\"}}\"}," +
		"{\"name\":\"standard-4\",\"spec\":\"{\\\"cpu\\\":{\\\"guest\\\":4},\\\"memory\\\":{\\\"guest\\\":\\\"16Gi\\\"}}\"}," +
		"{\"name\":\"standard-8\",\"spec\":\"{\\\"cpu\\\":{\\\"guest\\\":8},\\\"memory\\\":{\\\"guest\\\":\\\"32Gi\\\"}}\"}]}}"
)

func newClusterInstancetype(cpu uint32, memory string) *kvinstancetypev1alpha1.VirtualMachineClusterInstancetype {
	return &kvinstancetypev1alpha1.VirtualMachineClusterInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			Name: instancetypeName(cpu, memory),
		},
		Spec: getInstancetypeSpec(cpu, memory),
	}
}

func instancetypeName(cpu uint32, memory string) string {
	return fmt.Sprintf("cpu-%d-memory-%s", cpu, memory)
}

func getQuantity(q string) *resource.Quantity {
	res := resource.MustParse(q)
	return &res
}

func getInstancetypeSpec(cpu uint32, memory string) kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec {
	return kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
		CPU: kvinstancetypev1alpha1.CPUInstancetype{
			Guest: cpu,
		},
		Memory: kvinstancetypev1alpha1.MemoryInstancetype{
			Guest: *getQuantity(memory),
		},
	}
}

func TestListInstanceTypeEndpoint(t *testing.T) {
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
		// KUBEVIRT INSTANCETYPE LIST
		{
			Name:               "scenario 1: kubevirt kubeconfig provided",
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     "/api/v2/providers/kubevirt/instancetypes",
			HTTPRequestHeaders: []KeyValue{{Key: "Kubeconfig", Value: fakeKvConfig}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{instancetypeOk1, instancetypeOk2, instancetypeNotInQuotaLimit},

			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: instancetypeListResponse,
		},
		{
			Name:               "scenario 2: kubevirt kubeconfig from kubermatic preset",
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     "/api/v2/providers/kubevirt/instancetypes",
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
				GenKubeVirtKubermaticPreset(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{instancetypeOk1, instancetypeOk2, instancetypeNotInQuotaLimit},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        instancetypeListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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

func TestListInstancetypeNoCredentialsEndpoint(t *testing.T) {
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
		// KUBEVIRT INSTANCE TYPE LIST No Credentials
		{
			Name:              "scenario 1: kubevirt kubeconfig from cluster",
			HTTPRequestMethod: http.MethodGet,
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/instancetypes", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
						Kubevirt: &kubermaticv1.KubevirtCloudSpec{
							Kubeconfig: fakeKvConfig,
						},
					}
					return cluster
				}(),
			),
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{instancetypeOk1, instancetypeOk2, instancetypeNotInQuotaLimit},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        instancetypeListResponse,
		},
		{
			Name:              "scenario 2: - kubevirt kubeconfig from credential reference (secret)",
			HTTPRequestMethod: http.MethodGet,
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/instancetypes", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
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
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{instancetypeOk1, instancetypeOk2, instancetypeNotInQuotaLimit},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        instancetypeListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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
	preferenceCores        = newClusterPreference(kvinstancetypev1alpha1.PreferCores)
	preferenceSockets      = newClusterPreference(kvinstancetypev1alpha1.PreferSockets)
	preferenceListResponse = "{\"preferences\":" +
		"{\"custom\":[" +
		"{\"name\":\"preferCores\",\"spec\":\"{\\\"cpu\\\":{\\\"preferredCPUTopology\\\":\\\"preferCores\\\"}}\"}," +
		"{\"name\":\"preferSockets\",\"spec\":\"{\\\"cpu\\\":{\\\"preferredCPUTopology\\\":\\\"preferSockets\\\"}}\"}]," +
		"\"kubermatic\":[" +
		"{\"name\":\"sockets-advantage\",\"spec\":\"{\\\"cpu\\\":{\\\"preferredCPUTopology\\\":\\\"preferSockets\\\"}}\"}]}}"
)

func newClusterPreference(topology kvinstancetypev1alpha1.PreferredCPUTopology) *kvinstancetypev1alpha1.VirtualMachineClusterPreference {
	return &kvinstancetypev1alpha1.VirtualMachineClusterPreference{
		ObjectMeta: metav1.ObjectMeta{
			Name: string(topology),
		},
		Spec: kvinstancetypev1alpha1.VirtualMachinePreferenceSpec{
			CPU: &kvinstancetypev1alpha1.CPUPreferences{
				PreferredCPUTopology: topology,
			},
		},
	}
}

func TestPreferenceEndpoint(t *testing.T) {
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
		// KUBEVIRT PREFERENCE LIST
		{
			Name:               "scenario 1: kubevirt kubeconfig provided",
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     "/api/v2/providers/kubevirt/preferences",
			HTTPRequestHeaders: []KeyValue{{Key: "Kubeconfig", Value: fakeKvConfig}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{preferenceCores, preferenceSockets},

			ExistingAPIUser:  *test.GenDefaultAPIUser(),
			ExpectedResponse: preferenceListResponse,
		},
		{
			Name:               "scenario 2: kubevirt kubeconfig from kubermatic preset",
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     "/api/v2/providers/kubevirt/preferences",
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: []ctrlruntimeclient.Object{
				test.GenDefaultProject(),
				GenKubeVirtKubermaticPreset(),
			},
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{preferenceCores, preferenceSockets},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        preferenceListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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

func TestListPreferenceNoCredentialsEndpoint(t *testing.T) {
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
		// KUBEVIRT PREFERENCE LIST No Credentials
		{
			Name:              "scenario 1: kubevirt kubeconfig from cluster",
			HTTPRequestMethod: http.MethodGet,
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/preferences", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
						Kubevirt: &kubermaticv1.KubevirtCloudSpec{
							Kubeconfig: fakeKvConfig,
						},
					}
					return cluster
				}(),
			),
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{preferenceCores, preferenceSockets},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        preferenceListResponse,
		},
		{
			Name:              "scenario 2: - kubevirt kubeconfig from credential reference (secret)",
			HTTPRequestMethod: http.MethodGet,
			HTTPRequestURL:    fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/preferences", test.GenDefaultProject().Name, clusterId),
			Body:              ``,
			HTTPStatus:        http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
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
			ExistingKubevirtObjects: []ctrlruntimeclient.Object{preferenceCores, preferenceSockets},
			ExistingAPIUser:         *test.GenDefaultAPIUser(),
			ExpectedResponse:        preferenceListResponse,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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
	reclaimPolicy = corev1.PersistentVolumeReclaimDelete
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

	storageClassListResponse = `[{"name":"storageclass-1"},{"name":"storageclass-2"}]`
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
			HTTPRequestMethod:  http.MethodGet,
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
			HTTPRequestMethod:  http.MethodGet,
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
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/storageclasses", test.GenDefaultProject().Name, clusterId),
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
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
			HTTPRequestMethod:  http.MethodGet,
			HTTPRequestURL:     fmt.Sprintf("/api/v2/projects/%s/clusters/%s/providers/kubevirt/storageclasses", test.GenDefaultProject().Name, clusterId),
			HTTPRequestHeaders: []KeyValue{{Key: "Credential", Value: "kubermatic-preset"}},
			Body:               ``,
			HTTPStatus:         http.StatusOK,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster(clusterId, clusterName, test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud = kubermaticv1.CloudSpec{
						DatacenterName: kubevirtDatacenterName,
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
			setFakeNewKubeVirtClient(append(tc.ExistingKubevirtObjects, tc.ExistingKubevirtK8sObjects...))

			req := httptest.NewRequest(tc.HTTPRequestMethod, tc.HTTPRequestURL, strings.NewReader(tc.Body))
			for _, h := range tc.HTTPRequestHeaders {
				req.Header.Add(h.Key, h.Value)
			}
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(tc.ExistingAPIUser, tc.ExistingK8sObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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
