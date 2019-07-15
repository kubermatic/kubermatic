package provider

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

// We can not convert a single DatacenterMeta as the SeedDatacenter contains its NodeDatacenter
// which is also represented as a DatacenterMeta, hence we have to do it all at once
// TODO: Get rid of this once we don't support datacenters.yaml anymore
func DatacenterMetasToSeedDatacenterSpecs(dm map[string]DatacenterMeta) (map[string]*kubermaticv1.SeedDatacenter, error) {

	seedDatacenters := map[string]*kubermaticv1.SeedDatacenter{}

	for dcName, datacenterSpec := range dm {
		if datacenterSpec.IsSeed && datacenterSpec.Seed != "" {
			return nil, fmt.Errorf("datacenter %q is configured as seed but has a seed configured(%q) which is only allowed for datacenters that are not a seed", dcName, datacenterSpec.Seed)
		}
		if !datacenterSpec.IsSeed && datacenterSpec.Seed == "" {
			return nil, fmt.Errorf("datacenter %q is not configured as seed but does not have a corresponding seed configured. Configuring a seed datacenter is required for all node datacenters", dcName)
		}

		if datacenterSpec.IsSeed {
			// Keep existing map entries, because its possible that a NodeDC that uses this SeedDC
			// came before the SeedDC in the loop
			if seedDatacenters[dcName] == nil {
				seedDatacenters[dcName] = &kubermaticv1.SeedDatacenter{
					Spec: kubermaticv1.SeedDatacenterSpec{
						NodeLocations: map[string]kubermaticv1.NodeLocation{},
					},
				}
			}

			seedDatacenters[dcName].Name = dcName
			seedDatacenters[dcName].Spec.Country = datacenterSpec.Country
			seedDatacenters[dcName].Spec.Location = datacenterSpec.Location
			// TODO: What to do about the kubeconfig?
			seedDatacenters[dcName].Spec.Kubeconfig = corev1.ObjectReference{}
			seedDatacenters[dcName].Spec.SeedDNSOverwrite = datacenterSpec.SeedDNSOverwrite

		} else {
			if _, exists := dm[datacenterSpec.Seed]; !exists {
				return nil, fmt.Errorf("seedDatacenter %q used by nodeDatacenter %q does not exist", datacenterSpec.Seed, dcName)
			}
			if !dm[datacenterSpec.Seed].IsSeed {
				return nil, fmt.Errorf("datacenter %q referenced by nodeDatacenter %q as its seed is not configured to be a seed",
					dcName, datacenterSpec.Seed)

			}
			// Create entry for SeedDC if not already present
			if seedDatacenters[datacenterSpec.Seed] == nil {
				seedDatacenters[datacenterSpec.Seed] = &kubermaticv1.SeedDatacenter{
					Spec: kubermaticv1.SeedDatacenterSpec{
						NodeLocations: map[string]kubermaticv1.NodeLocation{},
					},
				}

			}
			seedDatacenters[datacenterSpec.Seed].Spec.NodeLocations[dcName] = kubermaticv1.NodeLocation{
				Country:  datacenterSpec.Country,
				Location: datacenterSpec.Location,
				Node:     datacenterSpec.Node,
				Spec:     datacenterSpec.Spec,
			}

		}
	}

	return seedDatacenters, nil
}

// Needed because the cloud providers are initialized once during startup and get all
// DCs.
// We need to change the cloud providers to by dynamically initialized when needed instead
// once we support datacenters as CRDs.
// TODO: Find a way to lift the current requirement of unique nodeDatacenter names. It is needed
// only because we put the nodeDatacenter name on the cluster but not the seed
func NodeLocationFromSeedMap(seeds map[string]*kubermaticv1.SeedDatacenter, nodeLocationName string) (*kubermaticv1.NodeLocation, error) {

	var results []kubermaticv1.NodeLocation
	for _, seed := range seeds {
		nodeLocation, exists := seed.Spec.NodeLocations[nodeLocationName]
		if !exists {
			continue
		}

		results = append(results, nodeLocation)
	}

	if n := len(results); n != 1 {
		return nil, fmt.Errorf("expected to find exactly one datacenter with name %q, got %d", nodeLocationName, n)
	}

	return &results[0], nil
}
