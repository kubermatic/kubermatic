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

package provider

import (
	"fmt"
	"net/http"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/email"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// DatacenterFromSeedMap returns datacenter from the seed:datacenter map.
//
// It is needed because the cloud providers are initialized once during startup and get all DCs. We need to change
// the cloud providers to by dynamically initialized when needed instead once we support datacenters as CRDs.
//
// TODO: Find a way to lift the current requirement of unique datacenter names. It is needed only because we put
// 	 the datacenter name in the cluster object but not the seed name.
func DatacenterFromSeedMap(userInfo *UserInfo, seedsGetter SeedsGetter, datacenterName string) (*kubermaticv1.Seed, *kubermaticv1.Datacenter, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	var matchingDatacenters []kubermaticv1.Datacenter
	var matchingSeeds []*kubermaticv1.Seed
	for _, seed := range seeds {
		if dc, exists := seed.Spec.Datacenters[datacenterName]; exists {
			matchingDatacenters = append(matchingDatacenters, dc)
			matchingSeeds = append(matchingSeeds, seed)
		}
	}

	if len(matchingDatacenters) == 0 {
		return nil, nil, errors.New(http.StatusNotFound, fmt.Sprintf("datacenter %q not found", datacenterName))
	}

	if count := len(matchingDatacenters); count > 1 {
		return nil, nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("expected to find exactly one datacenter with name %q, got %d", datacenterName, count))
	}

	matchingDatacenter := matchingDatacenters[0]
	matchingSeed := matchingSeeds[0]

	if !userInfo.IsAdmin {
		if !canAccessDatacenter(matchingDatacenter, userInfo.Email) {
			return nil, nil, errors.New(http.StatusForbidden, fmt.Sprintf("cannot access %s datacenter due to email requirements", datacenterName))
		}
	}

	return matchingSeed, &matchingDatacenter, nil
}

// canAccessDatacenter returns information if user with provided email can access given datacenter
// based on the required emails that are set on it.
func canAccessDatacenter(dc kubermaticv1.Datacenter, emailAddress string) bool {
	requirements := dc.Spec.RequiredEmailDomains
	if legacy := dc.Spec.RequiredEmailDomain; len(legacy) != 0 {
		requirements = append(requirements, legacy)
	}

	matches, _ := email.MatchesRequirements(emailAddress, requirements)

	return matches == true
}
