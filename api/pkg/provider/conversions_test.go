package provider

import (
	"errors"
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

func TestDatacenterMetasToSeedDatacenterSpecs(t *testing.T) {
	testCases := []struct {
		name           string
		datacenterMeta map[string]DatacenterMeta
		verify         func(map[string]*kubermaticv1.Seed, error) error
	}{
		{
			name: "Setting both IsSeed and Seed errors",
			datacenterMeta: map[string]DatacenterMeta{
				"my-dc": {
					Seed:   "my-seed",
					IsSeed: true,
				},
			},
			verify: func(_ map[string]*kubermaticv1.Seed, err error) error {
				if err == nil {
					return errors.New("expected error when both IsSeed and Seed are set")
				}
				return nil
			},
		},
		{
			name: "Not seed and missing seed name causes error",
			datacenterMeta: map[string]DatacenterMeta{
				"my-dc": {},
			},
			verify: func(_ map[string]*kubermaticv1.Seed, err error) error {
				if err == nil {
					return errors.New("expected error for node datacenter that has no seed configured")
				}
				return nil
			},
		},
		{
			name: "Referencing non-existing seed causes error",
			datacenterMeta: map[string]DatacenterMeta{
				"my-dc": {
					Seed: "my-seed",
				},
			},
			verify: func(_ map[string]*kubermaticv1.Seed, err error) error {
				expectedErr := `seedDatacenter "my-seed" used by node datacenter "my-dc" does not exist`
				if err == nil || err.Error() != expectedErr {
					return fmt.Errorf("Expected error to be %q, was %v", expectedErr, err)
				}
				return nil
			},
		},
		{
			name: "Referencing a Datacenter as seed causes error",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":               {IsSeed: true},
				"my-valid-datacenter":   {Seed: "my-seed"},
				"my-invalid-datacenter": {Seed: "my-valid-datacenter"},
			},
			verify: func(_ map[string]*kubermaticv1.Seed, err error) error {
				expectedErr := `datacenter "my-valid-datacenter" referenced by node datacenter "my-invalid-datacenter" as its seed is not configured to be a seed`
				if err == nil || err.Error() != expectedErr {
					return fmt.Errorf("expected error to be %q, was %v", expectedErr, err)
				}
				return nil
			},
		},
		{
			name: "All seed properties get copied over",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed": {
					IsSeed:           true,
					Location:         "Hamburg",
					Country:          "Germany",
					SeedDNSOverwrite: "dns-overwrite",
				},
			},
			verify: func(seeds map[string]*kubermaticv1.Seed, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("my-seed is nil")
				}
				if seeds["my-seed"].Name != "my-seed" {
					return fmt.Errorf("Expected Name to be 'my-seed', was %q", seeds["my-seed"].Name)
				}
				if seeds["my-seed"].Spec.Location != "Hamburg" {
					return fmt.Errorf("expected .Spec.Location to be 'Hamburg', was %q", seeds["my-seed"].Spec.Location)
				}
				if seeds["my-seed"].Spec.Country != "Germany" {
					return fmt.Errorf("expected .Spec.Country to be 'Germany', was %q", seeds["my-seed"].Spec.Country)
				}
				if seeds["my-seed"].Spec.SeedDNSOverwrite != "dns-overwrite" {
					return fmt.Errorf("Expected .Spec.SeedDNSOverwrite to be 'dns-overwrite', was %q", seeds["my-seed"].Spec.SeedDNSOverwrite)
				}
				return nil
			},
		},
		{
			name: "All datacenter properties get copied over",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed": {IsSeed: true},
				"my-datacenter": {
					Seed:     "my-seed",
					Location: "Hamburg",
					Country:  "Germany",
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
							Region: "Amsterdam",
						},
					},
					Node: kubermaticv1.NodeSettings{
						PauseImage: "pause",
					},
				},
			},
			verify: func(seeds map[string]*kubermaticv1.Seed, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("'my-seed' is nil")
				}
				if seeds["my-seed"].Spec.Datacenters == nil {
					return errors.New(".Spec.Datacenters is nil")
				}
				datacenter, exists := seeds["my-seed"].Spec.Datacenters["my-datacenter"]
				if !exists {
					return errors.New(`.Spec.Datacenters["my-datacenter"] doesnt exist`)
				}
				if datacenter.Country != "Germany" {
					return fmt.Errorf("Expected datacenter.Country to be 'Germany', was %q", datacenter.Country)
				}
				if datacenter.Location != "Hamburg" {
					return fmt.Errorf("Expected datacenter.Location to be 'Hamburg', was %q", datacenter.Location)
				}
				if datacenter.Node.PauseImage != "pause" {
					return fmt.Errorf("Expected datacenter.Node.PauseImage to be 'pause', was %q", datacenter.Node.PauseImage)
				}
				if datacenter.Spec.Digitalocean == nil || datacenter.Spec.Digitalocean.Region != "Amsterdam" {
					return fmt.Errorf("Expected datacenter.Spec.Digitalocean to be 'Amsterdam', was %v", datacenter.Spec.Digitalocean)
				}
				return nil
			},
		},
		{
			name: "One seed, one datacenter",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":       {IsSeed: true},
				"my-datacenter": {Seed: "my-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.Seed, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed")
				}
				if _, exists := seeds["my-seed"].Spec.Datacenters["my-datacenter"]; !exists {
					return errors.New("Datacenter 'my-datacenter' doesnt exist")
				}
				return nil
			},
		},
		{
			name: "One seed, multiple datacenters",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":              {IsSeed: true},
				"my-datacenter":        {Seed: "my-seed"},
				"my-second-datacenter": {Seed: "my-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.Seed, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed")
				}
				if _, exists := seeds["my-seed"].Spec.Datacenters["my-datacenter"]; !exists {
					return errors.New("Datacenter 'my-datacenter' doesnt exist")
				}
				if _, exists := seeds["my-seed"].Spec.Datacenters["my-second-datacenter"]; !exists {
					return errors.New("Datacenter 'my-second-datacenter' doesnt exist")
				}
				return nil
			},
		},
		{
			name: "Multiple seed with one datacenter each",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":              {IsSeed: true},
				"my-datacenter":        {Seed: "my-seed"},
				"my-second-seed":       {IsSeed: true},
				"my-second-datacenter": {Seed: "my-second-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.Seed, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed 'my-seed'")
				}
				if seeds["my-second-seed"] == nil {
					return errors.New("Couldnt find seed 'my-second-seed'")
				}
				if _, exists := seeds["my-seed"].Spec.Datacenters["my-datacenter"]; !exists {
					return errors.New("Datacenter 'my-datacenter' doesnt exist")
				}
				if _, exists := seeds["my-second-seed"].Spec.Datacenters["my-second-datacenter"]; !exists {
					return errors.New("Datacenter 'my-second-datacenter' doesnt exist")
				}
				return nil
			},
		},
		{
			name: "Multiple seeds with multiple datacenters each",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":              {IsSeed: true},
				"my-datacenter":        {Seed: "my-seed"},
				"my-third-datacenter":  {Seed: "my-seed"},
				"my-second-seed":       {IsSeed: true},
				"my-second-datacenter": {Seed: "my-second-seed"},
				"my-fourth-datacenter": {Seed: "my-second-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.Seed, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed 'my-seed'")
				}
				if seeds["my-second-seed"] == nil {
					return errors.New("Couldnt find seed 'my-second-seed'")
				}
				if _, exists := seeds["my-seed"].Spec.Datacenters["my-datacenter"]; !exists {
					return errors.New("Datacenter 'my-datacenter' doesnt exist")
				}
				if _, exists := seeds["my-seed"].Spec.Datacenters["my-third-datacenter"]; !exists {
					return errors.New("Datacenter 'my-third-datacenter' doesnt exist")
				}
				if _, exists := seeds["my-second-seed"].Spec.Datacenters["my-second-datacenter"]; !exists {
					return errors.New("Datacenter 'my-second-datacenter' doesnt exist")
				}
				if _, exists := seeds["my-second-seed"].Spec.Datacenters["my-fourth-datacenter"]; !exists {
					return errors.New("Datacenter 'my-fourth-datacenter' doesnt exist")
				}
				return nil
			},
		},
	}

	for _, testCase := range testCases {
		if testCase.datacenterMeta == nil {
			continue
		}
		t.Run(testCase.name, func(t *testing.T) {
			result, err := DatacenterMetasToSeeds(testCase.datacenterMeta)
			if tcErr := testCase.verify(result, err); tcErr != nil {
				t.Fatalf(tcErr.Error())
			}
		})
	}
}
