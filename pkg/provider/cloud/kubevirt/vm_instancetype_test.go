/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kubevirt

import (
	"context"
	"testing"

	kubevirtv1 "kubevirt.io/api/core/v1"
	kvinstancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestClient(t *testing.T, objs ...ctrlruntimeclient.Object) ctrlruntimeclient.Client {
	t.Helper()
	return ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		Build()
}

func TestDescribeInstanceType(t *testing.T) {
	testCases := []struct {
		name      string
		namespace string
		objects   []ctrlruntimeclient.Object
		matcher   *kubevirtv1.InstancetypeMatcher
		wantCPU   int64
		wantErr   bool
	}{
		{
			name:      "namespaced user-deployed instancetype with GPU is found",
			namespace: "kkp-dev",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu-2", Namespace: "kkp-dev"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
						GPUs:   []kubevirtv1.GPU{{Name: "A100", DeviceName: "nvidia.com/A100"}},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "custom-gpu-2", Kind: "VirtualMachineInstancetype"},
			wantCPU: 2,
		},
		{
			name:      "non-namespaced mode resolves instancetype in the cluster's dedicated namespace",
			namespace: "cluster-apqx2l7v72",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-cpu-2", Namespace: "cluster-apqx2l7v72"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("4Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "custom-cpu-2", Kind: "VirtualMachineInstancetype"},
			wantCPU: 2,
		},
		{
			name:      "namespaced lookup ignores a same-named instancetype in another tenant namespace",
			namespace: "tenant-a",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu", Namespace: "tenant-a"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
					},
				},
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu", Namespace: "tenant-b"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 8},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("32Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "custom-gpu", Kind: "VirtualMachineInstancetype"},
			wantCPU: 2,
		},
		{
			name:      "custom namespaced lookup fails when no infra namespace is configured",
			namespace: "",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu", Namespace: "tenant-a"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "custom-gpu", Kind: "VirtualMachineInstancetype"},
			wantErr: true,
		},
		{
			name: "cluster-scoped instancetype is found",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "standard-4"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 4},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("16Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "standard-4", Kind: "VirtualMachineClusterInstancetype"},
			wantCPU: 4,
		},
		{
			name: "empty kind resolves cluster-scoped instancetype",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "legacy-cluster"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("4Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "legacy-cluster", Kind: ""},
			wantCPU: 2,
		},
		{
			name:      "empty kind falls back to namespaced when no cluster-scoped match",
			namespace: "kkp-dev",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "legacy-namespaced", Namespace: "kkp-dev"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 3},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("12Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "legacy-namespaced", Kind: ""},
			wantCPU: 3,
		},
		{
			name:    "instancetype not found returns error",
			objects: nil,
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "nonexistent", Kind: "VirtualMachineInstancetype"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, tc.objects...)
			got, err := describeInstanceType(context.Background(), client, tc.namespace, tc.matcher)

			if (err != nil) != tc.wantErr {
				t.Fatalf("describeInstanceType() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got == nil {
				t.Fatal("expected non-nil NodeCapacity, got nil")
				return
			}
			if got.CPUCores == nil || got.CPUCores.Value() != tc.wantCPU {
				t.Errorf("CPU: got %v, want %d", got.CPUCores, tc.wantCPU)
			}

			// GPUs are intentionally not counted into node capacity for the
			// KubeVirt provider, so they must never be set.
			if got.GPUs != nil {
				t.Errorf("GPUs: got %v, want nil (GPUs are not counted in capacity)", got.GPUs)
			}
		})
	}
}

