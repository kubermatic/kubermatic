package kubernetes

import (
	"bufio"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// Datacenter describes a Kubermatic datacenter.
type DatacenterMeta struct {
	Location string `yaml:"location"`
	Country  string `yaml:"country"`
	Provider string `yaml:"provider"`
}

// datacentersMetadata describes a number of Kubermatic datacenters.
type datacentersMeta struct {
	Datacenters map[string]DatacenterMeta `yaml:"datacenters"`
}

// Datacenters loads datacenter metadata from the given path.
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
