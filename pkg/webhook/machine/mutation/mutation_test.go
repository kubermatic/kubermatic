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

package mutation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	kubevirtprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	kubevirttypes "k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMutatorIgnoresNonKubeVirtMachineAndReservesAnnotations(t *testing.T) {
	machine := machineWithProviderConfig(t, providerconfig.CloudProviderAWS, map[string]any{})
	machine.Annotations = map[string]string{
		accelerator.FootprintAnnotationKey:      "forged",
		accelerator.AnnotationPrefix + "future": "forged",
		"example.com/keep":                      "value",
	}

	if err := NewMutator(nil, "tenant-a").Default(context.Background(), machine); err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if _, exists := machine.Annotations[accelerator.FootprintAnnotationKey]; exists {
		t.Fatal("reserved footprint annotation was not removed")
	}
	if _, exists := machine.Annotations[accelerator.AnnotationPrefix+"future"]; exists {
		t.Fatal("unknown reserved annotation was not removed")
	}
	if machine.Annotations["example.com/keep"] != "value" {
		t.Fatal("unrelated annotation was changed")
	}
}

func TestMutatorStampsExplicitEmptyKubeVirtFootprintWithoutMatcher(t *testing.T) {
	machine := kubeVirtMachine(t, nil, "unused-kubeconfig")
	machine.Annotations = map[string]string{
		accelerator.FootprintAnnotationKey: "forged",
		"example.com/keep":                 "value",
	}
	mutator := NewMutator(nil, "tenant-a")
	mutator.resolveInstanceType = func(context.Context, string, string, *kubevirtv1.InstancetypeMatcher) (*kubevirtprovider.ResolvedInstanceType, error) {
		t.Fatal("resolver must not be called without an instancetype matcher")
		return nil, nil
	}

	if err := mutator.Default(context.Background(), machine); err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	footprint := decodeMachineFootprint(t, machine)
	if footprint.Resources == nil || len(footprint.Resources) != 0 {
		t.Fatalf("resources = %#v, want non-nil empty map", footprint.Resources)
	}
	if machine.Annotations["example.com/keep"] != "value" {
		t.Fatal("unrelated annotation was changed")
	}
}

func TestMutatorResolvesAndStampsKubeVirtFootprint(t *testing.T) {
	matcher := &kubevirtv1.InstancetypeMatcher{
		Name: "gpu-2d-8c-32gb-h200nvl",
		Kind: kubevirtprovider.VirtualMachineClusterInstancetypeKind,
	}
	machine := kubeVirtMachine(t, matcher, "infra-kubeconfig")
	machine.Annotations = map[string]string{
		accelerator.FootprintAnnotationKey:       `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{}}`,
		accelerator.AnnotationPrefix + "unknown": "forged",
		"example.com/keep":                       "value",
	}

	mutator := NewMutator(ctrlruntimefakeclient.NewClientBuilder().Build(), "tenant-general")
	callCount := 0
	mutator.resolveInstanceType = func(_ context.Context, kubeconfig, namespace string, gotMatcher *kubevirtv1.InstancetypeMatcher) (*kubevirtprovider.ResolvedInstanceType, error) {
		callCount++
		if kubeconfig != "infra-kubeconfig" {
			t.Errorf("kubeconfig = %q, want infra-kubeconfig", kubeconfig)
		}
		if namespace != "tenant-general" {
			t.Errorf("namespace = %q, want tenant-general", namespace)
		}
		if gotMatcher.Name != matcher.Name || gotMatcher.Kind != matcher.Kind {
			t.Errorf("matcher = %#v, want %#v", gotMatcher, matcher)
		}
		return &kubevirtprovider.ResolvedInstanceType{DeviceResources: corev1.ResourceList{
			"nvidia.com/GH100_H200_NVL": resource.MustParse("2"),
			"example.com/fpga":          resource.MustParse("1"),
		}}, nil
	}

	ctx := context.Background()
	if err := mutator.Default(ctx, machine); err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	firstValue := machine.Annotations[accelerator.FootprintAnnotationKey]
	if err := mutator.Default(ctx, machine); err != nil {
		t.Fatalf("second Default() error = %v", err)
	}
	if callCount != 2 {
		t.Fatalf("resolver call count = %d, want 2", callCount)
	}
	if got := machine.Annotations[accelerator.FootprintAnnotationKey]; got != firstValue {
		t.Fatalf("second mutation changed canonical footprint: first=%s second=%s", firstValue, got)
	}
	if _, exists := machine.Annotations[accelerator.AnnotationPrefix+"unknown"]; exists {
		t.Fatal("unknown reserved annotation was not removed")
	}
	if machine.Annotations["example.com/keep"] != "value" {
		t.Fatal("unrelated annotation was changed")
	}

	footprint := decodeMachineFootprint(t, machine)
	assertQuantity(t, footprint.Resources, "nvidia.com/GH100_H200_NVL", "2")
	assertQuantity(t, footprint.Resources, "example.com/fpga", "1")
}

