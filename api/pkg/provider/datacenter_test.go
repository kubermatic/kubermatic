package provider

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoadDatacentersMeta(t *testing.T) {
	datacentersYAML := `
datacenters:
#==================================
#===============Seed===============
#==================================
  europe-west3-c: #Master
    location: Frankfurt
    country: DE
    is_seed: true
    spec:
      bringyourown: {}
#==================================
#===========Digitalocean===========
#==================================
  do-ams3:
    location: Amsterdam
    seed: europe-west3-c
    country: NL
    spec:
      digitalocean:
        region: ams3
#==================================
#============OpenStack=============
#==================================
  auos-1:
    location: Australia
    seed: europe-west3-c
    country: AU
    spec:
      openstack:
        availability_zone: au1
        region: au
        dns_servers:
        - "8.8.8.8"
        - "8.8.4.4"
        images:
          ubuntu: "Ubuntu 18.04 LTS - 2018-08-10"
          centos: ""
          coreos: ""
        enforce_floating_ip: true`
	expectedSeeds := map[string]*kubermaticv1.Seed{
		"europe-west3-c": {
			ObjectMeta: metav1.ObjectMeta{
				Name: "europe-west3-c",
			},
			Spec: kubermaticv1.SeedSpec{
				Location: "Frankfurt",
				Country:  "DE",
				Datacenters: map[string]kubermaticv1.Datacenter{
					"do-ams3": {
						Location: "Amsterdam",
						Country:  "NL",
						Spec: kubermaticv1.DatacenterSpec{
							Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
								Region: "ams3",
							},
						},
					},
					"auos-1": {
						Location: "Australia",
						Country:  "AU",
						Spec: kubermaticv1.DatacenterSpec{
							Openstack: &kubermaticv1.DatacenterSpecOpenstack{
								AvailabilityZone: "au1",
								Region:           "au",
								DNSServers:       []string{"8.8.8.8", "8.8.4.4"},
								Images: kubermaticv1.ImageList{
									providerconfig.OperatingSystemUbuntu: "Ubuntu 18.04 LTS - 2018-08-10",
									providerconfig.OperatingSystemCentOS: "",
									providerconfig.OperatingSystemCoreos: "",
								},
								EnforceFloatingIP: true,
							},
						},
					},
				},
			},
		},
	}

	file, err := ioutil.TempFile(os.TempDir(), "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.Remove(file.Name()) // nolint: errcheck
	defer file.Close()           // nolint: errcheck

	_, err = file.WriteString(datacentersYAML)
	assert.NoError(t, err)

	err = file.Sync()
	assert.NoError(t, err)

	resultDatacenters, err := LoadSeeds(file.Name())
	if err != nil {
		t.Fatalf("failed to load datacenters: %v", err)
	}

	assert.Equal(t, expectedSeeds, resultDatacenters)
}

func TestMigrateDatacenters(t *testing.T) {
	testCases := []struct {
		name        string
		datacenters map[string]DatacenterMeta
		errExpected bool
	}{
		{
			name: "Invalid name, error",
			datacenters: map[string]DatacenterMeta{
				"&invalid": {
					IsSeed: true,
				},
			},
			errExpected: true,
		},
		{
			name: "Valid name succeeds",
			datacenters: map[string]DatacenterMeta{
				"valid": {
					IsSeed: true,
				},
			},
		},
		{
			name: "Invalid name, valid seed dns override",
			datacenters: map[string]DatacenterMeta{
				"&invalid": {
					IsSeed:           true,
					SeedDNSOverwrite: "valid",
				},
			},
		},
		{
			name: "Valid name, invalid seed dns override",
			datacenters: map[string]DatacenterMeta{
				"valid": {
					IsSeed:           true,
					SeedDNSOverwrite: "&invalid",
				},
			},
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			seeds, err := DatacenterMetasToSeeds(tc.datacenters)
			if err != nil {
				t.Fatalf("Failed to convert datacenters to seeds: %v", err)
			}

			for _, seed := range seeds {
				if err := ValidateSeed(seed); (err != nil) != tc.errExpected {
					t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
				}
			}
		})
	}
}

func TestSeedGetterFactorySetsDefaults(t *testing.T) {
	t.Parallel()
	initSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-seed",
			Namespace: "my-ns",
		},
		Spec: kubermaticv1.SeedSpec{
			ProxySettings: &kubermaticv1.ProxySettings{
				HTTPProxy: kubermaticv1.NewProxyValue("seed-proxy"),
			},
			Datacenters: map[string]kubermaticv1.Datacenter{"a": {}},
		},
	}
	client := fakectrlruntimeclient.NewFakeClient(initSeed)

	seedGetter, err := SeedGetterFactory(context.Background(), client, "my-seed", "", "my-ns", true)
	if err != nil {
		t.Fatalf("failed getting seedGetter: %v", err)
	}
	seed, err := seedGetter()
	if err != nil {
		t.Fatalf("failed calling seedGetter: %v", err)
	}
	if seed.Spec.Datacenters["a"].Node.ProxySettings.HTTPProxy.String() != "seed-proxy" {
		t.Errorf("expected the datacenters http proxy setting to get set but was %v",
			seed.Spec.Datacenters["a"].Node.ProxySettings.HTTPProxy)
	}
}

func TestSeedsGetterFactorySetsDefaults(t *testing.T) {
	t.Parallel()
	initSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-seed",
			Namespace: "my-ns",
		},
		Spec: kubermaticv1.SeedSpec{
			ProxySettings: &kubermaticv1.ProxySettings{
				HTTPProxy: kubermaticv1.NewProxyValue("seed-proxy"),
			},
			Datacenters: map[string]kubermaticv1.Datacenter{"a": {}},
		},
	}
	client := fakectrlruntimeclient.NewFakeClient(initSeed)

	seedsGetter, err := SeedsGetterFactory(context.Background(), client, "", "my-ns", "", true)
	if err != nil {
		t.Fatalf("failed getting seedsGetter: %v", err)
	}
	seeds, err := seedsGetter()
	if err != nil {
		t.Fatalf("failed calling seedsGetter: %v", err)
	}
	if _, exists := seeds["my-seed"]; !exists || len(seeds) != 1 {
		t.Fatalf("expceted to get a map with exactly one key `my-seed`, got %v", seeds)
	}
	seed := seeds["my-seed"]
	if seed.Spec.Datacenters["a"].Node.ProxySettings.HTTPProxy.String() != "seed-proxy" {
		t.Errorf("expected the datacenters http proxy setting to get set but was %v",
			seed.Spec.Datacenters["a"].Node.ProxySettings.HTTPProxy)
	}
}
