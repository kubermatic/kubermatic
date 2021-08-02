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
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// DatacenterFromSeedMap is needed because the cloud providers are initialized once during startup and get all DCs.
// We need to change the cloud providers to by dynamically initialized when needed instead once we support datacenters
// as CRDs.
// TODO: Find a way to lift the current requirement of unique datacenter names. It is needed only because we put
// 	 the datacenter name in the cluster object but not the seed name.
func DatacenterFromSeedMap(userInfo *UserInfo, seedsGetter SeedsGetter, datacenterName string) (*kubermaticv1.Seed, *kubermaticv1.Datacenter, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	var datacenters []kubermaticv1.Datacenter
	for _, seed := range seeds {
		if dc, exists := seed.Spec.Datacenters[datacenterName]; exists {
			datacenters = append(datacenters, dc)
		}
	}

	if len(datacenters) == 0 {
		return nil, nil, errors.New(http.StatusNotFound, fmt.Sprintf("datacenter %q not found", datacenterName))
	}

	if count := len(datacenters); count > 1 {
		return nil, nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("expected to find exactly one datacenter with name %q, got %d", datacenterName, count))
	}

	matchingDatacenter := datacenters[0]

	if !userInfo.IsAdmin {
		emailDomain, err := getUserEmailDomain(userInfo)
		if err != nil {
			return nil, nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("invalid email domain: %v", err))
		}

		if !canAccessDatacenter(matchingDatacenter, emailDomain) {
			return nil, nil, errors.New(http.StatusUnauthorized, fmt.Sprintf("cannot access %s datacenter due to email domain requirements", datacenterName))
		}
	}

	// TODO: Get seed.
	return nil, &matchingDatacenter, nil
}

// canAccessDatacenter returns information if user with provided email domain can access given datacenter
// based on the required email domains that are set on it.
func canAccessDatacenter(dc kubermaticv1.Datacenter, emailDomain string) bool {
	// Return false if required email domain is set, but it is different from user email domain.
	if dc.Spec.RequiredEmailDomain != "" && !strings.EqualFold(emailDomain, dc.Spec.RequiredEmailDomain) {
		return false
	}

	// Return false if required email domains are set, but all of them are different from user email domain.
	if len(dc.Spec.RequiredEmailDomains) > 0 {
		isMatching := false
		for _, domain := range dc.Spec.RequiredEmailDomains {
			if domain != "" && strings.EqualFold(emailDomain, domain) {
				isMatching = true
				break
			}
		}
		if !isMatching {
			return false
		}
	}

	return true
}

func getUserEmailDomain(userInfo *UserInfo) (string, error) {
	split := strings.Split(userInfo.Email, "@")
	if len(split) != 2 {
		return "", fmt.Errorf("invalid email address")
	}
	return split[1], nil
}