func TestResolveInstanceTypeDeviceResourcesAndSource(t *testing.T) {
	testCases := []struct {
		name            string
		namespace       string
		objects         []ctrlruntimeclient.Object
		matcher         *kubevirtv1.InstancetypeMatcher
		wantResources   corev1.ResourceList
		wantSource      InstanceTypeSource
		wantCapacityCPU int64
		wantMemory      resource.Quantity
	}{
		{
			name: "cluster-scoped host devices are counted and the explicit kind wins a same-name collision",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "h200-two-gpu"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 32},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Gi")},
						HostDevices: []kubevirtv1.HostDevice{
							{Name: "gpu-1", DeviceName: "nvidia.com/GH100_H200_NVL"},
							{Name: "gpu-2", DeviceName: "nvidia.com/GH100_H200_NVL"},
						},
					},
				},
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "h200-two-gpu", Namespace: "tenant-a"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 4},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("16Gi")},
					},
				},
			},
			namespace: "tenant-a",
			matcher: &kubevirtv1.InstancetypeMatcher{
				Name: "h200-two-gpu",
				Kind: VirtualMachineClusterInstancetypeKind,
			},
			wantResources: corev1.ResourceList{
				"nvidia.com/GH100_H200_NVL": resource.MustParse("2"),
			},
			wantSource: InstanceTypeSource{
				Kind: VirtualMachineClusterInstancetypeKind,
				Name: "h200-two-gpu",
			},
			wantCapacityCPU: 32,
			wantMemory:      resource.MustParse("128Gi"),
		},
		{
			name:      "namespaced GPU and host devices are combined by exact resource name",
			namespace: "tenant-a",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mixed-devices",
						Namespace: "tenant-a",
					},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 8},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("32Gi")},
						GPUs: []kubevirtv1.GPU{
							{Name: "gpu-1", DeviceName: "nvidia.com/A100"},
							{Name: "gpu-2", DeviceName: "nvidia.com/A100"},
						},
						HostDevices: []kubevirtv1.HostDevice{
							{Name: "gpu-3", DeviceName: "nvidia.com/A100"},
							{Name: "fpga-1", DeviceName: "example.com/fpga"},
						},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{
				Name: "mixed-devices",
				Kind: VirtualMachineInstancetypeKind,
			},
			wantResources: corev1.ResourceList{
				"nvidia.com/A100":  resource.MustParse("3"),
				"example.com/fpga": resource.MustParse("1"),
			},
			wantSource: InstanceTypeSource{
				Kind:      VirtualMachineInstancetypeKind,
				Namespace: "tenant-a",
				Name:      "mixed-devices",
			},
			wantCapacityCPU: 8,
			wantMemory:      resource.MustParse("32Gi"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, tc.objects...)
			got, err := resolveInstanceType(context.Background(), client, tc.namespace, tc.matcher)
			if err != nil {
				t.Fatalf("resolveInstanceType() error = %v", err)
			}

			if got.Capacity == nil || got.Capacity.CPUCores == nil || got.Capacity.CPUCores.Value() != tc.wantCapacityCPU {
				t.Fatalf("CPU: got %v, want %d", got.Capacity, tc.wantCapacityCPU)
			}
			if got.Capacity.GPUs != nil {
				t.Errorf("NodeCapacity.GPUs: got %v, want nil", got.Capacity.GPUs)
			}
			if got.Capacity.Memory == nil || got.Capacity.Memory.Cmp(tc.wantMemory) != 0 {
				t.Errorf("Memory: got %v, want %s", got.Capacity.Memory, tc.wantMemory.String())
			}
			assertResourceListEqual(t, got.DeviceResources, tc.wantResources)
			if got.Source != tc.wantSource {
				t.Errorf("Source: got %#v, want %#v", got.Source, tc.wantSource)
			}
		})
	}
}

func TestResolveInstanceTypeWithNilMatcher(t *testing.T) {
	_, err := resolveInstanceType(context.Background(), newTestClient(t), "tenant-a", nil)
	if err == nil {
		t.Fatal("resolveInstanceType() expected an error for a nil matcher")
	}
}

func TestResolveInstanceTypeRecordsActualSourceForLegacyMatcher(t *testing.T) {
	testCases := []struct {
		name       string
		namespace  string
		objects    []ctrlruntimeclient.Object
		wantSource InstanceTypeSource
	}{
		{
			name: "cluster-scoped match",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "legacy"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("4Gi")},
					},
				},
			},
			wantSource: InstanceTypeSource{
				Kind: VirtualMachineClusterInstancetypeKind,
				Name: "legacy",
			},
		},
		{
			name:      "namespaced fallback",
			namespace: "tenant-a",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "legacy", Namespace: "tenant-a"},
					Spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1beta1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("4Gi")},
					},
				},
			},
			wantSource: InstanceTypeSource{
				Kind:      VirtualMachineInstancetypeKind,
				Namespace: "tenant-a",
				Name:      "legacy",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveInstanceType(
				context.Background(),
				newTestClient(t, tc.objects...),
				tc.namespace,
				&kubevirtv1.InstancetypeMatcher{Name: "legacy"},
			)
			if err != nil {
				t.Fatalf("resolveInstanceType() error = %v", err)
			}
			if got.Source != tc.wantSource {
				t.Errorf("Source: got %#v, want %#v", got.Source, tc.wantSource)
			}
			if got.DeviceResources != nil {
				t.Errorf("DeviceResources: got %v, want nil", got.DeviceResources)
			}
		})
	}
}

func TestResolveInstanceTypeRejectsInvalidDeviceName(t *testing.T) {
	testCases := []struct {
		name string
		spec kvinstancetypev1beta1.VirtualMachineInstancetypeSpec
	}{
		{
			name: "GPU",
			spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
				GPUs: []kubevirtv1.GPU{{Name: "gpu-1"}},
			},
		},
		{
			name: "host device",
			spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
				HostDevices: []kubevirtv1.HostDevice{{Name: "host-device-1"}},
			},
		},
		{
			name: "unqualified resource name",
			spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
				GPUs: []kubevirtv1.GPU{{Name: "gpu-1", DeviceName: "nvidia-a100"}},
			},
		},
		{
			name: "resource name with multiple separators",
			spec: kvinstancetypev1beta1.VirtualMachineInstancetypeSpec{
				GPUs: []kubevirtv1.GPU{{Name: "gpu-1", DeviceName: "nvidia.com/gpu/a100"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveInstanceTypeSpec(tc.spec, InstanceTypeSource{})
			if err == nil {
				t.Fatal("resolveInstanceTypeSpec() expected an error for an invalid deviceName")
			}
		})
	}
}

func assertResourceListEqual(t *testing.T, got, want corev1.ResourceList) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("DeviceResources length: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for name, wantQuantity := range want {
		gotQuantity, exists := got[name]
		if !exists {
			t.Errorf("DeviceResources[%q]: missing, want %s", name, wantQuantity.String())
			continue
		}
		if gotQuantity.Cmp(wantQuantity) != 0 {
			t.Errorf("DeviceResources[%q]: got %s, want %s", name, gotQuantity.String(), wantQuantity.String())
		}
	}
}
