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
	// IPAMPoolResourceName represents "Resource" defined in Kubernetes.
	IPAMPoolResourceName = "ipampool"

	// IPAMPoolKindName represents "Kind" defined in Kubernetes.
	IPAMPoolKindName = "IPAMPool"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// IPAMPool is the object representing Multi-Cluster IP Address Management (IPAM)
// configuration for KKP user clusters.
type IPAMPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the Multi-Cluster IP Address Management (IPAM) configuration for KKP user clusters.
	Spec IPAMPoolSpec `json:"spec,omitempty"`
}

// IPAMPoolSpec specifies the  Multi-Cluster IP Address Management (IPAM)
// configuration for KKP user clusters.
type IPAMPoolSpec struct {
	// Datacenters contains a map of datacenters (DCs) for the allocation.
	Datacenters map[string]IPAMPoolDatacenterSettings `json:"datacenters"`
}

// IPAMPoolDatacenterSettings contains IPAM Pool configuration for a datacenter.
type IPAMPoolDatacenterSettings struct {
	// Type is the allocation type to be used.
	Type IPAMPoolAllocationType `json:"type"`

	// PoolCIDR is the pool CIDR to be used for the allocation.
	PoolCIDR SubnetCIDR `json:"poolCidr"`

	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=128
	// AllocationPrefix is the prefix for the allocation.
	// Used when "type=prefix".
	AllocationPrefix int `json:"allocationPrefix,omitempty"`

	// Optional: ExcludePrefixes is used to exclude particular subnets for the allocation.
	// NOTE: must be the same length as allocationPrefix.
	// Can be used when "type=prefix".
	ExcludePrefixes []SubnetCIDR `json:"excludePrefixes,omitempty"`

	// +kubebuilder:validation:Minimum:=1
	// AllocationRange is the range for the allocation.
	// Used when "type=range".
	AllocationRange int `json:"allocationRange,omitempty"`

	// Optional: ExcludeRanges is used to exclude particular IPs or IP ranges for the allocation.
	// Examples: "192.168.1.100-192.168.1.110", "192.168.1.255".
	// Can be used when "type=range".
	ExcludeRanges []string `json:"excludeRanges,omitempty"`
}

// +kubebuilder:validation:Pattern="((^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))/([0-9]|[1-2][0-9]|3[0-2])$)|(^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:))/([0-9]|[0-9][0-9]|1[0-1][0-9]|12[0-8])$))"
// SubnetCIDR is used to store IPv4/IPv6 CIDR.
type SubnetCIDR string

// +kubebuilder:validation:Enum=prefix;range
// IPAMPoolAllocationType defines the type of allocation to be used.
// Possible values are `prefix` and `range`.
type IPAMPoolAllocationType string

func (t IPAMPoolAllocationType) String() string {
	return string(t)
}

const (
	// IPAMPoolAllocationTypePrefix corresponds to prefix allocation type.
	IPAMPoolAllocationTypePrefix IPAMPoolAllocationType = "prefix"
	// IPAMPoolAllocationTypeRange corresponds to range allocation type.
	IPAMPoolAllocationTypeRange IPAMPoolAllocationType = "range"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// IPAMPoolList is the list of the object representing Multi-Cluster IP Address Management (IPAM)
// configuration for KKP user clusters.
type IPAMPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items holds the list of IPAM pool objects.
	Items []IPAMPool `json:"items"`
}
