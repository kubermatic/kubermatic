package provider

import (
	"errors"
	"fmt"
	"testing"

	utilpointer "k8s.io/utils/pointer"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

func TestDatacenterMetasToSeedDatacenterSpecs(t *testing.T) {
	testCases := []struct {
		name           string
		datacenterMeta map[string]DatacenterMeta
		verify         func(map[string]*kubermaticv1.SeedDatacenter, error) error
	}{
		{
			name: "Setting both IsSeed and Seed errors",
			datacenterMeta: map[string]DatacenterMeta{
				"my-dc": {
					Seed:   "my-seed",
					IsSeed: true,
				},
			},
			verify: func(_ map[string]*kubermaticv1.SeedDatacenter, err error) error {
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
			verify: func(_ map[string]*kubermaticv1.SeedDatacenter, err error) error {
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
			verify: func(_ map[string]*kubermaticv1.SeedDatacenter, err error) error {
				expectedErr := `seedDatacenter "my-seed" used by nodeDatacenter "my-dc" does not exist`
				if err == nil || err.Error() != expectedErr {
					return fmt.Errorf("Expected error to be %q, was %v", expectedErr, err)
				}
				return nil
			},
		},
		{
			name: "Referencing a nodeLocation as seed causes error",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":                 {IsSeed: true},
				"my-valid-nodelocation":   {Seed: "my-seed"},
				"my-invalid-nodelocation": {Seed: "my-valid-nodelocation"},
			},
			verify: func(_ map[string]*kubermaticv1.SeedDatacenter, err error) error {
				expectedErr := `datacenter "my-valid-nodelocation" referenced by nodeDatacenter "my-invalid-nodelocation" as its seed is not configured to be a seed`
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
					SeedDNSOverwrite: utilpointer.StringPtr("dns-overwrite"),
				},
			},
			verify: func(seeds map[string]*kubermaticv1.SeedDatacenter, err error) error {
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
				if seeds["my-seed"].Spec.SeedDNSOverwrite == nil || *seeds["my-seed"].Spec.SeedDNSOverwrite != "dns-overwrite" {
					return fmt.Errorf("Expected .Spec.SeedDNSOverwrite to be 'dns-overwrite', was %v", seeds["my-seed"].Spec.SeedDNSOverwrite)
				}
				return nil
			},
		},
		{
			name: "All nodeLocation properties get copied over",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed": {IsSeed: true},
				"my-nodelocation": {
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
			verify: func(seeds map[string]*kubermaticv1.SeedDatacenter, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("'my-seed' is nil")
				}
				if seeds["my-seed"].Spec.NodeLocations == nil {
					return errors.New(".Spec.NodeLocations is nil")
				}
				nodeLocation, exists := seeds["my-seed"].Spec.NodeLocations["my-nodelocation"]
				if !exists {
					return errors.New(`.Spec.NodeLocations["my-nodelocation"] doesnt exist`)
				}
				if nodeLocation.Country != "Germany" {
					return fmt.Errorf("Expected nodeLocation.Country to be 'Germany', was %q", nodeLocation.Country)
				}
				if nodeLocation.Location != "Hamburg" {
					return fmt.Errorf("Expected nodeLocation.Location to be 'Hamburg', was %q", nodeLocation.Location)
				}
				if nodeLocation.Node.PauseImage != "pause" {
					return fmt.Errorf("Expected nodeLocation.Node.PauseImage to be 'pause', was %q", nodeLocation.Node.PauseImage)
				}
				if nodeLocation.Spec.Digitalocean == nil || nodeLocation.Spec.Digitalocean.Region != "Amsterdam" {
					return fmt.Errorf("Expected nodeLocation.Spec.Digitalocean to be 'Amsterdam', was %v", nodeLocation.Spec.Digitalocean)
				}
				return nil
			},
		},
		{
			name: "One seed, one nodeLocation",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":         {IsSeed: true},
				"my-nodelocation": {Seed: "my-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.SeedDatacenter, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed")
				}
				if _, exists := seeds["my-seed"].Spec.NodeLocations["my-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-nodelocation' doesnt exist")
				}
				return nil
			},
		},
		{
			name: "One seed, multiple nodeLocations",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":                {IsSeed: true},
				"my-nodelocation":        {Seed: "my-seed"},
				"my-second-nodelocation": {Seed: "my-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.SeedDatacenter, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed")
				}
				if _, exists := seeds["my-seed"].Spec.NodeLocations["my-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-nodelocation' doesnt exist")
				}
				if _, exists := seeds["my-seed"].Spec.NodeLocations["my-second-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-second-nodelocation' doesnt exist")
				}
				return nil
			},
		},
		{
			name: "Multiple seed with one nodeLocation each",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":                {IsSeed: true},
				"my-nodelocation":        {Seed: "my-seed"},
				"my-second-seed":         {IsSeed: true},
				"my-second-nodelocation": {Seed: "my-second-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.SeedDatacenter, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed 'my-seed'")
				}
				if seeds["my-second-seed"] == nil {
					return errors.New("Couldnt find seed 'my-second-seed'")
				}
				if _, exists := seeds["my-seed"].Spec.NodeLocations["my-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-nodelocation' doesnt exist")
				}
				if _, exists := seeds["my-second-seed"].Spec.NodeLocations["my-second-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-second-nodelocation' doesnt exist")
				}
				return nil
			},
		},
		{
			name: "Multiple seeds with multiple nodeLocations each",
			datacenterMeta: map[string]DatacenterMeta{
				"my-seed":                {IsSeed: true},
				"my-nodelocation":        {Seed: "my-seed"},
				"my-third-nodelocation":  {Seed: "my-seed"},
				"my-second-seed":         {IsSeed: true},
				"my-second-nodelocation": {Seed: "my-second-seed"},
				"my-fourth-nodelocation": {Seed: "my-second-seed"},
			},
			verify: func(seeds map[string]*kubermaticv1.SeedDatacenter, err error) error {
				if err != nil {
					return fmt.Errorf("Expected error to be nil, was %v", err)
				}
				if seeds["my-seed"] == nil {
					return errors.New("Couldnt find seed 'my-seed'")
				}
				if seeds["my-second-seed"] == nil {
					return errors.New("Couldnt find seed 'my-second-seed'")
				}
				if _, exists := seeds["my-seed"].Spec.NodeLocations["my-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-nodelocation' doesnt exist")
				}
				if _, exists := seeds["my-seed"].Spec.NodeLocations["my-third-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-third-nodelocation' doesnt exist")
				}
				if _, exists := seeds["my-second-seed"].Spec.NodeLocations["my-second-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-second-nodelocation' doesnt exist")
				}
				if _, exists := seeds["my-second-seed"].Spec.NodeLocations["my-fourth-nodelocation"]; !exists {
					return errors.New("Nodelocation 'my-fourth-nodelocation' doesnt exist")
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
			result, err := DatacenterMetasToSeedDatacenterSpecs(testCase.datacenterMeta)
			if tcErr := testCase.verify(result, err); tcErr != nil {
				t.Fatalf(tcErr.Error())
			}
		})
	}
}
