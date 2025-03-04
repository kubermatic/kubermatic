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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// IPAMAllocationResourceName represents "Resource" defined in Kubernetes.
	IPAMAllocationResourceName = "ipamallocation"

	// IPAMAllocationKindName represents "Kind" defined in Kubernetes.
	IPAMAllocationKindName = "IPAMAllocation"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// IPAMAllocation is the object representing an allocation from an IPAMPool
// made for a particular KKP user cluster.
type IPAMAllocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IPAMAllocationSpec `json:"spec,omitempty"`
}

// IPAMAllocationSpec specifies an allocation from an IPAMPool
// made for a particular KKP user cluster.
type IPAMAllocationSpec struct {
	// Type is the allocation type that is being used.
	Type IPAMPoolAllocationType `json:"type"`
	// DC is the datacenter of the allocation.
	DC string `json:"dc"`
	// CIDR is the CIDR that is being used for the allocation.
	// Set when "type=prefix".
	CIDR SubnetCIDR `json:"cidr,omitempty"`
	// Addresses are the IP address ranges that are being used for the allocation.
	// Set when "type=range".
	Addresses []string `json:"addresses,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

type IPAMAllocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []IPAMAllocation `json:"items"`
}
