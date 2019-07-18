package provider

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
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

func TestValidateDataCenters(t *testing.T) {
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
					SeedDNSOverwrite: utilpointer.StringPtr("valid"),
				},
			},
		},
		{
			name: "Valid name, invalid seed dns override",
			datacenters: map[string]DatacenterMeta{
				"valid": {
					IsSeed:           true,
					SeedDNSOverwrite: utilpointer.StringPtr("&invalid"),
				},
			},
			errExpected: true,
		},
		{
			name: "Invalid name, but is not a seed",
			datacenters: map[string]DatacenterMeta{
				"&invalid": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDatacenters(tc.datacenters); (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
		})
	}
}
