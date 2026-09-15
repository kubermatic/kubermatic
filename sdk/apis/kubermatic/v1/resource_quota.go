/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ResourceQuotaKindName represents "Kind" defined in Kubernetes.
	ResourceQuotaKindName = "ResourceQuota"

	ResourceQuotaSubjectNameLabelKey = "subject-name"
	ResourceQuotaSubjectKindLabelKey = "subject-kind"

	ProjectSubjectKind = "project"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"
// +kubebuilder:printcolumn:JSONPath=".spec.subject.name",name="Subject Name",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.subject.kind",name="Subject Kind",type="string"

// ResourceQuota specifies the amount of cluster resources a project can use.
type ResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the desired state of the resource quota.
	Spec ResourceQuotaSpec `json:"spec,omitempty"`
	// Status holds the current state of the resource quota.
	Status ResourceQuotaStatus `json:"status,omitempty"`
}

// ResourceQuotaSpec describes the desired state of a resource quota.
type ResourceQuotaSpec struct {
	// Subject specifies to which entity the quota applies to.
	Subject Subject `json:"subject"`
	// Quota specifies the current maximum allowed usage of resources.
	Quota ResourceDetails `json:"quota"`
}

// ResourceQuotaStatus describes the current state of a resource quota.
type ResourceQuotaStatus struct {
	// GlobalUsage is holds the current usage of resources for all seeds.
	GlobalUsage ResourceDetails `json:"globalUsage,omitempty"`
	// LocalUsage is holds the current usage of resources for the local seed.
	LocalUsage ResourceDetails `json:"localUsage,omitempty"`

	// LocalAcceleratorAccounting contains the Seed-owned accelerator accounting
	// attestation for this ResourceQuota. It is not synchronized from the master.
	// +optional
	LocalAcceleratorAccounting *ResourceQuotaLocalAcceleratorAccountingStatus `json:"localAcceleratorAccounting,omitempty"`

	// GlobalAcceleratorAccounting contains the master-owned project-wide accelerator
	// accounting state. It is synchronized from the master to every Seed copy.
	// +optional
	GlobalAcceleratorAccounting *ResourceQuotaGlobalAcceleratorAccountingStatus `json:"globalAcceleratorAccounting,omitempty"`
}

// AcceleratorAccountingRevision is an opaque master-issued identity for an accelerator
// accounting transition. A new revision is required even when a quota changes back to a
// previously used value, so an old attestation cannot satisfy a later transition.
type AcceleratorAccountingRevision string

// AcceleratorQuotaDigest is the SHA-256 digest of a canonical accelerator quota.
type AcceleratorQuotaDigest string

// +kubebuilder:validation:Enum=Activating;Ready;Blocked

// AcceleratorAccountingPhase describes the observed project-wide activation state.
type AcceleratorAccountingPhase string

const (
	// AcceleratorAccountingPhaseActivating means the current accounting revision has
	// not yet received all required fresh attestations.
	AcceleratorAccountingPhaseActivating AcceleratorAccountingPhase = "Activating"

	// AcceleratorAccountingPhaseReady means every required reporter has provided a
	// compatible, fresh attestation for the current accounting revision and quota digest.
	AcceleratorAccountingPhaseReady AcceleratorAccountingPhase = "Ready"

	// AcceleratorAccountingPhaseBlocked means one or more actionable blockers prevent
	// accelerator accounting from becoming ready.
	AcceleratorAccountingPhaseBlocked AcceleratorAccountingPhase = "Blocked"
)

// +kubebuilder:validation:Enum=LegacyMachines;InvalidFootprints;UnsupportedFootprintSchema;IncompatibleControllerVersion;StaleHeartbeat;NewCluster;UnreachableSeed;MissingReport;RevisionMismatch;QuotaDigestMismatch

// AcceleratorAccountingBlockerType identifies why accelerator accounting is not ready.
type AcceleratorAccountingBlockerType string

