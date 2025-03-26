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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

func checkPrefixAllocation(currentAllocatedCIDR, poolCIDR string, excludePrefixes []kubermaticv1.SubnetCIDR, allocationPrefix int) error {
	currentAllocatedSubnetIP, currentAllocatedSubnet, err := net.ParseCIDR(currentAllocatedCIDR)
	if err != nil {
		return err
	}

	currentAllocatedSubnetPrefix, _ := currentAllocatedSubnet.Mask.Size()
	_, poolSubnet, err := net.ParseCIDR(poolCIDR)
	if err != nil {
		return err
	}

	poolPrefix, poolBits := poolSubnet.Mask.Size()
	for _, v := range excludePrefixes {
		excludePrefixIP, _, err := net.ParseCIDR(string(v))
		if err != nil {
			return err
		}
		if currentAllocatedSubnet.Contains(excludePrefixIP) {
			return errIncompatiblePool
		}
	}
	if currentAllocatedSubnetPrefix < poolPrefix {
		return errIncompatiblePool
	}
	if currentAllocatedSubnetPrefix > poolBits {
		return errIncompatiblePool
	}

	if !poolSubnet.Contains(currentAllocatedSubnetIP) {
		return errIncompatiblePool
	}

	return nil
}

func findFirstFreeSubnetOfPool(poolName, poolCIDR, currentAllocatedCIDR string, subnetPrefix int, dcIPAMPoolUsageMap sets.Set[string]) (string, error) {
	poolIP, poolSubnet, err := net.ParseCIDR(poolCIDR)
	if err != nil {
		return currentAllocatedCIDR, err
	}

	poolPrefix, bits := poolSubnet.Mask.Size()
	if subnetPrefix < poolPrefix {
		return currentAllocatedCIDR, errors.New("invalid prefix for subnet")
	}
	if subnetPrefix > bits {
		return currentAllocatedCIDR, errors.New("invalid prefix for subnet")
	}

	_, possibleSubnet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", poolIP.Mask(poolSubnet.Mask), subnetPrefix))
	if err != nil {
		return currentAllocatedCIDR, err
	}

	if currentAllocatedCIDR != "" {
		_, currentAllocatedSubnet, err := net.ParseCIDR(currentAllocatedCIDR)
		if err != nil {
			return currentAllocatedCIDR, err
		}
		currentAllocatedPrefix, _ := currentAllocatedSubnet.Mask.Size()
		if subnetPrefix == currentAllocatedPrefix {
			return currentAllocatedCIDR, nil
		}
	}

	for ; poolSubnet.Contains(possibleSubnet.IP); possibleSubnet, _ = nextSubnet(possibleSubnet, subnetPrefix) {
		if !dcIPAMPoolUsageMap.Has(possibleSubnet.String()) {
			dcIPAMPoolUsageMap.Insert(possibleSubnet.String())
			return possibleSubnet.String(), nil
		}
	}

	return currentAllocatedCIDR, fmt.Errorf("there is no free subnet available for IPAM Pool \"%s\"", poolName)
}
