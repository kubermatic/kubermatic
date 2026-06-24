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
	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

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
			name:      "namespaced user-deployed instancetype with GPU is found (GPUs not counted in capacity)",
			namespace: "kkp-dev",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1alpha1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu-2", Namespace: "kkp-dev"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
						GPUs:   []kubevirtv1.GPU{{Name: "A100", DeviceName: "nv-a100-standard"}},
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
				&kvinstancetypev1alpha1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-cpu-2", Namespace: "cluster-apqx2l7v72"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("4Gi")},
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
				&kvinstancetypev1alpha1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu", Namespace: "tenant-a"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
					},
				},
				&kvinstancetypev1alpha1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu", Namespace: "tenant-b"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 8},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("32Gi")},
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
				&kvinstancetypev1alpha1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "custom-gpu", Namespace: "tenant-a"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "custom-gpu", Kind: "VirtualMachineInstancetype"},
			wantErr: true,
		},
		{
			name: "cluster-scoped instancetype is found",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1alpha1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "standard-4"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 4},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("16Gi")},
					},
				},
			},
			matcher: &kubevirtv1.InstancetypeMatcher{Name: "standard-4", Kind: "VirtualMachineClusterInstancetype"},
			wantCPU: 4,
		},
		{
			name: "empty kind resolves cluster-scoped instancetype",
			objects: []ctrlruntimeclient.Object{
				&kvinstancetypev1alpha1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "legacy-cluster"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 2},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("4Gi")},
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
				&kvinstancetypev1alpha1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{Name: "legacy-namespaced", Namespace: "kkp-dev"},
					Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 3},
						Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("12Gi")},
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
