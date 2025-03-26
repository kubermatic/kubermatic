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

package validation

import (
	"net"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func getSliceAdditions(previousSlice, currentSlice []string) []string {
	var additions []string

	previousMembersMap := map[string]struct{}{}
	for _, v := range previousSlice {
		previousMembersMap[v] = struct{}{}
	}

	for _, v := range currentSlice {
		if _, existedBefore := previousMembersMap[v]; !existedBefore {
			additions = append(additions, v)
		}
	}

	return additions
}

func subnetCIDRSliceToStringSlice(s []kubermaticv1.SubnetCIDR) []string {
	convertedSlice := make([]string, len(s))

	for i, value := range s {
		convertedSlice[i] = string(value)
	}

	return convertedSlice
}

func addressRangesConflict(firstAddressRanges []string, secondAddressRanges []string) bool {
	firstAddressRangesIPs := map[string]struct{}{}

	// iterate first address ranges and save ips in a map
	for _, addressRange := range firstAddressRanges {
		ipRange := strings.SplitN(addressRange, "-", 2)
		firstIP := net.ParseIP(ipRange[0])
		if len(ipRange) == 1 {
			firstAddressRangesIPs[firstIP.String()] = struct{}{}
			continue
		}
		lastIP := net.ParseIP(ipRange[1])
		for ip := firstIP; !ip.Equal(lastIP); ip = incIP(ip) {
			firstAddressRangesIPs[ip.String()] = struct{}{}
		}
		firstAddressRangesIPs[lastIP.String()] = struct{}{}
	}

	// iterate second address ranges to search for conflicts
	for _, addressRange := range secondAddressRanges {
		ipRange := strings.SplitN(addressRange, "-", 2)
		firstIP := net.ParseIP(ipRange[0])
		if len(ipRange) == 1 {
			if _, isConflict := firstAddressRangesIPs[firstIP.String()]; isConflict {
				return true
			}
			continue
		}
		lastIP := net.ParseIP(ipRange[1])
		for ip := firstIP; !ip.Equal(lastIP); ip = incIP(ip) {
			if _, isConflict := firstAddressRangesIPs[ip.String()]; isConflict {
				return true
			}
		}
		if _, isConflict := firstAddressRangesIPs[lastIP.String()]; isConflict {
			return true
		}
	}

	return false
}

func incIP(ip net.IP) net.IP {
	ip = checkIPv4(ip)
	incIP := make([]byte, len(ip))
	copy(incIP, ip)
	for j := len(incIP) - 1; j >= 0; j-- {
		incIP[j]++
		if incIP[j] > 0 {
			break
		}
	}
	return incIP
}

func checkIPv4(ip net.IP) net.IP {
	// Go for some reason allocs IPv6len for IPv4 so we have to correct it
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}
