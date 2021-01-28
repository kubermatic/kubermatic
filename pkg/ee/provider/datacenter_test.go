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
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
          flatcar: ""
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
									providerconfig.OperatingSystemUbuntu:  "Ubuntu 18.04 LTS - 2018-08-10",
									providerconfig.OperatingSystemCentOS:  "",
									providerconfig.OperatingSystemFlatcar: "",
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
