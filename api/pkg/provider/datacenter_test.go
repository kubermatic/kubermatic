package provider

import (
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
    provider: Loodse
    is_seed: true
    spec:
      bringyourown:
        region: DE
      seed:
        bringyourown:
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
    seed: sydney-1
    country: AU
    provider: openstack
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
	expectedDatacenters := map[string]DatacenterMeta{
		"europe-west3-c": {
			Location: "Frankfurt",
			Seed:     "",
			Country:  "DE",
			Spec: DatacenterSpec{
				BringYourOwn: &BringYourOwnSpec{},
			},
			Private:          false,
			IsSeed:           true,
			SeedDNSOverwrite: nil,
		},
		"do-ams3": {
			Location: "Amsterdam",
			Seed:     "europe-west3-c",
			Country:  "NL",
			Spec: DatacenterSpec{
				Digitalocean: &DigitaloceanSpec{
					Region: "ams3",
				},
			},
			Private:          false,
			IsSeed:           false,
			SeedDNSOverwrite: nil,
		},
		"auos-1": {
			Location: "Australia",
			Seed:     "sydney-1",
			Country:  "AU",
			Spec: DatacenterSpec{
				Openstack: &OpenstackSpec{
					AvailabilityZone: "au1",
					Region:           "au",
					DNSServers:       []string{"8.8.8.8", "8.8.4.4"},
					Images: ImageList{
						providerconfig.OperatingSystemUbuntu: "Ubuntu 18.04 LTS - 2018-08-10",
						providerconfig.OperatingSystemCentOS: "",
						providerconfig.OperatingSystemCoreos: "",
					},
					EnforceFloatingIP: true,
				},
			},
			Private:          false,
			IsSeed:           false,
			SeedDNSOverwrite: nil,
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

	resultDatacenters, err := LoadDatacentersMeta(file.Name())
	assert.NoError(t, err)

	assert.Equal(t, expectedDatacenters, resultDatacenters)
}