func TestMutatorFailsClosed(t *testing.T) {
	matcher := &kubevirtv1.InstancetypeMatcher{Name: "h200-worker", Kind: kubevirtprovider.VirtualMachineClusterInstancetypeKind}
	inferFailurePolicy := kubevirtv1.RejectInferFromVolumeFailure
	testCases := []struct {
		name           string
		machine        func(*testing.T) *clusterv1alpha1.Machine
		namespace      string
		resolverResult *kubevirtprovider.ResolvedInstanceType
		resolverErr    error
		errorString    string
	}{
		{
			name:        "malformed provider config",
			machine:     func(*testing.T) *clusterv1alpha1.Machine { return malformedMachine() },
			namespace:   "tenant-a",
			errorString: "failed to read machine.spec.providerSpec",
		},
		{
			name: "empty instancetype matcher name",
			machine: func(t *testing.T) *clusterv1alpha1.Machine {
				return kubeVirtMachine(t, &kubevirtv1.InstancetypeMatcher{}, "kubeconfig")
			},
			namespace:   "tenant-a",
			errorString: "matcher name must not be empty",
		},
		{
			name: "instancetype revision",
			machine: func(t *testing.T) *clusterv1alpha1.Machine {
				return kubeVirtMachine(t, &kubevirtv1.InstancetypeMatcher{Name: "h200-worker", RevisionName: "h200-worker-v1"}, "kubeconfig")
			},
			namespace:   "tenant-a",
			errorString: "revisionName is not supported",
		},
		{
			name: "instancetype inferred from volume",
			machine: func(t *testing.T) *clusterv1alpha1.Machine {
				return kubeVirtMachine(t, &kubevirtv1.InstancetypeMatcher{InferFromVolume: "rootdisk"}, "kubeconfig")
			},
			namespace:   "tenant-a",
			errorString: "inference from volume is not supported",
		},
		{
			name: "instancetype inference failure policy",
			machine: func(t *testing.T) *clusterv1alpha1.Machine {
				return kubeVirtMachine(t, &kubevirtv1.InstancetypeMatcher{
					Name:                         "h200-worker",
					InferFromVolumeFailurePolicy: &inferFailurePolicy,
				}, "kubeconfig")
			},
			namespace:   "tenant-a",
			errorString: "inference failure policy is not supported",
		},
		{
			name:        "missing infrastructure namespace",
			machine:     func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, matcher, "kubeconfig") },
			errorString: "no infrastructure namespace configured",
		},
		{
			name:        "missing kubeconfig",
			machine:     func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, matcher, "") },
			namespace:   "tenant-a",
			errorString: "value is empty",
		},
		{
			name:        "resolver error",
			machine:     func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, matcher, "kubeconfig") },
			namespace:   "tenant-a",
			resolverErr: errors.New("collision"),
			errorString: "collision",
		},
		{
			name:           "nil resolver result",
			machine:        func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, matcher, "kubeconfig") },
			namespace:      "tenant-a",
			resolverResult: nil,
			errorString:    "resolver returned no result",
		},
		{
			name:      "invalid resolved resource quantity",
			machine:   func(t *testing.T) *clusterv1alpha1.Machine { return kubeVirtMachine(t, matcher, "kubeconfig") },
			namespace: "tenant-a",
			resolverResult: &kubevirtprovider.ResolvedInstanceType{DeviceResources: corev1.ResourceList{
				"nvidia.com/H200": resource.MustParse("500m"),
			}},
			errorString: "whole-number quantity",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			machine := tc.machine(t)
			mutator := NewMutator(ctrlruntimefakeclient.NewClientBuilder().Build(), tc.namespace)
			mutator.resolveInstanceType = func(context.Context, string, string, *kubevirtv1.InstancetypeMatcher) (*kubevirtprovider.ResolvedInstanceType, error) {
				if tc.resolverErr != nil {
					return nil, tc.resolverErr
				}
				return tc.resolverResult, nil
			}

			err := mutator.Default(context.Background(), machine)
			if err == nil {
				t.Fatal("Default() expected an error")
			}
			if !strings.Contains(err.Error(), tc.errorString) {
				t.Fatalf("Default() error = %q, want substring %q", err, tc.errorString)
			}
			if machine.Annotations != nil {
				if _, exists := machine.Annotations[accelerator.FootprintAnnotationKey]; exists {
					t.Fatal("failed mutation left a footprint annotation")
				}
			}
		})
	}
}

func machineWithProviderConfig(t *testing.T, provider providerconfig.CloudProvider, cloudProviderSpec any) *clusterv1alpha1.Machine {
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

func kubeVirtMachine(t *testing.T, matcher *kubevirtv1.InstancetypeMatcher, kubeconfig string) *clusterv1alpha1.Machine {
	t.Helper()
	return machineWithProviderConfig(t, providerconfig.CloudProviderKubeVirt, kubevirttypes.RawConfig{
		Auth: kubevirttypes.Auth{Kubeconfig: providerconfig.ConfigVarString{Value: kubeconfig}},
		VirtualMachine: kubevirttypes.VirtualMachine{
			Instancetype: matcher,
		},
	})
}

func malformedMachine() *clusterv1alpha1.Machine {
	return &clusterv1alpha1.Machine{Spec: clusterv1alpha1.MachineSpec{
		ProviderSpec: clusterv1alpha1.ProviderSpec{Value: &runtime.RawExtension{Raw: []byte(`not-json`)}},
	}}
}

func decodeMachineFootprint(t *testing.T, machine *clusterv1alpha1.Machine) accelerator.Footprint {
	t.Helper()
	value, exists := machine.Annotations[accelerator.FootprintAnnotationKey]
	if !exists {
		t.Fatal("Machine has no accelerator footprint annotation")
	}
	footprint, err := accelerator.Decode(value)
	if err != nil {
		t.Fatalf("failed to decode Machine footprint: %v", err)
	}
	return footprint
}

func assertQuantity(t *testing.T, resources corev1.ResourceList, name corev1.ResourceName, expected string) {
	t.Helper()
	quantity, exists := resources[name]
	if !exists {
		t.Fatalf("resource %q is missing", name)
	}
	if quantity.Cmp(resource.MustParse(expected)) != 0 {
		t.Fatalf("resource %q quantity = %s, want %s", name, quantity.String(), expected)
	}
}
