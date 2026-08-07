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
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestResourceDetailsJSONBackwardCompatibility(t *testing.T) {
	resourceDetails := ResourceDetails{}
	if err := json.Unmarshal([]byte(`{"cpu":"2","memory":"4Gi","storage":"10Gi"}`), &resourceDetails); err != nil {
		t.Fatalf("failed to decode resource details: %v", err)
	}
	if resourceDetails.Accelerators != nil {
		t.Fatalf("expected accelerators to remain nil, got %v", resourceDetails.Accelerators)
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
		t.Fatal("expected an unset accelerator map to be omitted")
	}
}

func TestResourceDetailsJSONPreservesExplicitZeroAcceleratorQuota(t *testing.T) {
	resourceDetails := ResourceDetails{
		Accelerators: map[string]resource.Quantity{
			"nvidia.com/GH100_H200_NVL": resource.MustParse("0"),
		},
	}

	encoded, err := json.Marshal(resourceDetails)
	if err != nil {
		t.Fatalf("failed to encode resource details: %v", err)
	}

	decoded := ResourceDetails{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("failed to decode resource details: %v", err)
	}

	quantity, exists := decoded.Accelerators["nvidia.com/GH100_H200_NVL"]
	if !exists {
		t.Fatal("expected explicit zero accelerator quota to be preserved")
	}
	if !quantity.IsZero() {
		t.Fatalf("expected explicit zero accelerator quota, got %s", quantity.String())
	}
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
			name: "empty accelerator map",
			resourceDetails: ResourceDetails{
				Accelerators: map[string]resource.Quantity{},
			},
			expected: true,
		},
		{
			name: "zero accelerator quantity",
			resourceDetails: ResourceDetails{
				Accelerators: map[string]resource.Quantity{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("0"),
				},
			},
			expected: false,
		},
		{
			name: "non-zero accelerator quantity",
			resourceDetails: ResourceDetails{
				Accelerators: map[string]resource.Quantity{
					"nvidia.com/GH100_H200_NVL": resource.MustParse("2"),
				},
			},
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

func TestResourceDetailsAdd(t *testing.T) {
	memory := resource.MustParse("4Gi")
	memory.AsDec()
	a100 := resource.MustParse("1")
	a100.AsDec()

	resourceDetails := ResourceDetails{
		CPU: quantityPtr("1"),
		Accelerators: map[string]resource.Quantity{
			"nvidia.com/GH100_H200_NVL": resource.MustParse("1"),
		},
	}
	other := ResourceDetails{
		CPU:     quantityPtr("2"),
		Memory:  &memory,
		Storage: quantityPtr("10Gi"),
		Accelerators: map[string]resource.Quantity{
			"nvidia.com/GH100_H200_NVL": resource.MustParse("2"),
			"nvidia.com/A100_80GB":      a100,
		},
	}

	resourceDetails.Add(other)

	assertQuantityEqual(t, resourceDetails.CPU, "3")
	assertQuantityEqual(t, resourceDetails.Memory, "4Gi")
	assertQuantityEqual(t, resourceDetails.Storage, "10Gi")
	assertQuantityEqual(t, acceleratorQuantityPtr(resourceDetails, "nvidia.com/GH100_H200_NVL"), "3")
	assertQuantityEqual(t, acceleratorQuantityPtr(resourceDetails, "nvidia.com/A100_80GB"), "1")

	// Mutating the source after Add must not mutate the destination.
	other.Memory.Add(resource.MustParse("1Gi"))
	otherAccelerator := other.Accelerators["nvidia.com/A100_80GB"]
	otherAccelerator.Add(resource.MustParse("1"))
	other.Accelerators["nvidia.com/A100_80GB"] = otherAccelerator

	assertQuantityEqual(t, resourceDetails.Memory, "4Gi")
	assertQuantityEqual(t, acceleratorQuantityPtr(resourceDetails, "nvidia.com/A100_80GB"), "1")
}

func TestResourceDetailsDeepCopy(t *testing.T) {
	original := ResourceDetails{
		Accelerators: map[string]resource.Quantity{
			"nvidia.com/GH100_H200_NVL": resource.MustParse("1"),
		},
	}
	copied := original.DeepCopy()

	quantity := original.Accelerators["nvidia.com/GH100_H200_NVL"]
	quantity.Add(resource.MustParse("1"))
	original.Accelerators["nvidia.com/GH100_H200_NVL"] = quantity

	assertQuantityEqual(t, acceleratorQuantityPtr(*copied, "nvidia.com/GH100_H200_NVL"), "1")
}

func acceleratorQuantityPtr(resourceDetails ResourceDetails, name string) *resource.Quantity {
	quantity, ok := resourceDetails.Accelerators[name]
	if !ok {
		return nil
	}
	return &quantity
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
