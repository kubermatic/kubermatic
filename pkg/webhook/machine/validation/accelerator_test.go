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

package validation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	kubevirttypes "k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidateCreate(t *testing.T) {
	validFootprint, err := accelerator.Encode(accelerator.NewKubeVirtFootprint(corev1.ResourceList{
		"nvidia.com/H200": resource.MustParse("2"),
	}))
	if err != nil {
		t.Fatalf("failed to encode test footprint: %v", err)
	}

	testCases := []struct {
		name        string
		machine     func(*testing.T) *clusterv1alpha1.Machine
		annotations map[string]string
		errorString string
	}{
		{
			name:        "valid KubeVirt footprint",
			machine:     func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, "h200-v1") },
			annotations: map[string]string{accelerator.FootprintAnnotationKey: validFootprint},
		},
		{
			name: "valid non-KubeVirt Machine",
			machine: func(t *testing.T) *clusterv1alpha1.Machine {
				return providerMachine(t, providerconfig.CloudProviderAWS, map[string]any{})
			},
		},
		{
			name:        "missing KubeVirt footprint",
			machine:     func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, "h200-v1") },
			errorString: "is missing",
		},
		{
			name:        "invalid footprint",
			machine:     func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, "h200-v1") },
			annotations: map[string]string{accelerator.FootprintAnnotationKey: "invalid"},
			errorString: "is invalid",
		},
		{
			name:    "unknown reserved annotation",
			machine: func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, "h200-v1") },
			annotations: map[string]string{
				accelerator.FootprintAnnotationKey:      validFootprint,
				accelerator.AnnotationPrefix + "future": "value",
			},
			errorString: "reserved accelerator prefix",
		},
		{
			name: "footprint on non-KubeVirt Machine",
			machine: func(t *testing.T) *clusterv1alpha1.Machine {
				return providerMachine(t, providerconfig.CloudProviderAWS, map[string]any{})
			},
			annotations: map[string]string{accelerator.FootprintAnnotationKey: validFootprint},
			errorString: "only valid for KubeVirt",
		},
		{
			name: "malformed provider config",
			machine: func(*testing.T) *clusterv1alpha1.Machine {
				return &clusterv1alpha1.Machine{Spec: clusterv1alpha1.MachineSpec{ProviderSpec: clusterv1alpha1.ProviderSpec{
					Value: &runtime.RawExtension{Raw: []byte(`not-json`)},
				}}}
			},
			errorString: "failed to read machine.spec.providerSpec",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			machine := tc.machine(t)
			machine.Annotations = tc.annotations
			_, err := NewAcceleratorValidator().ValidateCreate(context.Background(), machine)
			assertErrorContains(t, err, tc.errorString)
		})
	}
}