const (
	AcceleratorAccountingBlockerTypeLegacyMachines                AcceleratorAccountingBlockerType = "LegacyMachines"
	AcceleratorAccountingBlockerTypeInvalidFootprints             AcceleratorAccountingBlockerType = "InvalidFootprints"
	AcceleratorAccountingBlockerTypeUnsupportedFootprintSchema    AcceleratorAccountingBlockerType = "UnsupportedFootprintSchema"
	AcceleratorAccountingBlockerTypeIncompatibleControllerVersion AcceleratorAccountingBlockerType = "IncompatibleControllerVersion"
	AcceleratorAccountingBlockerTypeStaleHeartbeat                AcceleratorAccountingBlockerType = "StaleHeartbeat"
	AcceleratorAccountingBlockerTypeNewCluster                    AcceleratorAccountingBlockerType = "NewCluster"
	AcceleratorAccountingBlockerTypeUnreachableSeed               AcceleratorAccountingBlockerType = "UnreachableSeed"
	AcceleratorAccountingBlockerTypeMissingReport                 AcceleratorAccountingBlockerType = "MissingReport"
	AcceleratorAccountingBlockerTypeRevisionMismatch              AcceleratorAccountingBlockerType = "RevisionMismatch"
	AcceleratorAccountingBlockerTypeQuotaDigestMismatch           AcceleratorAccountingBlockerType = "QuotaDigestMismatch"
)

// AcceleratorAccountingBlocker describes an actionable reason why accelerator accounting
// is not ready. SeedName and ClusterName identify the affected scope when applicable.
type AcceleratorAccountingBlocker struct {
	// Type identifies the class of blocker.
	Type AcceleratorAccountingBlockerType `json:"type"`

	// Message contains human-readable details about the blocker.
	// +optional
	Message string `json:"message,omitempty"`

	// SeedName identifies the affected Seed when the blocker is Seed-specific.
	// +optional
	SeedName string `json:"seedName,omitempty"`

	// ClusterName identifies the affected user cluster when the blocker is cluster-specific.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// Count is the number of affected objects when the blocker represents an aggregate.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Count int32 `json:"count,omitempty"`
}

// ResourceQuotaLocalAcceleratorAccountingStatus is the Seed-owned accelerator accounting
// attestation aggregated from all relevant KubeVirt clusters for one project on that Seed.
type ResourceQuotaLocalAcceleratorAccountingStatus struct {
	// ObservedAccountingRevision is the master-issued accounting revision this Seed observed.
	ObservedAccountingRevision AcceleratorAccountingRevision `json:"observedAccountingRevision"`

	// ObservedQuotaDigest is the canonical accelerator quota digest this Seed observed.
	ObservedQuotaDigest AcceleratorQuotaDigest `json:"observedQuotaDigest"`

	// ObservedAt is the oldest heartbeat among the cluster attestations included in this report.
	// +optional
	ObservedAt metav1.Time `json:"observedAt,omitempty"`

	// LegacyMachinesWithoutFootprint is the number of Machines that predate trusted footprint capture.
	// +kubebuilder:validation:Minimum=0
	LegacyMachinesWithoutFootprint int32 `json:"legacyMachinesWithoutFootprint"`

	// MachinesWithInvalidFootprint is the number of Machines whose footprint cannot be accounted.
	// +kubebuilder:validation:Minimum=0
	MachinesWithInvalidFootprint int32 `json:"machinesWithInvalidFootprint"`

	// Ready is true when every relevant cluster has provided a compatible, fresh attestation
	// for ObservedAccountingRevision and ObservedQuotaDigest and no blocker remains.
	Ready bool `json:"ready"`

	// Blockers contains actionable reasons why this Seed-local attestation is not ready.
	// +optional
	Blockers []AcceleratorAccountingBlocker `json:"blockers,omitempty"`
}

// ResourceQuotaGlobalAcceleratorAccountingStatus is the master-owned accelerator accounting
// state aggregated from every configured Seed for one project.
type ResourceQuotaGlobalAcceleratorAccountingStatus struct {
	// ActivationPhase describes the current project-wide accelerator accounting state.
	ActivationPhase AcceleratorAccountingPhase `json:"activationPhase"`

	// ObservedAccountingRevision is the authoritative master-issued current accounting
	// revision. Despite the Observed prefix, child reporters must attest to this exact value.
	ObservedAccountingRevision AcceleratorAccountingRevision `json:"observedAccountingRevision"`

	// ObservedQuotaDigest is the authoritative canonical digest of the current accelerator
	// quota. Despite the Observed prefix, child reporters must attest to this exact value.
	ObservedQuotaDigest AcceleratorQuotaDigest `json:"observedQuotaDigest"`

	// ObservedAt is the oldest heartbeat among the Seed attestations included in this state.
	// +optional
	ObservedAt metav1.Time `json:"observedAt,omitempty"`

	// LegacyMachinesWithoutFootprint is the project-wide number of Machines that predate
	// trusted footprint capture.
	// +kubebuilder:validation:Minimum=0
	LegacyMachinesWithoutFootprint int32 `json:"legacyMachinesWithoutFootprint"`

	// MachinesWithInvalidFootprint is the project-wide number of Machines whose footprint
	// cannot be accounted.
	// +kubebuilder:validation:Minimum=0
	MachinesWithInvalidFootprint int32 `json:"machinesWithInvalidFootprint"`

	// Ready is true when every configured Seed has provided a compatible, fresh attestation
	// for ObservedAccountingRevision and ObservedQuotaDigest and no blocker remains.
	Ready bool `json:"ready"`

	// Blockers contains actionable reasons why project-wide accounting is not ready.
	// +optional
	Blockers []AcceleratorAccountingBlocker `json:"blockers,omitempty"`
}

