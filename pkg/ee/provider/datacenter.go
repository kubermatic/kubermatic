// +build ee

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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	providertypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var (
	allOperatingSystems = sets.NewString()
)

func init() {
	// build a quicker, convenient lookup mechanism
	for _, os := range providertypes.AllOperatingSystems {
		allOperatingSystems.Insert(string(os))
	}
}

func validateImageList(images kubermaticv1.ImageList) error {
	for s := range images {
		if !allOperatingSystems.Has(string(s)) {
			return fmt.Errorf("invalid operating system defined '%s'. Possible values: %s", s, strings.Join(allOperatingSystems.List(), ","))
		}
	}

	return nil
}

// ValidateSeed takes a seed and returns an error if the seed's spec is invalid.
func ValidateSeed(seed *kubermaticv1.Seed) error {
	for name, dc := range seed.Spec.Datacenters {
		if dc.Spec.VSphere != nil {
			if err := validateImageList(dc.Spec.VSphere.Templates); err != nil {
				return fmt.Errorf("invalid datacenter defined '%s': %v", name, err)
			}
		}
		if dc.Spec.Openstack != nil {
			if err := validateImageList(dc.Spec.Openstack.Images); err != nil {
				return fmt.Errorf("invalid datacenter defined '%s': %v", name, err)
			}
		}
	}

	// invalid DNS overwrites can happen when a seed was freshly converted from
	// the datacenters.yaml and has not yet been validated
	if seed.Spec.SeedDNSOverwrite != "" {
		if errs := validation.IsDNS1123Subdomain(seed.Spec.SeedDNSOverwrite); errs != nil {
			return fmt.Errorf("DNS overwrite %q is not a valid DNS name: %v", seed.Spec.SeedDNSOverwrite, errs)
		}
	} else {
		if errs := validation.IsDNS1123Subdomain(seed.Name); errs != nil {
			return fmt.Errorf("seed name %q is not a valid DNS name: %v", seed.Name, errs)
		}
	}

	return nil
}

// DatacenterMeta describes a Kubermatic datacenter.
type DatacenterMeta struct {
	Location         string                      `json:"location"`
	Seed             string                      `json:"seed"`
	Country          string                      `json:"country"`
	Spec             kubermaticv1.DatacenterSpec `json:"spec"`
	IsSeed           bool                        `json:"is_seed"`
	SeedDNSOverwrite string                      `json:"seed_dns_overwrite,omitempty"`
	Node             *kubermaticv1.NodeSettings  `json:"node,omitempty"`
}

// datacentersMeta describes a number of Kubermatic datacenters.
type datacentersMeta struct {
	Datacenters map[string]DatacenterMeta `json:"datacenters"`
}

// LoadSeeds loads all Datacenters from the given path.
func LoadSeeds(path string) (map[string]*kubermaticv1.Seed, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read datacenter.yaml: %v", err)
	}

	dcMetas := datacentersMeta{}
	if err := yaml.UnmarshalStrict(bytes, &dcMetas); err != nil {
		return nil, fmt.Errorf("failed to parse datacenters.yaml: %v", err)
	}

	seeds, err := DatacenterMetasToSeeds(dcMetas.Datacenters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert datacenters.yaml: %v", err)
	}

	for seedName, seed := range seeds {
		if err := ValidateSeed(seed); err != nil {
			return nil, fmt.Errorf("failed to validate datacenters.yaml: seed %q is invalid: %v", seedName, err)
		}
	}

	return seeds, nil
}

// LoadSeed loads an existing datacenters.yaml from disk and returns the given
// datacenter as a seed or an error if the datacenter does not exist.
func LoadSeed(path, datacenterName string) (*kubermaticv1.Seed, error) {
	seeds, err := LoadSeeds(path)
	if err != nil {
		return nil, err
	}

	datacenter, exists := seeds[datacenterName]
	if !exists {
		return nil, fmt.Errorf("datacenter %q is not in datacenters.yaml", datacenterName)
	}

	return datacenter, nil
}

type EESeedGetter func() (*kubermaticv1.Seed, error)

func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Reader, dcFile string, seedName string) (EESeedGetter, error) {
	if dcFile == "" {
		return nil, errors.New("--datacenters is required")
	}
	// Make sure we fail early, an error here is not recoverable
	seed, err := LoadSeed(dcFile, seedName)
	if err != nil {
		return nil, err
	}
	return func() (*kubermaticv1.Seed, error) {
		return seed.DeepCopy(), nil
	}, nil
}

type EESeedsGetter func() (map[string]*kubermaticv1.Seed, error)

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile string, namespace string, dynamicDatacenters bool) (EESeedsGetter, error) {
	if dynamicDatacenters {
		// We only have a options func for raw *metav1.ListOpts as the rbac controller currently required that
		listOpts := &ctrlruntimeclient.ListOptions{
			Namespace: namespace,
		}

		return func() (map[string]*kubermaticv1.Seed, error) {
			seeds := &kubermaticv1.SeedList{}
			if err := client.List(ctx, seeds, listOpts); err != nil {
				return nil, fmt.Errorf("failed to list the seeds: %v", err)
			}
			seedMap := map[string]*kubermaticv1.Seed{}
			for idx, seed := range seeds.Items {
				seedMap[seed.Name] = &seeds.Items[idx]
			}
			return seedMap, nil
		}, nil
	}

	if dcFile == "" {
		return nil, errors.New("--datacenters is required")
	}
	// Make sure we fail early, an error here is not recoverable
	seedMap, err := LoadSeeds(dcFile)
	if err != nil {
		return nil, err
	}

	return func() (map[string]*kubermaticv1.Seed, error) {
		// copy it, just to be safe
		mapToReturn := map[string]*kubermaticv1.Seed{}
		for k, v := range seedMap {
			mapToReturn[k] = v.DeepCopy()
		}
		return mapToReturn, nil
	}, nil
}

type EESeedKubeconfigGetter = func(seed *kubermaticv1.Seed) (*rest.Config, error)

// Ensures that SeedKubeconfigGetter implements EESeedKubeconfigGetter
var _ EESeedKubeconfigGetter = SeedKubeconfigGetter

// SeedKubeconfigGetter implements provider.SeedKubeconfigGetter.
func SeedKubeconfigGetter(seed *kubermaticv1.Seed) (*rest.Config, error) {
	cfg, err := ctrlruntimeconfig.GetConfigWithContext(seed.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get restConfig for seed %q: %v", seed.Name, err)
	}
	return cfg, nil
}
