package provider

import (
	"bufio"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// DigitaloceanSpec describes a digital ocean datacenter
type DigitaloceanSpec struct {
	Region string `yaml:"region"`
}

// AWSSpec describes a digital ocean datacenter
type AWSSpec struct {
	Region string `yaml:"region"`
	AMI    string `yaml:"ami"`
	Zone   string `yaml:"zone"`
}

// BringYourOwnSpec describes a datacenter our of bring your own nodes
type BringYourOwnSpec struct {
}

// BareMetalSpec describes a datacenter hosted on bare metal
type BareMetalSpec struct {
	URL string `yaml:"url"`
}

// SeedSpec describes a seed in the given datacenter.
type SeedSpec struct {
	Digitalocean struct {
		SSHKeys []string `yaml:"sshKeys"`
	} `yaml:"digitalocean"`
	BringYourOwn struct {
		PrivateIntf string `yaml:"privateInterface"`
	} `yaml:"bringyourown"`
	AWS struct {
		SSHKeyName string `yaml:"ssh_key_fingerprint"`
	} `yaml:"aws"`

	Network struct {
		Flannel struct {
			CIDR string `yaml:"cidr"`
		} `yaml:"flannel"`
	} `yaml:"network"`

	ApiserverSSH struct {
		Private string `yaml:"private"`
		Public  string `yaml:"public"`
	} `yaml:"apiserverSSH"`
}

// DatacenterSpec describes mutually points to provider datacenter spec
type DatacenterSpec struct {
	Digitalocean *DigitaloceanSpec `yaml:"digitalocean"`
	BringYourOwn *BringYourOwnSpec `yaml:"bringyourown"`
	AWS          *AWSSpec          `yaml:"aws"`
	BareMetal    *BareMetalSpec    `yaml:"baremetal"`

	Seed SeedSpec `yaml:"seed"`
}

// DatacenterMeta describes a Kubermatic datacenter.
type DatacenterMeta struct {
	Location string         `yaml:"location"`
	Seed     string         `yaml:"seed"`
	Country  string         `yaml:"country"`
	Spec     DatacenterSpec `yaml:"spec"`
	Private  bool           `yaml:"private"`
}

// datacentersMeta describes a number of Kubermatic datacenters.
type datacentersMeta struct {
	Datacenters map[string]DatacenterMeta `yaml:"datacenters"`
}

// DatacentersMeta loads datacenter metadata from the given path.
func DatacentersMeta(path string) (map[string]DatacenterMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	dcs := datacentersMeta{}
	err = yaml.Unmarshal(bytes, &dcs)
	if err != nil {
		return nil, err
	}

	return dcs.Datacenters, nil
}
