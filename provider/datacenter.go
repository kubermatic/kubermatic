package provider

import (
	"bytes"
	"hash/fnv"
	"io/ioutil"

	"github.com/kubermatic/api"

	"encoding/base64"

	yaml "gopkg.in/yaml.v2"
)

// Datacenter holds information of one Datacenter.
type Datacenter struct {
	ID        string `yaml:"-"`
	KubeletID string `yaml:"kubelet-id"`

	Metainfo api.DatacenterSpec `yaml:"metainfo"`
	Network  api.NetworkSpec    `yaml:"network"`
	// BUG: Where is this used ?
	ApiserverSSH struct {
		Private string `yaml:"private"`
		Public  string `yaml:"public"`
	} `yaml:"apiserverSSH"`
}

// Datacenters describes all running datacenters.
type Datacenters struct {
	// Provider are all seed datacenters. Each key is the name of a datacenter provider.
	Drivers map[string][]Datacenter `yaml:"driver"`
}

// init creates the uid for each DC which is used by the computer.
// It also sets the Driver type from the group it was placed in.
func (d *Datacenters) init() error {
	var bufsum *bytes.Buffer
	var err error
	hash := fnv.New32a()
	for name, providers := range d.Drivers {
		for _, provider := range providers {
			provider.DriverType = name
			_, err = bufsum.WriteString(provider.Metainfo.ExactName)
			if err != nil {
				return err
			}
			_, err = bufsum.WriteString(provider.Metainfo.Location)
			if err != nil {
				return err
			}
			_, err = bufsum.WriteString(provider.Metainfo.Country)
			if err != nil {
				return err
			}
			_, err = bufsum.WriteString(provider.Metainfo.DriverType)
			if err != nil {
				return err
			}

			provider.ID = base64.URLEncoding.EncodeToString(hash.Sum(bufsum.Bytes()))
			bufsum.Reset()
			hash.Reset()
		}
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
