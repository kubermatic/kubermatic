package provider

import (
	"bufio"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// DigitaloceanSpec describes a DigitalOcean datacenter
type DigitaloceanSpec struct {
	Region string `yaml:"region"`
}

// HetznerSpec describes a Hetzner cloud datacenter
type HetznerSpec struct {
	Datacenter string `yaml:"datacenter"`
	Location   string `yaml:"location"`
}

// OpenstackSpec describes a open stack datacenter
type OpenstackSpec struct {
	AuthURL          string `yaml:"auth_url"`
	AvailabilityZone string `yaml:"availability_zone"`
	Region           string `yaml:"region"`
	IgnoreVolumeAZ   bool   `yaml:"ignore_volume_az"`
	// Used for automatic network creation
	DNSServers []string `yaml:"dns_servers"`
}

type VMWareSpec struct {
}

// AWSSpec describes a digital ocean datacenter
type AWSSpec struct {
	Region        string `yaml:"region"`
	AMI           string `yaml:"ami"`
	ZoneCharacter string `yaml:"zone_character"`
}

// BringYourOwnSpec describes a datacenter our of bring your own nodes
type BringYourOwnSpec struct {
}

// DatacenterSpec describes mutually points to provider datacenter spec
type DatacenterSpec struct {
	Digitalocean *DigitaloceanSpec `yaml:"digitalocean"`
	BringYourOwn *BringYourOwnSpec `yaml:"bringyourown"`
	AWS          *AWSSpec          `yaml:"aws"`
	Openstack    *OpenstackSpec    `yaml:"openstack"`
	Hetzner      *HetznerSpec      `yaml:"hetzner"`
	VMWare       *VMWareSpec       `yaml:"vmware"`
}

// DatacenterMeta describes a Kubermatic datacenter.
type DatacenterMeta struct {
	Location string         `yaml:"location"`
	Seed     string         `yaml:"seed"`
	Country  string         `yaml:"country"`
	Spec     DatacenterSpec `yaml:"spec"`
	Private  bool           `yaml:"private"`
	IsSeed   bool           `yaml:"is_seed"`
}

// datacentersMeta describes a number of Kubermatic datacenters.
type datacentersMeta struct {
	Datacenters map[string]DatacenterMeta `yaml:"datacenters"`
}

// LoadDatacentersMeta loads datacenter metadata from the given path.
func LoadDatacentersMeta(path string) (map[string]DatacenterMeta, error) {
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
