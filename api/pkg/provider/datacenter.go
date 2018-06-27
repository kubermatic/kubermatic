package provider

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/util/sets"
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

// OperatingSystem type definition for the operating system
type OperatingSystem string

const (
	// OperatingSystemCoreos name of the operating system coreos
	OperatingSystemCoreos OperatingSystem = "coreos"
	// OperatingSystemUbuntu name of the operating system ubuntu
	OperatingSystemUbuntu OperatingSystem = "ubuntu"
	// OperatingSystemCentOS name of the operating system centos
	OperatingSystemCentOS OperatingSystem = "centos"
)

var (
	// AllOperatingSystems defines all available operating systems
	AllOperatingSystems = sets.NewString(string(OperatingSystemCoreos), string(OperatingSystemCentOS), string(OperatingSystemUbuntu))
)

// ImageList defines a map of operating system and the image to use
type ImageList map[OperatingSystem]string

// OpenstackSpec describes a open stack datacenter
type OpenstackSpec struct {
	AuthURL          string `yaml:"auth_url"`
	AvailabilityZone string `yaml:"availability_zone"`
	Region           string `yaml:"region"`
	IgnoreVolumeAZ   bool   `yaml:"ignore_volume_az"`
	// Used for automatic network creation
	DNSServers []string  `yaml:"dns_servers"`
	Images     ImageList `yaml:"images"`
}

// AzureSpec describes an Azure cloud datacenter
type AzureSpec struct {
	Location string `yaml:"location"`
}

// VSphereSpec describes a vsphere datacenter
type VSphereSpec struct {
	Endpoint      string `yaml:"endpoint"`
	AllowInsecure bool   `yaml:"allow_insecure"`

	Datastore  string    `yaml:"datastore"`
	Datacenter string    `yaml:"datacenter"`
	Cluster    string    `yaml:"cluster"`
	RootPath   string    `yaml:"root_path"`
	Templates  ImageList `yaml:"templates"`
}

// AWSSpec describes a aws datacenter
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
	Azure        *AzureSpec        `yaml:"azure"`
	Openstack    *OpenstackSpec    `yaml:"openstack"`
	Hetzner      *HetznerSpec      `yaml:"hetzner"`
	VSphere      *VSphereSpec      `yaml:"vsphere"`
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

	return dcs.Datacenters, validateDatacenters(dcs.Datacenters)
}

func validateImageList(images ImageList) error {
	for s := range images {
		if !AllOperatingSystems.Has(string(s)) {
			return fmt.Errorf("invalid operating system defined '%s'. Possible values: %s", s, strings.Join(AllOperatingSystems.List(), ","))
		}
	}

	return nil
}

func validateDatacenters(datacenters map[string]DatacenterMeta) error {
	for name, dc := range datacenters {
		if dc.Spec.VSphere != nil {
			if err := validateImageList(dc.Spec.VSphere.Templates); err != nil {
				return fmt.Errorf("invalid datacenter defined '%s': %v", name, err)
			}
		}
		if dc.Spec.Openstack != nil {
			if err := validateImageList(dc.Spec.Openstack.Images); err != nil {
				return fmt.Errorf("invalid datacenter defined '%s': %v", name, err)
			}
		}
	}

	return nil
}
