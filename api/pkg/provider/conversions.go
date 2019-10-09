package provider

import (
	"fmt"
	"net/http"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// We can not convert a single DatacenterMeta as the SeedDatacenter contains its NodeDatacenter
// which is also represented as a DatacenterMeta, hence we have to do it all at once
// TODO: Get rid of this once we don't support datacenters.yaml anymore
func DatacenterMetasToSeeds(dm map[string]DatacenterMeta) (map[string]*kubermaticv1.Seed, error) {

	seeds := map[string]*kubermaticv1.Seed{}

	for dcName, datacenterSpec := range dm {
		if datacenterSpec.IsSeed && datacenterSpec.Seed != "" {
			return nil, fmt.Errorf("datacenter %q is configured as seed but has a seed configured (%q) which is only allowed for datacenters that are not a seed", dcName, datacenterSpec.Seed)
		}
		if !datacenterSpec.IsSeed && datacenterSpec.Seed == "" {
			return nil, fmt.Errorf("datacenter %q is not configured as seed but does not have a corresponding seed configured. Configuring a seed datacenter is required for all node datacenters", dcName)
		}

		if datacenterSpec.IsSeed {
			// Keep existing map entries, because its possible that a NodeDC that uses this SeedDC
			// came before the SeedDC in the loop
			if seeds[dcName] == nil {
				seeds[dcName] = &kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{},
					},
				}
			}

			seeds[dcName].Name = dcName
			seeds[dcName].Spec.Country = datacenterSpec.Country
			seeds[dcName].Spec.Location = datacenterSpec.Location
			seeds[dcName].Spec.SeedDNSOverwrite = datacenterSpec.SeedDNSOverwrite

			// Kubeconfig object ref is injected during the automated migration.
		} else {
			if _, exists := dm[datacenterSpec.Seed]; !exists {
				return nil, fmt.Errorf("seedDatacenter %q used by node datacenter %q does not exist", datacenterSpec.Seed, dcName)
			}
			if !dm[datacenterSpec.Seed].IsSeed {
				return nil, fmt.Errorf("datacenter %q referenced by node datacenter %q as its seed is not configured to be a seed",
					datacenterSpec.Seed, dcName)

			}
			// Create entry for SeedDC if not already present
			if seeds[datacenterSpec.Seed] == nil {
				seeds[datacenterSpec.Seed] = &kubermaticv1.Seed{
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{},
					},
				}

			}
			seeds[datacenterSpec.Seed].Spec.Datacenters[dcName] = kubermaticv1.Datacenter{
				Country:  datacenterSpec.Country,
				Location: datacenterSpec.Location,
				Node:     datacenterSpec.Node,
				Spec:     datacenterSpec.Spec,
			}

		}
	}

	return seeds, nil
}

// Needed because the cloud providers are initialized once during startup and get all
// DCs.
// We need to change the cloud providers to by dynamically initialized when needed instead
// once we support datacenters as CRDs.
// TODO: Find a way to lift the current requirement of unique nodeDatacenter names. It is needed
// only because we put the nodeDatacenter name on the cluster but not the seed
func DatacenterFromSeedMap(seedsGetter SeedsGetter, datacenterName string) (*kubermaticv1.Seed, *kubermaticv1.Datacenter, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	var foundDatacenters []kubermaticv1.Datacenter
	var foundSeeds []*kubermaticv1.Seed
	for _, seed := range seeds {
		datacenter, exists := seed.Spec.Datacenters[datacenterName]
		if !exists {
			continue
		}

		foundSeeds = append(foundSeeds, seed)
		foundDatacenters = append(foundDatacenters, datacenter)
	}

	if n := len(foundDatacenters); n != 1 {
		return nil, nil, fmt.Errorf("expected to find exactly one datacenter with name %q, got %d", datacenterName, n)
	}

	return foundSeeds[0], &foundDatacenters[0], nil
}
