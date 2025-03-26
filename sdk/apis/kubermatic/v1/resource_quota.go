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

// ResourceDetails holds the CPU, Memory and Storage quantities.
type ResourceDetails struct {
	// CPU holds the quantity of CPU. For the format, please check k8s.io/apimachinery/pkg/api/resource.Quantity.
	CPU *resource.Quantity `json:"cpu,omitempty"`
	// Memory represents the quantity of RAM size. For the format, please check k8s.io/apimachinery/pkg/api/resource.Quantity.
	Memory *resource.Quantity `json:"memory,omitempty"`
	// Storage represents the disk size. For the format, please check k8s.io/apimachinery/pkg/api/resource.Quantity.
	Storage *resource.Quantity `json:"storage,omitempty"`
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
