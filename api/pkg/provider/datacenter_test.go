package provider

import (
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
        region: ams3`
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
