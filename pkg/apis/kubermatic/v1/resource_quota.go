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

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// ResourceQuota specifies the amount of cluster resources a project can use.
type ResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceQuotaSpec   `json:"spec,omitempty"`
	Status ResourceQuotaStatus `json:"status,omitempty"`
}

// ResourceQuotaSpec describes the desired state of a resource quota.
type ResourceQuotaSpec struct {
	// QuotaSubject specifies to which entity the quota applies to.
	QuotaSubject QuotaSubject `json:"quotaSubject"`
	// Quota specifies the current maximum allowed usage of resources.
	Quota ResourceDetails `json:"quota"`
}

// ResourceQuotaStatus describes the current state of a resource quota.
type ResourceQuotaStatus struct {
	// ResourceConsumption is holds the current usage of resources for all seeds.
	ResourceConsumption ResourceDetails `json:"globalUsage,omitempty"`
	// LocalConsumption is holds the current usage of resources for the local seed.
	LocalConsumption ResourceDetails `json:"localUsage,omitempty"`
}

// QuotaSubject describes the entity to which the quota applies to.
type QuotaSubject struct {
	// Name of the quota subject.
	Name string `json:"name"`

	// +kubebuilder:validation:Enum=project
	// +kubebuilder:default=project

	// Type of the quota subject. For now the only possible type is project
	Type string `json:"type"`
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
