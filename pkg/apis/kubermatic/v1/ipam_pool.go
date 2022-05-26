/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	// IPAMPoolResourceName represents "Resource" defined in Kubernetes.
	IPAMPoolResourceName = "ipampool"

	// IPAMPoolKindName represents "Kind" defined in Kubernetes.
	IPAMPoolKindName = "IPAMPool"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// IPAMPool is the object representing IP Address Management (IPAM) configuration
// for KKP user cluster applications.
type IPAMPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IPAMPoolSpec `json:"spec,omitempty"`
}

// IPAMPoolSpec specifies the IP Address Management (IPAM) configuration
// for KKP user cluster applications.
type IPAMPoolSpec struct {
	// Type is the allocation type to be used.
	Type IPAMPoolAllocationType `json:"type"`
	// Datacenters contains a map of datacenters (DCs) for the allocation.
	Datacenters map[string]IPAMPoolDatacenterSettings `json:"datacenters"`
}

// IPAMPoolDatacenterSettings contains IPAM Pool configuration for a datacenter.
type IPAMPoolDatacenterSettings struct {
	// PoolCIDR is the pool CIDR to be used for the allocation
	PoolCIDR CIDR `json:"poolCIDR"`

	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=32

	// AllocationPrefix is the prefix for the allocation range
	AllocationPrefix uint8 `json:"allocationPrefix"`
}

// +kubebuilder:validation:Enum=subnet

// IPAMPoolAllocationType defines the type of allocation to be used.
// Possible values are `subnet`.
type IPAMPoolAllocationType string

func (t IPAMPoolAllocationType) String() string {
	return string(t)
}

const (
	// IPAMPoolAllocationType corresponds to subnet allocation type.
	IPAMPoolAllocationTypeSubnet IPAMPoolAllocationType = "subnet"
)