func TestValidateUpdateProtectsFootprintAndProviderSpec(t *testing.T) {
	validFootprint, err := accelerator.Encode(accelerator.NewKubeVirtFootprint(corev1.ResourceList{
		"nvidia.com/H200": resource.MustParse("1"),
	}))
	if err != nil {
		t.Fatalf("failed to encode test footprint: %v", err)
	}

	base := kubeVirtMachine(t, "h200-v1")
	base.Annotations = map[string]string{
		accelerator.FootprintAnnotationKey: validFootprint,
		"example.com/value":                "old",
	}

	testCases := []struct {
		name        string
		oldMachine  func(*testing.T) *clusterv1alpha1.Machine
		mutate      func(*testing.T, *clusterv1alpha1.Machine)
		errorString string
	}{
		{
			name:       "unchanged accounting intent",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine { return base.DeepCopy() },
			mutate: func(_ *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Annotations["example.com/value"] = "new"
			},
		},
		{
			name:       "footprint cannot change",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine { return base.DeepCopy() },
			mutate: func(_ *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Annotations[accelerator.FootprintAnnotationKey] = "changed"
			},
			errorString: "are managed by KKP and are immutable",
		},
		{
			name:       "footprint cannot be removed",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine { return base.DeepCopy() },
			mutate: func(_ *testing.T, machine *clusterv1alpha1.Machine) {
				delete(machine.Annotations, accelerator.FootprintAnnotationKey)
			},
			errorString: "are managed by KKP and are immutable",
		},
		{
			name:       "reserved annotation cannot be added",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine { return base.DeepCopy() },
			mutate: func(_ *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Annotations[accelerator.AnnotationPrefix+"future"] = "value"
			},
			errorString: "are managed by KKP and are immutable",
		},
		{
			name: "unchanged malformed legacy footprint does not wedge metadata updates",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine {
				machine := base.DeepCopy()
				machine.Annotations[accelerator.FootprintAnnotationKey] = "legacy-invalid-value"
				return machine
			},
			mutate: func(_ *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Labels = map[string]string{"example.com/key": "value"}
			},
		},
		{
			name:       "providerSpec cannot change through machine-controller bypass",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine { return base.DeepCopy() },
			mutate: func(t *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Spec.ProviderSpec = kubeVirtMachine(t, "h200-v2").Spec.ProviderSpec
				machine.Annotations["kubermatic.io/bypass-no-spec-mutation-requirement"] = "true"
			},
			errorString: "spec.providerSpec is immutable for accelerator accounting",
		},
		{
			name: "legacy providerConfig migration to KubeVirt ProviderSpec is allowed after bypass consumption",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine {
				return &clusterv1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: metav1.NamespaceSystem},
				}
			},
			mutate: func(t *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Spec.ProviderSpec = kubeVirtMachine(t, "h200-v1").Spec.ProviderSpec
			},
		},
		{
			name: "legacy providerConfig migration to malformed ProviderSpec is rejected",
			oldMachine: func(*testing.T) *clusterv1alpha1.Machine {
				return &clusterv1alpha1.Machine{
					ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: metav1.NamespaceSystem},
				}
			},
			mutate: func(_ *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Spec.ProviderSpec.Value = &runtime.RawExtension{Raw: []byte(`not-json`)}
			},
			errorString: "spec.providerSpec is immutable for accelerator accounting",
		},
		{
			name: "legacy KubeVirt Machine allows unrelated update",
			oldMachine: func(t *testing.T) *clusterv1alpha1.Machine {
				return kubeVirtMachine(t, "h200-v1")
			},
			mutate: func(_ *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Labels = map[string]string{"example.com/key": "value"}
			},
		},
		{
			name: "transition into KubeVirt cannot bypass footprint creation",
			oldMachine: func(t *testing.T) *clusterv1alpha1.Machine {
				return providerMachine(t, providerconfig.CloudProviderAWS, map[string]any{"instanceType": "m5.large"})
			},
			mutate: func(t *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Spec.ProviderSpec = kubeVirtMachine(t, "h200-v1").Spec.ProviderSpec
			},
			errorString: "spec.providerSpec is immutable for accelerator accounting",
		},
		{
			name: "unprotected non-KubeVirt providerSpec change remains untouched",
			oldMachine: func(t *testing.T) *clusterv1alpha1.Machine {
				return providerMachine(t, providerconfig.CloudProviderAWS, map[string]any{"instanceType": "m5.large"})
			},
			mutate: func(t *testing.T, machine *clusterv1alpha1.Machine) {
				machine.Spec.ProviderSpec = providerMachine(t, providerconfig.CloudProviderAWS, map[string]any{"instanceType": "m5.xlarge"}).Spec.ProviderSpec
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldMachine := tc.oldMachine(t)
			newMachine := oldMachine.DeepCopy()
			tc.mutate(t, newMachine)

			_, err := NewAcceleratorValidator().ValidateUpdate(context.Background(), oldMachine, newMachine)
			assertErrorContains(t, err, tc.errorString)
		})
	}
}

func providerMachine(t *testing.T, provider providerconfig.CloudProvider, cloudProviderSpec any) *clusterv1alpha1.Machine {
	t.Helper()
	rawCloudProviderSpec, err := json.Marshal(cloudProviderSpec)
	if err != nil {
		t.Fatalf("failed to marshal cloud provider config: %v", err)
	}
	rawProviderSpec, err := json.Marshal(providerconfig.Config{
		CloudProvider:     provider,
		CloudProviderSpec: runtime.RawExtension{Raw: rawCloudProviderSpec},
	})
	if err != nil {
		t.Fatalf("failed to marshal provider config: %v", err)
	}
	return &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: metav1.NamespaceSystem},
		Spec: clusterv1alpha1.MachineSpec{ProviderSpec: clusterv1alpha1.ProviderSpec{
			Value: &runtime.RawExtension{Raw: rawProviderSpec},
		}},
	}
}

func kubeVirtMachine(t *testing.T, instancetypeName string) *clusterv1alpha1.Machine {
	t.Helper()
	return providerMachine(t, providerconfig.CloudProviderKubeVirt, kubevirttypes.RawConfig{
		VirtualMachine: kubevirttypes.VirtualMachine{Instancetype: &kubevirtv1.InstancetypeMatcher{
			Name: instancetypeName,
			Kind: "VirtualMachineClusterInstancetype",
		}},
	})
}

func assertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if expected == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", expected)
	}
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("error = %q, want substring %q", err, expected)
	}
}