// Subject describes the entity to which the quota applies to.
type Subject struct {
	// Name of the quota subject.
	Name string `json:"name"`

	// +kubebuilder:validation:Enum=project
	// +kubebuilder:default=project

	// Kind of the quota subject. For now the only possible kind is project.
	Kind string `json:"kind"`
}

// AcceleratorQuota holds accelerator limits for one KKP infrastructure provider.
//
// +kubebuilder:validation:XValidation:rule="self.resources.size() > 0",message="accelerator quota resources must not be empty"
type AcceleratorQuota struct {
	// Provider is the KKP infrastructure provider identifier, not the accelerator vendor.
	// The alpha API supports only kubevirt.
	//
	// +kubebuilder:validation:Enum=kubevirt
	Provider string `json:"provider"`

	// Resources contains provider-native accelerator resource names and their limits.
	// KubeVirt resource names are exact deviceName values. A missing resource name is
	// unconstrained, zero denies that resource, and a positive whole number sets its limit.
	Resources corev1.ResourceList `json:"resources"`
}

// ResourceDetails holds compute, storage, and accelerator resource quantities.
type ResourceDetails struct {
	// CPU holds the quantity of CPU. For the format, please check k8s.io/apimachinery/pkg/api/resource.Quantity.
	CPU *resource.Quantity `json:"cpu,omitempty"`
	// Memory represents the quantity of RAM size. For the format, please check k8s.io/apimachinery/pkg/api/resource.Quantity.
	Memory *resource.Quantity `json:"memory,omitempty"`
	// Storage represents the disk size. For the format, please check k8s.io/apimachinery/pkg/api/resource.Quantity.
	Storage *resource.Quantity `json:"storage,omitempty"`
	// Accelerators holds provider-specific accelerator limits. An absent or empty list means
	// that no accelerator limits are configured. A missing provider or provider/resource pair
	// is unconstrained; this field is not an allowlist. The list is atomic so a future provider
	// scope can participate in entry identity without redefining provider as the sole map key.
	//
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=1
	Accelerators []AcceleratorQuota `json:"accelerators,omitempty"`
}

func (r ResourceDetails) IsEmpty() bool {
	return (r.CPU == nil || r.CPU.IsZero()) && (r.Memory == nil || r.Memory.IsZero()) && (r.Storage == nil || r.Storage.IsZero())
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ResourceQuotaList is a collection of resource quotas.
type ResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of the resource quotas.
	Items []ResourceQuota `json:"items"`
}

func NewResourceDetails(cpu, memory, storage resource.Quantity) *ResourceDetails {
	return &ResourceDetails{
		CPU:     &cpu,
		Memory:  &memory,
		Storage: &storage,
	}
}

// NewResourceDetailsWithAccelerators constructs ResourceDetails with accelerator quotas
// while isolating the result from subsequent mutations of the provided quantities and maps.
func NewResourceDetailsWithAccelerators(cpu, memory, storage resource.Quantity, accelerators ...AcceleratorQuota) *ResourceDetails {
	cpuCopy := cpu.DeepCopy()
	memoryCopy := memory.DeepCopy()
	storageCopy := storage.DeepCopy()

	result := &ResourceDetails{
		CPU:     &cpuCopy,
		Memory:  &memoryCopy,
		Storage: &storageCopy,
	}

	if len(accelerators) > 0 {
		result.Accelerators = make([]AcceleratorQuota, len(accelerators))
		for i := range accelerators {
			result.Accelerators[i] = AcceleratorQuota{
				Provider:  accelerators[i].Provider,
				Resources: accelerators[i].Resources.DeepCopy(),
			}
		}
	}

	return result
}

