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
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestAcceleratorQuotaCanonicalHelpers(t *testing.T) {
	emptyDigest := AcceleratorQuotaDigestFor(nil)
	if !strings.HasPrefix(string(emptyDigest), "sha256:") || len(emptyDigest) != len("sha256:")+sha256HexLength {
		t.Fatalf("expected a sha256 digest, got %q", emptyDigest)
	}
	if emptyDigest != AcceleratorQuotaDigestFor([]AcceleratorQuota{}) {
		t.Fatal("expected nil and empty accelerator quota lists to have the same digest")
	}
	if !AcceleratorQuotasEqual(nil, []AcceleratorQuota{}) {
		t.Fatal("expected nil and empty accelerator quota lists to be equal")
	}

	a := []AcceleratorQuota{
		{
			Provider: "future-provider",
			Resources: corev1.ResourceList{
				corev1.ResourceName("example.com/device"): resource.MustParse("1Gi"),
			},
		},
		{
			Provider: kubeVirtAcceleratorProvider,
			Resources: corev1.ResourceList{
				h200ResourceName: resource.MustParse("1"),
				a100ResourceName: resource.MustParse("2"),
			},
		},
	}
	b := []AcceleratorQuota{
		{
			Provider: kubeVirtAcceleratorProvider,
			Resources: corev1.ResourceList{
				a100ResourceName: resource.MustParse("2000m"),
				h200ResourceName: resource.MustParse("1000m"),
			},
		},
		{
			Provider: "future-provider",
			Resources: corev1.ResourceList{
				corev1.ResourceName("example.com/device"): resource.MustParse("1024Mi"),
			},
		},
	}

	if !AcceleratorQuotasEqual(a, b) {
		t.Fatal("expected provider order, resource order, and quantity format to be canonicalized")
	}
	if AcceleratorQuotaDigestFor(a) != AcceleratorQuotaDigestFor(b) {
		t.Fatal("expected semantically equal accelerator quotas to have the same digest")
	}

	different := b[0].DeepCopy()
	different.Resources[h200ResourceName] = resource.MustParse("2")
	b[0] = *different
	if AcceleratorQuotasEqual(a, b) {
		t.Fatal("expected different accelerator quantities not to compare equal")
	}
	if AcceleratorQuotaDigestFor(a) == AcceleratorQuotaDigestFor(b) {
		t.Fatal("expected different accelerator quantities to have different digests")
	}
}

func TestAddAcceleratorUsage(t *testing.T) {
	existingResources := corev1.ResourceList{
		h200ResourceName: resource.MustParse("1"),
	}
	target := ResourceDetails{
		Accelerators: []AcceleratorQuota{{
			Provider:  kubeVirtAcceleratorProvider,
			Resources: existingResources,
		}},
	}

	incomingResources := corev1.ResourceList{
		h200ResourceName: resource.MustParse("2000m"),
		a100ResourceName: resource.MustParse("1024Mi"),
	}
	usage := []AcceleratorQuota{
		{
			Provider: kubeVirtAcceleratorProvider,
			Resources: corev1.ResourceList{
				a100ResourceName: resource.MustParse("0"),
			},
		},
		{
			Provider: "future-provider",
			Resources: corev1.ResourceList{
				corev1.ResourceName("example.com/device"): resource.MustParse("1"),
			},
		},
		{
			Provider:  kubeVirtAcceleratorProvider,
			Resources: incomingResources,
		},
	}

	AddAcceleratorUsage(&target, usage)

	if len(target.Accelerators) != 2 {
		t.Fatalf("expected two canonical provider entries, got %d", len(target.Accelerators))
	}
	if target.Accelerators[0].Provider != "future-provider" || target.Accelerators[1].Provider != kubeVirtAcceleratorProvider {
		t.Fatalf("expected provider entries in canonical order, got %q then %q", target.Accelerators[0].Provider, target.Accelerators[1].Provider)
	}
	assertQuantityEqual(t, resourceListQuantityPtr(target.Accelerators[0].Resources, corev1.ResourceName("example.com/device")), "1")
	assertQuantityEqual(t, resourceListQuantityPtr(target.Accelerators[1].Resources, h200ResourceName), "3")
	assertQuantityEqual(t, resourceListQuantityPtr(target.Accelerators[1].Resources, a100ResourceName), "1Gi")

	h200 := target.Accelerators[1].Resources[h200ResourceName]
	if actual := h200.String(); actual != "3" {
		t.Fatalf("expected canonical DecimalSI quantity representation, got %q", actual)
	}
	a100 := target.Accelerators[1].Resources[a100ResourceName]
	if actual := a100.String(); actual != "1073741824" {
		t.Fatalf("expected format-independent canonical quantity representation, got %q", actual)
	}

	// Mutating the source maps after aggregation must not affect the result.
	existingResources[h200ResourceName] = resource.MustParse("100")
	incomingResources[h200ResourceName] = resource.MustParse("100")
	assertQuantityEqual(t, resourceListQuantityPtr(target.Accelerators[1].Resources, h200ResourceName), "3")

	empty := ResourceDetails{Accelerators: []AcceleratorQuota{}}
	AddAcceleratorUsage(&empty, []AcceleratorQuota{{Provider: kubeVirtAcceleratorProvider, Resources: corev1.ResourceList{}}})
	if empty.Accelerators != nil {
		t.Fatalf("expected empty usage to normalize to nil, got %#v", empty.Accelerators)
	}
}

