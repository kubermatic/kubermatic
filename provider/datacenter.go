package provider

import (
	"bytes"
	"hash/fnv"
	"io/ioutil"

	"github.com/kubermatic/api/provider/drivers/flag"

	yaml "gopkg.in/yaml.v2"
)

// Datacenters describes all running datacenters.
type Datacenters struct {
	// Provider are all seed datacenters. Each key is the name of a datacenter provider.
	Provider map[string]struct {
		ProviderType string `yaml:"-"` // e.g "aws", "digitalocean"

		// Metainformation
		Location string `yaml:"location"`
		Country  string `yaml:"country"`
		Private  bool   `yaml:"private"`

		// Routing information
		Region    string `yaml:"region,omitempty"`
		Zone      string `yaml:"zone,omitempty"`
		Exactname string `yaml:"exact-name"`

		CustomerPatches []flag.Flags `yaml:"customer-patches,omitempty"` // Patches that are applied to customer clusters.
		SeedPatches     []flag.Flags `yaml:"seed-patches,omitempty"`     // Patches that are applied to seed clusters.

		Network struct {
			Flannel struct {
				CIDR string `yaml:"cidr"`
			} `yaml:"flannel"`
		} `yaml:"network"`

		ApiserverSSH struct {
			Private string `yaml:"private"`
			Public  string `yaml:"public"`
		} `yaml:"apiserverSSH"`

		UniqueID string `yaml:"-"`
	}
}

func (d *Datacenters) init() error {
	var buf *bytes.Buffer
	var err error
	hash := fnv.New32a()
	for name, provider := range d.Provider {
		provider.ProviderType = name
		_, err = buf.WriteString(provider.Exactname)
		if err != nil {
			return err
		}
		_, err = buf.WriteString(provider.Location)
		if err != nil {
			return err
		}
		_, err = buf.WriteString(provider.Country)
		if err != nil {
			return err
		}
		_, err = buf.WriteString(provider.ProviderType)
		if err != nil {
			return err
		}
		provider.UniqueID = string(hash.Sum(buf.Bytes()))
		buf.Reset()
		hash.Reset()
	}
	return nil
}

// UnmarshalYAML takes a binary yaml file and parses it into a Datacenters struct,
// containing all drivers.
func UnmarshalYAML(data []byte) (dcs *Datacenters, err error) {
	err = yaml.Unmarshal(data, dcs)
	if err != nil {
		return nil, err
	}
	err = dcs.init()
	return
}

// ParseDatacenterFile parses a file containing seed datacenter information from it's filepath
func ParseDatacenterFile(filepath string) (*Datacenters, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return UnmarshalYAML(data)
}
