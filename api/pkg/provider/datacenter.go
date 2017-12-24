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

// OpenstackSpec describes a open stack datacenter
type OpenstackSpec struct {
	AuthURL          string `yaml:"auth_url"`
	AvailabilityZone string `yaml:"availability_zone"`
	// Used for automatic network creation
	DNSServers []string `yaml:"dns_servers"`
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

// BareMetalSpec describes a datacenter hosted on bare metal
type BareMetalSpec struct {
	URL          string `yaml:"url"`
	AuthUser     string `yaml:"auth-user"`
	AuthPassword string `yaml:"auth-password"`
}

// DatacenterSpec describes mutually points to provider datacenter spec
type DatacenterSpec struct {
	Digitalocean *DigitaloceanSpec `yaml:"digitalocean"`
	BringYourOwn *BringYourOwnSpec `yaml:"bringyourown"`
	AWS          *AWSSpec          `yaml:"aws"`
	BareMetal    *BareMetalSpec    `yaml:"baremetal"`
	Openstack    *OpenstackSpec    `yaml:"openstack"`
}

func (d *DatacenterSpec) AWSSpec() interface{} {
	return d.AWS
}

func (d *DatacenterSpec) FakeSpec() interface{} {
	return nil
}

func (d *DatacenterSpec) DigitaloceanSpec() interface{} {
	return d.Digitalocean
}

func (d *DatacenterSpec) BringYourOwnSpec() interface{} {
	return d.BringYourOwn
}

func (d *DatacenterSpec) BareMetalSpec() interface{} {
	return d.BareMetal
}

func (d *DatacenterSpec) OpenStackSpec() interface{} {
	return d.Openstack
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