func TestAcceleratorAccountingStatusJSONRoundTrip(t *testing.T) {
	observedAt := metav1.NewTime(time.Date(2026, time.August, 14, 12, 34, 56, 0, time.UTC))
	clusterStatus := ClusterStatus{
		AcceleratorAccounting: &ClusterAcceleratorAccountingStatus{
			ObservedAccountingRevision:   "revision-1",
			ObservedQuotaDigest:          AcceleratorQuotaDigestFor(nil),
			FootprintSchemaVersion:       "v1alpha1",
			ControllerVersion:            "v2.30.0",
			ObservedAt:                   observedAt,
			MachinesWithoutFootprint:     0,
			MachinesWithInvalidFootprint: 1,
			Ready:                        false,
			Blockers: []AcceleratorAccountingBlocker{{
				Type:        AcceleratorAccountingBlockerTypeInvalidFootprints,
				Message:     "one Machine has an invalid footprint",
				ClusterName: "cluster-a",
				Count:       1,
			}},
		},
	}

	clusterJSON, err := json.Marshal(clusterStatus)
	if err != nil {
		t.Fatalf("failed to encode cluster status: %v", err)
	}
	clusterPayload := map[string]json.RawMessage{}
	if err := json.Unmarshal(clusterJSON, &clusterPayload); err != nil {
		t.Fatalf("failed to decode cluster status payload: %v", err)
	}
	accountingJSON, exists := clusterPayload["acceleratorAccounting"]
	if !exists {
		t.Fatal("expected acceleratorAccounting in the ClusterStatus JSON payload")
	}
	assertJSONEqual(t, accountingJSON, []byte(`{
			"observedAccountingRevision":"revision-1",
			"observedQuotaDigest":"`+string(AcceleratorQuotaDigestFor(nil))+`",
			"footprintSchemaVersion":"v1alpha1",
			"controllerVersion":"v2.30.0",
			"observedAt":"2026-08-14T12:34:56Z",
			"machinesWithoutFootprint":0,
			"machinesWithInvalidFootprint":1,
			"ready":false,
			"blockers":[{"type":"InvalidFootprints","message":"one Machine has an invalid footprint","clusterName":"cluster-a","count":1}]
	}`))

	decodedClusterStatus := ClusterStatus{}
	if err := json.Unmarshal(clusterJSON, &decodedClusterStatus); err != nil {
		t.Fatalf("failed to decode cluster status: %v", err)
	}
	if !decodedClusterStatus.AcceleratorAccounting.ObservedAt.Time.Equal(clusterStatus.AcceleratorAccounting.ObservedAt.Time) {
		t.Fatalf("cluster observedAt changed during round trip: got %v, want %v", decodedClusterStatus.AcceleratorAccounting.ObservedAt, clusterStatus.AcceleratorAccounting.ObservedAt)
	}
	decodedClusterStatus.AcceleratorAccounting.ObservedAt = clusterStatus.AcceleratorAccounting.ObservedAt
	if !reflect.DeepEqual(clusterStatus.AcceleratorAccounting, decodedClusterStatus.AcceleratorAccounting) {
		t.Fatalf("cluster accelerator accounting status changed during round trip:\nactual:   %#v\nexpected: %#v", decodedClusterStatus.AcceleratorAccounting, clusterStatus.AcceleratorAccounting)
	}

	quotaStatus := ResourceQuotaStatus{
		LocalAcceleratorAccounting: &ResourceQuotaLocalAcceleratorAccountingStatus{
			ObservedAccountingRevision:     "revision-1",
			ObservedQuotaDigest:            AcceleratorQuotaDigestFor(nil),
			ObservedAt:                     observedAt,
			LegacyMachinesWithoutFootprint: 2,
			MachinesWithInvalidFootprint:   0,
			Ready:                          false,
			Blockers: []AcceleratorAccountingBlocker{{
				Type:     AcceleratorAccountingBlockerTypeLegacyMachines,
				SeedName: "seed-a",
				Count:    2,
			}},
		},
		GlobalAcceleratorAccounting: &ResourceQuotaGlobalAcceleratorAccountingStatus{
			ActivationPhase:                AcceleratorAccountingPhaseBlocked,
			ObservedAccountingRevision:     "revision-1",
			ObservedQuotaDigest:            AcceleratorQuotaDigestFor(nil),
			ObservedAt:                     observedAt,
			LegacyMachinesWithoutFootprint: 2,
			MachinesWithInvalidFootprint:   0,
			Ready:                          false,
			Blockers: []AcceleratorAccountingBlocker{{
				Type:     AcceleratorAccountingBlockerTypeLegacyMachines,
				SeedName: "seed-a",
				Count:    2,
			}},
		},
	}

	quotaJSON, err := json.Marshal(quotaStatus)
	if err != nil {
		t.Fatalf("failed to encode ResourceQuota status: %v", err)
	}
	quotaPayload := map[string]json.RawMessage{}
	if err := json.Unmarshal(quotaJSON, &quotaPayload); err != nil {
		t.Fatalf("failed to decode ResourceQuota status payload: %v", err)
	}
	if _, exists := quotaPayload["localAcceleratorAccounting"]; !exists {
		t.Fatal("expected localAcceleratorAccounting in the ResourceQuotaStatus JSON payload")
	}
	if _, exists := quotaPayload["globalAcceleratorAccounting"]; !exists {
		t.Fatal("expected globalAcceleratorAccounting in the ResourceQuotaStatus JSON payload")
	}
	decodedQuotaStatus := ResourceQuotaStatus{}
	if err := json.Unmarshal(quotaJSON, &decodedQuotaStatus); err != nil {
		t.Fatalf("failed to decode ResourceQuota status: %v", err)
	}
	if !decodedQuotaStatus.LocalAcceleratorAccounting.ObservedAt.Time.Equal(quotaStatus.LocalAcceleratorAccounting.ObservedAt.Time) {
		t.Fatalf("local observedAt changed during round trip: got %v, want %v", decodedQuotaStatus.LocalAcceleratorAccounting.ObservedAt, quotaStatus.LocalAcceleratorAccounting.ObservedAt)
	}
	if !decodedQuotaStatus.GlobalAcceleratorAccounting.ObservedAt.Time.Equal(quotaStatus.GlobalAcceleratorAccounting.ObservedAt.Time) {
		t.Fatalf("global observedAt changed during round trip: got %v, want %v", decodedQuotaStatus.GlobalAcceleratorAccounting.ObservedAt, quotaStatus.GlobalAcceleratorAccounting.ObservedAt)
	}
	decodedQuotaStatus.LocalAcceleratorAccounting.ObservedAt = quotaStatus.LocalAcceleratorAccounting.ObservedAt
	decodedQuotaStatus.GlobalAcceleratorAccounting.ObservedAt = quotaStatus.GlobalAcceleratorAccounting.ObservedAt
	if !reflect.DeepEqual(quotaStatus, decodedQuotaStatus) {
		t.Fatalf("ResourceQuota accelerator accounting status changed during round trip:\nactual:   %#v\nexpected: %#v", decodedQuotaStatus, quotaStatus)
	}
}

