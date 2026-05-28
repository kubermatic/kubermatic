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
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestClient(t *testing.T, objs ...ctrlruntimeclient.Object) ctrlruntimeclient.Client {
	t.Helper()
	return fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		Build()
}

func TestInstanceTypeToNodeCapacity_GPUCount(t *testing.T) {
	spec := kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
		CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 2},
		Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
		GPUs: []kubevirtv1.GPU{
			{Name: "A100", DeviceName: "nv-a100-standard"},
		},
	}

	got, err := instanceTypeToNodeCapacity(spec)
	if err != nil {
		t.Fatalf("instanceTypeToNodeCapacity returned error: %v", err)
	}

	if got.GPUs == nil {
		t.Fatalf("expected GPUs to be set, got nil")
	}
	if got.GPUs.Value() != 1 {
		t.Errorf("expected 1 GPU, got %d", got.GPUs.Value())
	}
}

func TestDescribeInstanceType_NamespacedCustomFound(t *testing.T) {
	// A user-deployed namespaced VirtualMachineInstancetype that is NOT one of the
	// embedded Kubermatic standards — describeInstanceType must find it via the cluster.
	custom := &kvinstancetypev1alpha1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{Name: "standard-gpu-2", Namespace: "kkp-dev"},
		Spec: kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
			CPU:    kvinstancetypev1alpha1.CPUInstancetype{Guest: 2},
			Memory: kvinstancetypev1alpha1.MemoryInstancetype{Guest: resource.MustParse("8Gi")},
			GPUs: []kubevirtv1.GPU{
				{Name: "A100", DeviceName: "nv-a100-standard"},
			},
		},
	}

	client := newTestClient(t, custom)
	matcher := &kubevirtv1.InstancetypeMatcher{Name: "standard-gpu-2", Kind: "VirtualMachineInstancetype"}

	got, err := describeInstanceType(context.Background(), client, matcher)
	if err != nil {
		t.Fatalf("describeInstanceType returned error for custom namespaced instancetype: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil NodeCapacity for found instancetype")
	}
	if got.CPUCores == nil || got.CPUCores.Value() != 2 {
		t.Errorf("expected 2 CPU cores, got %v", got.CPUCores)
	}
	if got.GPUs == nil || got.GPUs.Value() != 1 {
		t.Errorf("expected 1 GPU, got %v", got.GPUs)
	}
}
