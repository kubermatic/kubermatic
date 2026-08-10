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

package v1

import (
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	kubeVirtAcceleratorProvider = "kubevirt"
	h200ResourceName            = corev1.ResourceName("nvidia.com/GH100_H200_NVL")
	a100ResourceName            = corev1.ResourceName("nvidia.com/A100_80GB")
)

func TestResourceDetailsJSONAcceleratorListOmission(t *testing.T) {
	testCases := []struct {
		name                  string
		payload               string
		expectNilAccelerators bool
	}{
		{
			name:                  "omitted accelerator list",
			payload:               `{"cpu":"2","memory":"4Gi","storage":"10Gi"}`,
			expectNilAccelerators: true,
		},
		{
			name:    "empty accelerator list",
			payload: `{"accelerators":[]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceDetails := ResourceDetails{}
			if err := json.Unmarshal([]byte(tc.payload), &resourceDetails); err != nil {
				t.Fatalf("failed to decode resource details: %v", err)
			}
			if len(resourceDetails.Accelerators) != 0 {
				t.Fatalf("expected no accelerators, got %v", resourceDetails.Accelerators)
			}
			if tc.expectNilAccelerators && resourceDetails.Accelerators != nil {
				t.Fatalf("expected accelerators to remain nil, got %v", resourceDetails.Accelerators)
			}
			if !tc.expectNilAccelerators && resourceDetails.Accelerators == nil {
				t.Fatal("expected the explicit empty accelerator list to be preserved when decoding")
			}

			encoded, err := json.Marshal(resourceDetails)
			if err != nil {
				t.Fatalf("failed to encode resource details: %v", err)
			}

			payload := map[string]json.RawMessage{}
			if err := json.Unmarshal(encoded, &payload); err != nil {
				t.Fatalf("failed to decode resource details payload: %v", err)
			}
			if _, exists := payload["accelerators"]; exists {
				t.Fatal("expected an unset or empty accelerator list to be omitted")
			}
		})
	}
}

func TestResourceDetailsJSONAcceleratorRoundTrip(t *testing.T) {
	input := []byte(`{"accelerators":[{"provider":"kubevirt","resources":{"nvidia.com/GH100_H200_NVL":"0","nvidia.com/A100_80GB":"2"}}]}`)

	resourceDetails := ResourceDetails{}
	if err := json.Unmarshal(input, &resourceDetails); err != nil {
		t.Fatalf("failed to decode resource details: %v", err)
	}
	if len(resourceDetails.Accelerators) != 1 {
		t.Fatalf("expected one accelerator provider entry, got %d", len(resourceDetails.Accelerators))
	}

	accelerators := resourceDetails.Accelerators[0]
	if accelerators.Provider != kubeVirtAcceleratorProvider {
		t.Fatalf("expected provider %q, got %q", kubeVirtAcceleratorProvider, accelerators.Provider)
	}
	if len(accelerators.Resources) != 2 {
		t.Fatalf("expected two accelerator resources, got %d", len(accelerators.Resources))
	}
	assertQuantityEqual(t, resourceListQuantityPtr(accelerators.Resources, h200ResourceName), "0")
	assertQuantityEqual(t, resourceListQuantityPtr(accelerators.Resources, a100ResourceName), "2")
	if _, exists := accelerators.Resources[corev1.ResourceName("kubevirt/"+string(h200ResourceName))]; exists {
		t.Fatal("provider must not be encoded into the accelerator resource name")
	}

	encoded, err := json.Marshal(resourceDetails)
	if err != nil {
		t.Fatalf("failed to encode resource details: %v", err)
	}
	assertJSONEqual(t, encoded, input)
}

func TestResourceDetailsIsEmpty(t *testing.T) {
	testCases := []struct {
		name            string
		resourceDetails ResourceDetails
		expected        bool
	}{
		{
			name:     "empty resource details",
			expected: true,
		},
		{
			name: "empty accelerator list",
			resourceDetails: ResourceDetails{
				Accelerators: []AcceleratorQuota{},
			},
			expected: true,
		},
		{
			name: "accelerator-only details retain existing empty semantics",
			resourceDetails: ResourceDetails{
				Accelerators: []AcceleratorQuota{{
					Provider: kubeVirtAcceleratorProvider,
					Resources: corev1.ResourceList{
						h200ResourceName: resource.MustParse("2"),
					},
				}},
			},
			expected: true,
		},
		{
			name: "zero scalar resource quantity",
			resourceDetails: ResourceDetails{
				CPU: quantityPtr("0"),
			},
			expected: true,
		},
		{
			name: "non-zero scalar resource quantity",
			resourceDetails: ResourceDetails{
				CPU: quantityPtr("1"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := tc.resourceDetails.IsEmpty(); actual != tc.expected {
				t.Fatalf("expected IsEmpty() to return %t, got %t", tc.expected, actual)
			}
		})
	}
}

func TestResourceDetailsDeepCopy(t *testing.T) {
	cpu := resource.MustParse("2")
	cpu.AsDec()
	h200 := resource.MustParse("4")
	h200.AsDec()
	original := ResourceDetails{
		CPU: &cpu,
		Accelerators: []AcceleratorQuota{{
			Provider: kubeVirtAcceleratorProvider,
			Resources: corev1.ResourceList{
				h200ResourceName: h200,
			},
		}},
	}
	copied := original.DeepCopy()

	original.CPU.Add(resource.MustParse("1"))
	original.Accelerators[0].Provider = "changed"
	quantity := original.Accelerators[0].Resources[h200ResourceName]
	quantity.Add(resource.MustParse("1"))
	original.Accelerators[0].Resources[h200ResourceName] = quantity
	original.Accelerators[0].Resources[a100ResourceName] = resource.MustParse("1")
	original.Accelerators = append(original.Accelerators, AcceleratorQuota{Provider: "changed-again"})

	assertQuantityEqual(t, copied.CPU, "2")
	if len(copied.Accelerators) != 1 {
		t.Fatalf("expected one copied accelerator provider entry, got %d", len(copied.Accelerators))
	}
	if copied.Accelerators[0].Provider != kubeVirtAcceleratorProvider {
		t.Fatalf("expected copied provider %q, got %q", kubeVirtAcceleratorProvider, copied.Accelerators[0].Provider)
	}
	if len(copied.Accelerators[0].Resources) != 1 {
		t.Fatalf("expected one copied accelerator resource, got %d", len(copied.Accelerators[0].Resources))
	}
	assertQuantityEqual(t, resourceListQuantityPtr(copied.Accelerators[0].Resources, h200ResourceName), "4")
}

func TestNewResourceDetailsWithAcceleratorsCopiesInputs(t *testing.T) {
	cpu := resource.MustParse("2")
	memory := resource.MustParse("4Gi")
	storage := resource.MustParse("10Gi")
	h200 := resource.MustParse("4")
	for _, quantity := range []*resource.Quantity{&cpu, &memory, &storage, &h200} {
		quantity.AsDec()
	}

	accelerators := []AcceleratorQuota{{
		Provider: kubeVirtAcceleratorProvider,
		Resources: corev1.ResourceList{
			h200ResourceName: h200,
		},
	}}
	resourceDetails := NewResourceDetailsWithAccelerators(cpu, memory, storage, accelerators...)

	cpu.Add(resource.MustParse("1"))
	memory.Add(resource.MustParse("1Gi"))
	storage.Add(resource.MustParse("1Gi"))
	accelerators[0].Provider = "changed"
	quantity := accelerators[0].Resources[h200ResourceName]
	quantity.Add(resource.MustParse("1"))
	accelerators[0].Resources[h200ResourceName] = quantity
	accelerators[0].Resources[a100ResourceName] = resource.MustParse("1")

	assertQuantityEqual(t, resourceDetails.CPU, "2")
	assertQuantityEqual(t, resourceDetails.Memory, "4Gi")
	assertQuantityEqual(t, resourceDetails.Storage, "10Gi")
	if len(resourceDetails.Accelerators) != 1 {
		t.Fatalf("expected one accelerator provider entry, got %d", len(resourceDetails.Accelerators))
	}
	if resourceDetails.Accelerators[0].Provider != kubeVirtAcceleratorProvider {
		t.Fatalf("expected provider %q, got %q", kubeVirtAcceleratorProvider, resourceDetails.Accelerators[0].Provider)
	}
	if len(resourceDetails.Accelerators[0].Resources) != 1 {
		t.Fatalf("expected one accelerator resource, got %d", len(resourceDetails.Accelerators[0].Resources))
	}
	assertQuantityEqual(t, resourceListQuantityPtr(resourceDetails.Accelerators[0].Resources, h200ResourceName), "4")

	withoutAccelerators := NewResourceDetails(resource.Quantity{}, resource.Quantity{}, resource.Quantity{})
	if withoutAccelerators.Accelerators != nil {
		t.Fatalf("expected backwards-compatible constructor call to leave accelerators nil, got %v", withoutAccelerators.Accelerators)
	}
}

func TestNewResourceDetailsRetainsOriginalFunctionSignature(t *testing.T) {
	type constructorFunc func(resource.Quantity, resource.Quantity, resource.Quantity) *ResourceDetails
	constructor := constructorFunc(NewResourceDetails)

	resourceDetails := constructor(resource.MustParse("2"), resource.MustParse("4Gi"), resource.MustParse("10Gi"))
	assertQuantityEqual(t, resourceDetails.CPU, "2")
	assertQuantityEqual(t, resourceDetails.Memory, "4Gi")
	assertQuantityEqual(t, resourceDetails.Storage, "10Gi")
	if resourceDetails.Accelerators != nil {
		t.Fatalf("expected original constructor to leave accelerators nil, got %v", resourceDetails.Accelerators)
	}
}

func resourceListQuantityPtr(resources corev1.ResourceList, name corev1.ResourceName) *resource.Quantity {
	quantity, ok := resources[name]
	if !ok {
		return nil
	}
	return &quantity
}

func assertJSONEqual(t *testing.T, actual, expected []byte) {
	t.Helper()

	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("failed to decode actual JSON: %v", err)
	}
	var expectedValue any
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatalf("failed to decode expected JSON: %v", err)
	}
	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("JSON differs:\nactual:   %s\nexpected: %s", actual, expected)
	}
}

func assertQuantityEqual(t *testing.T, actual *resource.Quantity, expected string) {
	t.Helper()
	if actual == nil {
		t.Fatalf("expected quantity %q, got nil", expected)
	}

	expectedQuantity := resource.MustParse(expected)
	if !actual.Equal(expectedQuantity) {
		t.Fatalf("expected quantity %q, got %q", expected, actual.String())
	}
}

func quantityPtr(value string) *resource.Quantity {
	quantity := resource.MustParse(value)
	return &quantity
}