// AcceleratorQuotaDigestFor returns a SHA-256 digest of the canonical accelerator quota.
// Provider and resource ordering do not affect the result, semantically equal quantities
// have the same representation, and nil and empty quota lists are equivalent.
func AcceleratorQuotaDigestFor(accelerators []AcceleratorQuota) AcceleratorQuotaDigest {
	sum := sha256.Sum256(canonicalAcceleratorQuotaJSON(accelerators))
	return AcceleratorQuotaDigest("sha256:" + hex.EncodeToString(sum[:]))
}

// AcceleratorQuotasEqual reports whether two accelerator quota lists are semantically equal.
// Provider and resource ordering are ignored, semantically equal quantities compare equal,
// and nil and empty quota lists are equivalent.
func AcceleratorQuotasEqual(a, b []AcceleratorQuota) bool {
	return bytes.Equal(canonicalAcceleratorQuotaJSON(a), canonicalAcceleratorQuotaJSON(b))
}

// AddAcceleratorUsage adds usage to target and canonicalizes the resulting provider list.
// It does not retain maps or quantity internals owned by either target or usage. Provider
// entries are ordered by provider and quantities use a format-independent representation.
func AddAcceleratorUsage(target *ResourceDetails, usage []AcceleratorQuota) {
	if target == nil {
		return
	}

	byProvider := map[string]corev1.ResourceList{}
	add := func(accelerators []AcceleratorQuota) {
		for _, accelerator := range accelerators {
			for name, quantity := range accelerator.Resources {
				quantity = quantity.DeepCopy()
				resources := byProvider[accelerator.Provider]
				if current, exists := resources[name]; exists {
					current = current.DeepCopy()
					current.Add(quantity)
					quantity = current
				}
				if quantity.IsZero() {
					delete(resources, name)
					continue
				}
				if resources == nil {
					resources = corev1.ResourceList{}
					byProvider[accelerator.Provider] = resources
				}
				resources[name] = canonicalAcceleratorQuantity(quantity)
			}
		}
	}

	add(target.Accelerators)
	add(usage)
	for provider, resources := range byProvider {
		if len(resources) == 0 {
			delete(byProvider, provider)
		}
	}

	if len(byProvider) == 0 {
		target.Accelerators = nil
		return
	}

	providers := make([]string, 0, len(byProvider))
	for provider := range byProvider {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	target.Accelerators = make([]AcceleratorQuota, 0, len(providers))
	for _, provider := range providers {
		target.Accelerators = append(target.Accelerators, AcceleratorQuota{
			Provider:  provider,
			Resources: byProvider[provider],
		})
	}
}

type canonicalAcceleratorQuota struct {
	Provider  string                         `json:"provider"`
	Resources []canonicalAcceleratorResource `json:"resources"`
}

type canonicalAcceleratorResource struct {
	Name     string `json:"name"`
	Mantissa string `json:"mantissa"`
	Exponent int32  `json:"exponent"`
}

func canonicalAcceleratorQuotaJSON(accelerators []AcceleratorQuota) []byte {
	canonical := make([]canonicalAcceleratorQuota, 0, len(accelerators))
	for _, accelerator := range accelerators {
		resources := make([]canonicalAcceleratorResource, 0, len(accelerator.Resources))
		for name, quantity := range accelerator.Resources {
			quantity = quantity.DeepCopy()
			mantissa, exponent := quantity.AsCanonicalBytes(nil)
			resources = append(resources, canonicalAcceleratorResource{
				Name:     string(name),
				Mantissa: string(mantissa),
				Exponent: exponent,
			})
		}
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].Name < resources[j].Name
		})

		canonical = append(canonical, canonicalAcceleratorQuota{
			Provider:  accelerator.Provider,
			Resources: resources,
		})
	}

	// Sort the full entries rather than only provider names so even invalid input
	// containing duplicate providers has an order-independent canonical form.
	sort.Slice(canonical, func(i, j int) bool {
		if canonical[i].Provider != canonical[j].Provider {
			return canonical[i].Provider < canonical[j].Provider
		}

		iResources, _ := json.Marshal(canonical[i].Resources)
		jResources, _ := json.Marshal(canonical[j].Resources)
		return bytes.Compare(iResources, jResources) < 0
	})

	encoded, _ := json.Marshal(canonical)
	return encoded
}

func canonicalAcceleratorQuantity(quantity resource.Quantity) resource.Quantity {
	quantity = quantity.DeepCopy()
	return resource.NewDecimalQuantity(*quantity.AsDec(), resource.DecimalSI).DeepCopy()
}
