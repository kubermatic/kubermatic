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

package openstack

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/pagination"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

const (
	subnetCIDR         = "192.168.1.0/24"
	subnetFirstAddress = "192.168.1.2"
	subnetLastAddress  = "192.168.1.254"

	defaultIPv6SubnetCIDR = "fd00::/64"

	resourceNamePrefix = "kubernetes-"
)

func getFlavors(authClient *gophercloud.ProviderClient, region string) ([]osflavors.Flavor, error) {
	computeClient, err := goopenstack.NewComputeV2(authClient, gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic, Region: region})
	if err != nil {
		// this is special case for services that span only one region.
		if isEndpointNotFoundErr(err) {
			computeClient, err = goopenstack.NewComputeV2(authClient, gophercloud.EndpointOpts{})
			if err != nil {
				return nil, fmt.Errorf("couldn't get identity endpoint: %w", err)
			}
		} else {
			return nil, fmt.Errorf("couldn't get identity endpoint: %w", err)
		}
	}

	var allFlavors []osflavors.Flavor
	pager := osflavors.ListDetail(computeClient, osflavors.ListOpts{})
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		flavors, err := osflavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		allFlavors = append(allFlavors, flavors...)
		return true, nil
	})

	if err != nil {
		return nil, err
	}
	return allFlavors, nil
}

func isNotFoundErr(err error) bool {
	var errNotFound gophercloud.ErrDefault404

	return errors.As(err, &errNotFound) || strings.Contains(err.Error(), "not found")
}

func isEndpointNotFoundErr(err error) bool {
	var endpointNotFoundErr *gophercloud.ErrEndpointNotFound
	// left side of the || to catch any error returned as pointer to struct (current case of gophercloud)
	// right side of the || to catch any error returned as struct (in case...)
	return errors.As(err, &endpointNotFoundErr) || errors.As(err, &gophercloud.ErrEndpointNotFound{})
}

// isMultipleSGs returns true if the input string contains more than one non-empty security group.
func isMultipleSGs(sg string) bool {
	return len(strings.Split(sg, ",")) > 1
}

func retryOnError(
	numTries int,
	operation func() error,
) error {
	steps := 0
	backoff := wait.Backoff{
		Steps:    numTries,
		Duration: 500 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}
	retriable := func(err error) bool {
		steps++

		return steps <= backoff.Steps
	}
	return retry.OnError(backoff, retriable, func() error {
		return operation()
	})
}