func TestAcceleratorAccountingStatusDeepCopy(t *testing.T) {
	clusterStatus := ClusterStatus{
		AcceleratorAccounting: &ClusterAcceleratorAccountingStatus{
			Blockers: []AcceleratorAccountingBlocker{{Message: "cluster original"}},
		},
	}
	clusterCopy := clusterStatus.DeepCopy()
	clusterStatus.AcceleratorAccounting.Blockers[0].Message = "cluster changed"
	if actual := clusterCopy.AcceleratorAccounting.Blockers[0].Message; actual != "cluster original" {
		t.Fatalf("cluster status deepcopy retained blocker slice, got %q", actual)
	}

	quotaStatus := ResourceQuotaStatus{
		LocalAcceleratorAccounting: &ResourceQuotaLocalAcceleratorAccountingStatus{
			Blockers: []AcceleratorAccountingBlocker{{Message: "local original"}},
		},
		GlobalAcceleratorAccounting: &ResourceQuotaGlobalAcceleratorAccountingStatus{
			Blockers: []AcceleratorAccountingBlocker{{Message: "global original"}},
		},
	}
	quotaCopy := quotaStatus.DeepCopy()
	quotaStatus.LocalAcceleratorAccounting.Blockers[0].Message = "local changed"
	quotaStatus.GlobalAcceleratorAccounting.Blockers[0].Message = "global changed"
	if actual := quotaCopy.LocalAcceleratorAccounting.Blockers[0].Message; actual != "local original" {
		t.Fatalf("local status deepcopy retained blocker slice, got %q", actual)
	}
	if actual := quotaCopy.GlobalAcceleratorAccounting.Blockers[0].Message; actual != "global original" {
		t.Fatalf("global status deepcopy retained blocker slice, got %q", actual)
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

const sha256HexLength = 64
