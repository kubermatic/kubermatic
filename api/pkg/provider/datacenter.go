package provider

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	// AllOperatingSystems defines all available operating systems
	AllOperatingSystems = sets.NewString(string(providerconfig.OperatingSystemCoreos), string(providerconfig.OperatingSystemCentOS), string(providerconfig.OperatingSystemUbuntu))
)

// SeedGetter is a function to retrieve Seeds
type SeedGetter = func() (*kubermaticv1.Seed, error)

// DatacenterMeta describes a Kubermatic datacenter.
type DatacenterMeta struct {
	Location         string                      `json:"location"`
	Seed             string                      `json:"seed"`
	Country          string                      `json:"country"`
	Spec             kubermaticv1.DatacenterSpec `json:"spec"`
	IsSeed           bool                        `json:"is_seed"`
	SeedDNSOverwrite *string                     `json:"seed_dns_overwrite,omitempty"`
	Node             kubermaticv1.NodeSettings   `json:"node,omitempty"`
}

// datacentersMeta describes a number of Kubermatic datacenters.
type datacentersMeta struct {
	Datacenters map[string]DatacenterMeta `json:"datacenters"`
}

// LoadDatacenters loads all Datacenters from the given path.
func LoadSeeds(path string) (map[string]*kubermaticv1.Seed, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	dcMetas := datacentersMeta{}
	if err := yaml.UnmarshalStrict(bytes, &dcMetas); err != nil {
		return nil, err
	}

	if err := validateDatacenters(dcMetas.Datacenters); err != nil {
		return nil, err
	}

	dcs, err := DatacenterMetasToSeeds(dcMetas.Datacenters)
	if err != nil {
		return nil, err
	}

	return dcs, nil
}

func LoadSeed(path, datacenterName string) (*kubermaticv1.Seed, error) {
	seeds, err := LoadSeeds(path)
	if err != nil {
		return nil, err
	}

	datacenter, exists := seeds[datacenterName]
	if !exists {
		return nil, fmt.Errorf("Datacenter %q is not in datacenters.yaml", datacenterName)
	}

	return datacenter, nil
}

func validateImageList(images kubermaticv1.ImageList) error {
	for s := range images {
		if !AllOperatingSystems.Has(string(s)) {
			return fmt.Errorf("invalid operating system defined '%s'. Possible values: %s", s, strings.Join(AllOperatingSystems.List(), ","))
		}
	}

	return nil
}

func validateDatacenters(datacenters map[string]DatacenterMeta) error {
	for name, dc := range datacenters {
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

	for datacenterName, datacenter := range datacenters {
		if !datacenter.IsSeed {
			continue
		}
		if datacenter.SeedDNSOverwrite != nil && *datacenter.SeedDNSOverwrite != "" {
			if errs := validation.IsDNS1123Subdomain(*datacenter.SeedDNSOverwrite); errs != nil {
				return fmt.Errorf("SeedDNS overwrite %q of datacenter %q is not a valid DNS name: %v",
					*datacenter.SeedDNSOverwrite, datacenterName, errs)
			}
			continue
		}
		if errs := validation.IsDNS1123Subdomain(datacenterName); errs != nil {
			return fmt.Errorf("Datacentername %q is not a valid DNS name: %v", datacenterName, errs)
		}
	}

	return nil
}

// SeedGetterFactory returns a SeedGetter. It has validation of all its arguments
func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile string, dynamicDatacenters bool) (SeedGetter, error) {
	if dcFile != "" && dynamicDatacenters {
		return nil, errors.New("--datacenters must be empty when --dynamic-datacenters is enabled")
	}

	if !dynamicDatacenters {
		if dcFile == "" {
			return nil, errors.New("--datacenters is required")
		}
		// Make sure we fail early, an error here is nor recoverable
		seed, err := LoadSeed(dcFile, seedName)
		if err != nil {
			return nil, err
		}
		return func() (*kubermaticv1.Seed, error) {
			return seed.DeepCopy(), nil
		}, nil
	}

	return func() (*kubermaticv1.Seed, error) {
		seed := &kubermaticv1.Seed{}
		if err := client.Get(ctx, types.NamespacedName{Name: seedName}, seed); err != nil {
			return nil, fmt.Errorf("failed to get seed %q: %v", seedName, err)
		}
		return seed, nil
	}, nil
}
