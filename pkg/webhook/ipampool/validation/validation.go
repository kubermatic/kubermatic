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
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Resource Quota CRD.
type validator struct {
	seedGetter       provider.SeedGetter
	seedClientGetter provider.SeedClientGetter
}

// NewValidator returns a new Resource Quota validator.
func NewValidator(seedGetter provider.SeedGetter, seedClientGetter provider.SeedClientGetter) *validator {
	return &validator{
		seedGetter:       seedGetter,
		seedClientGetter: seedClientGetter,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	if err := v.validate(ctx, newObj); err != nil {
		return nil, err
	}

	newIPAMPool := newObj.(*kubermaticv1.IPAMPool)
	oldIPAMPool := oldObj.(*kubermaticv1.IPAMPool)

	// loop old IPAMPool datacenters
	for dc, dcOldConfig := range oldIPAMPool.Spec.Datacenters {
		dcNewConfig, dcExistsInNewPool := newIPAMPool.Spec.Datacenters[dc]
		if !dcExistsInNewPool {
			// we allow deletion of a specific datacenter from the IPAM Pool
			continue
		}

		if dcOldConfig.PoolCIDR != dcNewConfig.PoolCIDR {
			return nil, errors.New("it's not allowed to update the pool CIDR for a datacenter")
		}

		if dcOldConfig.Type != dcNewConfig.Type {
			return nil, errors.New("it's not allowed to update the allocation type for a datacenter")
		}

		var addedExclusions []string

		switch dcOldConfig.Type {
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			addedExclusions = getSliceAdditions(dcOldConfig.ExcludeRanges, dcNewConfig.ExcludeRanges)
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			if dcOldConfig.AllocationPrefix != dcNewConfig.AllocationPrefix {
				return nil, errors.New("it's not allowed to update the allocation prefix for a datacenter")
			}
			addedExclusions = getSliceAdditions(
				subnetCIDRSliceToStringSlice(dcOldConfig.ExcludePrefixes),
				subnetCIDRSliceToStringSlice(dcNewConfig.ExcludePrefixes),
			)
		}

		if err := v.checkExclusionsNotAllocated(ctx, addedExclusions, oldIPAMPool.Name, dc, dcOldConfig.Type); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (v *validator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	// NOP we allow delete operation
	return nil, nil
}

func (v *validator) validate(ctx context.Context, obj runtime.Object) error {
	ipamPool, ok := obj.(*kubermaticv1.IPAMPool)
	if !ok {
		return errors.New("object is not a IPAMPool")
	}

	for _, dcConfig := range ipamPool.Spec.Datacenters {
		_, poolSubnet, err := net.ParseCIDR(string(dcConfig.PoolCIDR))
		if err != nil {
			return err
		}
		poolPrefix, bits := poolSubnet.Mask.Size()

		switch dcConfig.Type {
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			if dcConfig.AllocationRange <= 0 {
				return errors.New("allocation range should be greater than zero")
			}

			numberOfPoolSubnetIPsFloat64 := math.Pow(2, float64(bits-poolPrefix))
			numberOfPoolSubnetIPs := int(numberOfPoolSubnetIPsFloat64)
			if float64(numberOfPoolSubnetIPs) != numberOfPoolSubnetIPsFloat64 {
				return errors.New("the pool is too big to be processed")
			}

			if bits-poolPrefix > 12 {
				return errors.New("pool prefix is too low for range allocation type")
			}

			if dcConfig.AllocationRange > numberOfPoolSubnetIPs {
				return errors.New("allocation range cannot be greater than the pool subnet possible number of IP addresses")
			}

			for _, rangeToExclude := range dcConfig.ExcludeRanges {
				if err := validateRange(rangeToExclude); err != nil {
					return err
				}
			}
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			if dcConfig.AllocationPrefix < poolPrefix {
				return errors.New("allocation prefix cannot be smaller than the pool subnet mask size")
			}
			if dcConfig.AllocationPrefix > bits {
				return errors.New("invalid allocation prefix for IP version")
			}

			for _, subnetCIDRToExclude := range dcConfig.ExcludePrefixes {
				_, subnet, err := net.ParseCIDR(string(subnetCIDRToExclude))
				if err != nil {
					return fmt.Errorf("invalid CIDR for subnet to exclude: %w", err)
				}
				subnetPrefix, _ := subnet.Mask.Size()
				if dcConfig.AllocationPrefix != subnetPrefix {
					return fmt.Errorf("invalid length for subnet to exclude \"%s\": must be the same as the pool allocation prefix (%d)", subnetCIDRToExclude, subnetPrefix)
				}
			}
		}
	}

	return nil
}

func validateRange(r string) error {
	splitRange := strings.Split(r, "-")
	if len(splitRange) != 1 && len(splitRange) != 2 {
		return fmt.Errorf("invalid format for range: \"%s\" (format should be \"{first_ip}-{last_ip}\" or single \"{ip}\")", r)
	}
	var firstIP, lastIP net.IP
	firstIP = net.ParseIP(splitRange[0])
	if firstIP == nil {
		return fmt.Errorf("invalid IP format for \"%s\" in range \"%s\"", splitRange[0], r)
	}
	if len(splitRange) == 2 {
		lastIP = net.ParseIP(splitRange[1])
		if lastIP == nil {
			return fmt.Errorf("invalid IP format for \"%s\" in range \"%s\"", splitRange[1], r)
		}
		if (firstIP.To4() == nil && lastIP.To4() != nil) || (firstIP.To4() != nil && lastIP.To4() == nil) {
			return fmt.Errorf("different IP versions for range \"%s\"", r)
		}
		if bytes.Compare(lastIP, firstIP) < 0 {
			return fmt.Errorf("invalid range order for \"%s\"", r)
		}
	}

	return nil
}

func (v *validator) getSeedClient(ctx context.Context) (ctrlruntimeclient.Client, error) {
	seed, err := v.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get current seed: %w", err)
	}
	if seed == nil {
		return nil, errors.New("webhook not configured for a seed cluster")
	}

	client, err := v.seedClientGetter(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed client: %w", err)
	}

	return client, nil
}

func (v *validator) checkExclusionsNotAllocated(ctx context.Context, exclusions []string, ipamPoolName string, dc string, allocationType kubermaticv1.IPAMPoolAllocationType) error {
	if len(exclusions) == 0 {
		return nil
	}

	seedClient, err := v.getSeedClient(ctx)
	if err != nil {
		return err
	}

	// List all IPAM allocations
	ipamAllocationList := &kubermaticv1.IPAMAllocationList{}
	err = seedClient.List(ctx, ipamAllocationList)
	if err != nil {
		return fmt.Errorf("failed to list IPAM allocations: %w", err)
	}

	// Iterate current IPAM allocations to check if there is an allocation for the exclusion
	for _, ipamAllocation := range ipamAllocationList.Items {
		if ipamAllocation.Name != ipamPoolName || ipamAllocation.Spec.DC != dc {
			continue
		}

		errExclusionConflict := fmt.Errorf("failed to add exclusion: there is an conflicted allocation in IPAM pool \"%s\" and datacenter \"%s\"", ipamPoolName, dc)

		switch allocationType {
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			if addressRangesConflict(ipamAllocation.Spec.Addresses, exclusions) {
				return errExclusionConflict
			}
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			for _, exclusion := range exclusions {
				excludePrefixIP, _, err := net.ParseCIDR(exclusion)
				if err != nil {
					return err
				}
				_, allocatedSubnet, err := net.ParseCIDR(string(ipamAllocation.Spec.CIDR))
				if err != nil {
					return err
				}
				if string(ipamAllocation.Spec.CIDR) == exclusion || allocatedSubnet.Contains(excludePrefixIP) {
					return errExclusionConflict
				}
			}
		}
	}

	return nil
}
