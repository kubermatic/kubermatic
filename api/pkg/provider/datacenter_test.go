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
  syseleven-dbl1:
    location: Syseleven - dbl1
    seed: europe-west3-c
    country: DE
    provider: openstack
    spec:
      openstack:
        auth_url: https://api.cbk.cloud.syseleven.net:5000/v3
        availability_zone: dbl1
        region: dbl
        dns_servers:
        - "37.123.105.116"
        - "37.123.105.117"
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
		"syseleven-dbl1": {
			Location: "Syseleven - dbl1",
			Seed:     "europe-west3-c",
			Country:  "DE",
			Spec: DatacenterSpec{
				Openstack: &OpenstackSpec{
					AuthURL:          "https://api.cbk.cloud.syseleven.net:5000/v3",
					AvailabilityZone: "dbl1",
					Region:           "dbl",
					DNSServers:       []string{"37.123.105.116", "37.123.105.117"},
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
