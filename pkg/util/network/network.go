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

package network

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/utils/net"
)

// IsIPv4OnlyCluster returns true if the cluster networking is IPv4-only.
func IsIPv4OnlyCluster(cluster *kubermaticv1.Cluster) bool {
	return len(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks) == 1 && net.IsIPv4CIDRString(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
}

// IsIPv6OnlyCluster returns true if the cluster networking is IPv6-only.
func IsIPv6OnlyCluster(cluster *kubermaticv1.Cluster) bool {
	return len(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks) == 1 && net.IsIPv6CIDRString(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
}

// IsDualStackCluster returns true if the cluster networking is dual-stack (IPv4 + IPv6).
func IsDualStackCluster(cluster *kubermaticv1.Cluster) bool {
	res, err := net.IsDualStackCIDRStrings(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks)
	if err != nil {
		return false
	}
	return res
}

// GetIPv4CIDR returns the first found IPv4 CIDR in the provided network ranges, or an empty string if no IPv4 CIDR is found.
func GetIPv4CIDR(nr kubermaticv1.NetworkRanges) string {
	for _, cidr := range nr.CIDRBlocks {
		if net.IsIPv4CIDRString(cidr) {
			return cidr
		}
	}
	return ""
}

// GetIPv6CIDR returns the first found IPv6 CIDR in the provided network ranges, or an empty string if no IPv6 CIDR is found.
func GetIPv6CIDR(nr kubermaticv1.NetworkRanges) string {
	for _, cidr := range nr.CIDRBlocks {
		if net.IsIPv6CIDRString(cidr) {
			return cidr
		}
	}
	return ""
}

// HasIPv4CIDR returns true if the provided network ranges contain any IPv4 CIDR, false otherwise.
func HasIPv4CIDR(nr kubermaticv1.NetworkRanges) bool {
	return GetIPv4CIDR(nr) != ""
}

// HasIPv6CIDR returns true if the provided network ranges contain any IPv6 CIDR, false otherwise.
func HasIPv6CIDR(nr kubermaticv1.NetworkRanges) bool {
	return GetIPv6CIDR(nr) != ""
}
