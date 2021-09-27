//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Loodse GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package provider

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
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
				return nil, fmt.Errorf("datacenter %q used by node datacenter %q does not exist", datacenterSpec.Seed, dcName)
			}
			if !dm[datacenterSpec.Seed].IsSeed {
				return nil, fmt.Errorf("datacenter %q referenced by node datacenter %q as its seed, but %q is not configured to be a seed",
					datacenterSpec.Seed, dcName, datacenterSpec.Seed)
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
