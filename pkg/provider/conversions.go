package provider

import (
	"fmt"
	"net/http"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// Needed because the cloud providers are initialized once during startup and get all
// DCs.
// We need to change the cloud providers to by dynamically initialized when needed instead
// once we support datacenters as CRDs.
// TODO: Find a way to lift the current requirement of unique nodeDatacenter names. It is needed
// only because we put the nodeDatacenter name on the cluster but not the seed
func DatacenterFromSeedMap(userInfo *UserInfo, seedsGetter SeedsGetter, datacenterName string) (*kubermaticv1.Seed, *kubermaticv1.Datacenter, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	var foundDatacenters []kubermaticv1.Datacenter
	var foundSeeds []*kubermaticv1.Seed

iterateOverSeeds:
	for _, seed := range seeds {
		datacenter, exists := seed.Spec.Datacenters[datacenterName]
		if !exists {
			continue
		}

		requiredEmailDomain := datacenter.Spec.RequiredEmailDomain
		requiredEmailDomains := datacenter.Spec.RequiredEmailDomains

		if requiredEmailDomain == "" && len(requiredEmailDomains) == 0 {
			// find datacenter for "all" without RequiredEmailDomain(s) field
			foundSeeds = append(foundSeeds, seed)
			foundDatacenters = append(foundDatacenters, datacenter)
			continue iterateOverSeeds
		} else {
			// find datacenter for specific email domain
			split := strings.Split(userInfo.Email, "@")
			if len(split) != 2 {
				return nil, nil, fmt.Errorf("invalid email address")
			}
			userDomain := split[1]

			if requiredEmailDomain != "" && strings.EqualFold(userDomain, requiredEmailDomain) {
				foundSeeds = append(foundSeeds, seed)
				foundDatacenters = append(foundDatacenters, datacenter)
				continue iterateOverSeeds
			}

			for _, whitelistedDomain := range requiredEmailDomains {
				if whitelistedDomain != "" && strings.EqualFold(userDomain, whitelistedDomain) {
					foundSeeds = append(foundSeeds, seed)
					foundDatacenters = append(foundDatacenters, datacenter)
					continue iterateOverSeeds
				}
			}
		}
	}

	if len(foundDatacenters) == 0 {
		return nil, nil, errors.New(http.StatusNotFound, fmt.Sprintf("datacenter %q not found", datacenterName))
	}
	if n := len(foundDatacenters); n > 1 {
		return nil, nil, fmt.Errorf("expected to find exactly one datacenter with name %q, got %d", datacenterName, n)
	}

	return foundSeeds[0], &foundDatacenters[0], nil
}
