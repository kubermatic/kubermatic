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

package ipam

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

func getIPsFromAddressRanges(addressRanges []string) ([]string, error) {
	ips := []string{}

	for _, addressRange := range addressRanges {
		ipRange := strings.SplitN(addressRange, "-", 2)
		if len(ipRange) != 2 && len(ipRange) != 1 {
			return nil, errors.New("wrong ip range format")
		}
		firstIP := net.ParseIP(ipRange[0])
		if firstIP == nil {
			return nil, errors.New("wrong ip format")
		}
		if len(ipRange) == 2 {
			lastIP := net.ParseIP(ipRange[1])
			if lastIP == nil {
				return nil, errors.New("wrong ip format")
			}
			for ip := firstIP; !ip.Equal(lastIP); ip = incIP(ip) {
				ips = append(ips, ip.String())
			}
			ips = append(ips, lastIP.String())
		} else {
			ips = append(ips, firstIP.String())
		}
	}

	return ips, nil
}

func checkRangeAllocation(ips []string, poolCIDR string, allocationRange int) error {
	if allocationRange != len(ips) {
		return errIncompatiblePool
	}

	_, poolSubnet, err := net.ParseCIDR(poolCIDR)
	if err != nil {
		return err
	}

	for _, ip := range ips {
		if !poolSubnet.Contains(net.ParseIP(ip)) {
			return errIncompatiblePool
		}
	}

	return nil
}

func calculateRangeFreeIPsFromDatacenterPool(poolCIDR string, dcIPAMPoolUsageMap sets.Set[string]) ([]string, error) {
	rangeFreeIPs := []string{}

	ip, ipNet, err := net.ParseCIDR(poolCIDR)
	if err != nil {
		return nil, err
	}
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); ip = incIP(ip) {
		if dcIPAMPoolUsageMap.Has(ip.String()) {
			continue
		}
		rangeFreeIPs = append(rangeFreeIPs, ip.String())
	}

	return rangeFreeIPs, nil
}

func findFirstFreeRangesOfPool(poolName, poolCIDR string, allocationRange int, dcIPAMPoolUsageMap sets.Set[string]) ([]string, error) {
	addressRanges := []string{}

	rangeFreeIPs, err := calculateRangeFreeIPsFromDatacenterPool(poolCIDR, dcIPAMPoolUsageMap)
	if err != nil {
		return nil, err
	}

	if allocationRange > len(rangeFreeIPs) {
		return nil, fmt.Errorf("there is no enough free IPs available for IPAM pool \"%s\"", poolName)
	}

	rangeFreeIPsIterator := 0
	firstAddressRangeIP := rangeFreeIPs[rangeFreeIPsIterator]
	for j := 0; j < allocationRange; j++ {
		ipToAllocate := rangeFreeIPs[rangeFreeIPsIterator]
		dcIPAMPoolUsageMap.Insert(ipToAllocate)
		rangeFreeIPsIterator++
		// if no next ip to allocate or next ip is not the next one, close a new address range
		if j+1 == allocationRange || !isTheNextIP(rangeFreeIPs[rangeFreeIPsIterator], ipToAllocate) {
			addressRange := fmt.Sprintf("%s-%s", firstAddressRangeIP, ipToAllocate)
			addressRanges = append(addressRanges, addressRange)
			if j+1 < allocationRange {
				firstAddressRangeIP = rangeFreeIPs[rangeFreeIPsIterator]
			}
		}
	}

	return addressRanges, nil
}
