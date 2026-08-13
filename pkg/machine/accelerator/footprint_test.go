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

package accelerator

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestEncodeCanonicalFootprint(t *testing.T) {
	resources := corev1.ResourceList{
		"nvidia.com/H200":  resource.MustParse("2"),
		"example.com/fpga": resource.MustParse("1"),
	}
	footprint := NewKubeVirtFootprint(resources)

	// The constructor must isolate the trusted value from caller mutations.
	resources["nvidia.com/H200"] = resource.MustParse("9")
	resources["example.com/new"] = resource.MustParse("1")

	encoded, err := Encode(footprint)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	const expected = `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"example.com/fpga":"1","nvidia.com/H200":"2"}}`
	if encoded != expected {
		t.Fatalf("Encode() = %s, want %s", encoded, expected)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	h200 := decoded.Resources["nvidia.com/H200"]
	if h200.Cmp(resource.MustParse("2")) != 0 {
		t.Fatalf("decoded H200 quantity = %s, want 2", h200.String())
	}
}

func TestEmptyFootprint(t *testing.T) {
	encoded, err := Encode(NewKubeVirtFootprint(nil))
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	const expected = `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{}}`
	if encoded != expected {
		t.Fatalf("Encode() = %s, want %s", encoded, expected)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded.Resources == nil || len(decoded.Resources) != 0 {
		t.Fatalf("decoded resources = %#v, want non-nil empty map", decoded.Resources)
	}
}

func TestNewKubeVirtFootprintDeepCopiesLargeQuantity(t *testing.T) {
	const resourceName = "nvidia.com/H200"
	resources := corev1.ResourceList{resourceName: resource.MustParse("9223372036854775808")}
	footprint := NewKubeVirtFootprint(resources)

	quantity := resources[resourceName]
	quantity.Add(resource.MustParse("1"))

	got := footprint.Resources[resourceName]
	if got.Cmp(resource.MustParse("9223372036854775808")) != 0 {
		t.Fatalf("copied quantity = %s, want original large value", got.String())
	}
}

func TestDecodeNormalizesResourceQuantities(t *testing.T) {
	decoded, err := Decode(`{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"nvidia.com/H200":"1000m"}}`)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	encoded, err := Encode(decoded)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if !strings.Contains(encoded, `"nvidia.com/H200":"1"`) {
		t.Fatalf("Encode() = %s, want canonical whole-number quantity", encoded)
	}
}

func TestDecodeRejectsInvalidFootprints(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		errorString string
	}{
		{name: "malformed JSON", value: `{`, errorString: "failed to decode"},
		{name: "trailing JSON", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{}} {}`, errorString: "failed to decode"},
		{name: "unknown field", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{},"unknown":true}`, errorString: "unknown field"},
		{name: "duplicate top-level field", value: `{"schemaVersion":"v1alpha1","schemaVersion":"v1alpha1","provider":"kubevirt","resources":{}}`, errorString: "duplicate field"},
		{name: "duplicate resource", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"nvidia.com/H200":"1","nvidia.com/H200":"2"}}`, errorString: "duplicate field"},
		{name: "missing schema", value: `{"provider":"kubevirt","resources":{}}`, errorString: "schemaVersion"},
		{name: "unsupported schema", value: `{"schemaVersion":"v2","provider":"kubevirt","resources":{}}`, errorString: "unsupported accelerator footprint schemaVersion"},
		{name: "missing provider", value: `{"schemaVersion":"v1alpha1","resources":{}}`, errorString: "provider"},
		{name: "unsupported provider", value: `{"schemaVersion":"v1alpha1","provider":"aws","resources":{}}`, errorString: "unsupported accelerator footprint provider"},
		{name: "missing resources", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt"}`, errorString: "resources must not be null"},
		{name: "null resources", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":null}`, errorString: "resources must not be null"},
		{name: "unqualified resource", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"H200":"1"}}`, errorString: "qualified name"},
		{name: "multiple slashes", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"nvidia.com/gpu/H200":"1"}}`, errorString: "qualified name"},
		{name: "malformed resource", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"NVIDIA!.com/H200":"1"}}`, errorString: "not a valid qualified name"},
		{name: "zero quantity", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"nvidia.com/H200":"0"}}`, errorString: "positive quantity"},
		{name: "negative quantity", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"nvidia.com/H200":"-1"}}`, errorString: "positive quantity"},
		{name: "fractional quantity", value: `{"schemaVersion":"v1alpha1","provider":"kubevirt","resources":{"nvidia.com/H200":"500m"}}`, errorString: "whole-number quantity"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(tc.value)
			if err == nil {
				t.Fatal("Decode() expected an error")
			}
			if !strings.Contains(err.Error(), tc.errorString) {
				t.Fatalf("Decode() error = %q, want substring %q", err, tc.errorString)
			}
		})
	}
}

func TestEncodeAcceptsLargeWholeQuantity(t *testing.T) {
	_, err := Encode(Footprint{
		SchemaVersion: SchemaVersionV1Alpha1,
		Provider:      ProviderKubeVirt,
		Resources: corev1.ResourceList{
			"nvidia.com/H200": resource.MustParse("9223372036854775808"),
		},
	})
	if err != nil {
		t.Fatalf("Encode() rejected a large whole-number quantity: %v", err)
	}
}

func TestIsReservedAnnotationKey(t *testing.T) {
	for _, key := range []string{FootprintAnnotationKey, AnnotationPrefix + "future-field"} {
		if !IsReservedAnnotationKey(key) {
			t.Errorf("IsReservedAnnotationKey(%q) = false, want true", key)
		}
	}
	for _, key := range []string{"", "accelerators.kubermatic.io", "example.com/accelerators.kubermatic.io/footprint"} {
		if IsReservedAnnotationKey(key) {
			t.Errorf("IsReservedAnnotationKey(%q) = true, want false", key)
		}
	}
}
