/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"
	"net"
	"strings"

	netutils "k8s.io/utils/net"
)

// AllExposeStrategies is a set containing all the ExposeStrategy.
var AllExposeStrategies = NewExposeStrategiesSet(ExposeStrategyNodePort, ExposeStrategyLoadBalancer, ExposeStrategyTunneling)

// ExposeStrategyFromString returns the expose strategy which String
// representation corresponds to the input string, and a bool saying whether a
// match was found or not.
func ExposeStrategyFromString(s string) (ExposeStrategy, bool) {
	es := ExposeStrategy(s)
	return es, AllExposeStrategies.Has(es)
}

// String returns the string representation of the ExposeStrategy.
func (e ExposeStrategy) String() string {
	return string(e)
}

// ExposeStrategiesSet is a set of ExposeStrategies.
type ExposeStrategiesSet map[ExposeStrategy]struct{}

// NewExposeStrategiesSet creates a ExposeStrategiesSet from a list of values.
func NewExposeStrategiesSet(items ...ExposeStrategy) ExposeStrategiesSet {
	es := ExposeStrategiesSet{}
	for _, item := range items {
		es[item] = struct{}{}
	}
	return es
}

// Has returns true if and only if item is contained in the set.
func (e ExposeStrategiesSet) Has(item ExposeStrategy) bool {
	_, contained := e[item]
	return contained
}

// Has returns true if and only if item is contained in the set.
func (e ExposeStrategiesSet) String() string {
	es := make([]string, 0, len(e))
	for k := range e {
		es = append(es, string(k))
	}
	// can be easily optimized in terms of allocations by using a bytes buffer
	// with some more verbosity, but this is not supposed to be called in a
	// perf critical path.
	return fmt.Sprintf("[%s]", strings.Join(es, ", "))
}

func (e ExposeStrategiesSet) Items() []string {
	var items []string
	for s := range e {
		items = append(items, string(s))
	}
	return items
}

// IsIPv4Only returns true if the cluster networking is IPv4-only.
func (c *Cluster) IsIPv4Only() bool {
	return len(c.Spec.ClusterNetwork.Pods.CIDRBlocks) == 1 && netutils.IsIPv4CIDRString(c.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
}

// IsIPv6Only returns true if the cluster networking is IPv6-only.
func (c *Cluster) IsIPv6Only() bool {
	return len(c.Spec.ClusterNetwork.Pods.CIDRBlocks) == 1 && netutils.IsIPv6CIDRString(c.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
}

// IsDualStack returns true if the cluster networking is dual-stack (IPv4 + IPv6).
func (c *Cluster) IsDualStack() bool {
	res, err := netutils.IsDualStackCIDRStrings(c.Spec.ClusterNetwork.Pods.CIDRBlocks)
	if err != nil {
		return false
	}
	return res
}

// Validate validates the network ranges. Returns nil if valid, error otherwise.
func (r *NetworkRanges) Validate() error {
	if r == nil {
		return nil
	}
	for _, cidr := range r.CIDRBlocks {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("unable to parse CIDR %q: %w", cidr, err)
		}
	}
	return nil
}

// GetIPv4CIDR returns the first found IPv4 CIDR in the network ranges, or an empty string if no IPv4 CIDR is found.
func (r *NetworkRanges) GetIPv4CIDR() string {
	for _, cidr := range r.CIDRBlocks {
		if netutils.IsIPv4CIDRString(cidr) {
			return cidr
		}
	}
	return ""
}

// GetIPv4CIDRs returns all IPv4 CIDRs in the network ranges, or an empty string if no IPv4 CIDR is found.
func (r *NetworkRanges) GetIPv4CIDRs() (res []string) {
	for _, cidr := range r.CIDRBlocks {
		if netutils.IsIPv4CIDRString(cidr) {
			res = append(res, cidr)
		}
	}
	return
}

// HasIPv4CIDR returns true if the network ranges contain any IPv4 CIDR, false otherwise.
func (r *NetworkRanges) HasIPv4CIDR() bool {
	return r.GetIPv4CIDR() != ""
}

// GetIPv6CIDR returns the first found IPv6 CIDR in the network ranges, or an empty string if no IPv6 CIDR is found.
func (r *NetworkRanges) GetIPv6CIDR() string {
	for _, cidr := range r.CIDRBlocks {
		if netutils.IsIPv6CIDRString(cidr) {
			return cidr
		}
	}
	return ""
}

// GetIPv6CIDRs returns all IPv6 CIDRs in the network ranges, or an empty string if no IPv6 CIDR is found.
func (r *NetworkRanges) GetIPv6CIDRs() (res []string) {
	for _, cidr := range r.CIDRBlocks {
		if netutils.IsIPv6CIDRString(cidr) {
			res = append(res, cidr)
		}
	}
	return
}

// HasIPv6CIDR returns true if the network ranges contain any IPv6 CIDR, false otherwise.
func (r *NetworkRanges) HasIPv6CIDR() bool {
	return r.GetIPv6CIDR() != ""
}
