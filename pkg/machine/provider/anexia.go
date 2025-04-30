/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/anexia"
	"k8c.io/machine-controller/sdk/providerconfig"
)

type anexiaConfig struct {
	anexia.RawConfig
}

func NewAnexiaConfig() *anexiaConfig {
	return &anexiaConfig{}
}

func (b *anexiaConfig) Build() anexia.RawConfig {
	return b.RawConfig
}

func (b *anexiaConfig) WithLocationID(locationID string) *anexiaConfig {
	b.LocationID.Value = locationID
	return b
}

func (b *anexiaConfig) WithTemplateID(templateID string) *anexiaConfig {
	b.TemplateID.Value = templateID
	return b
}

func (b *anexiaConfig) WithVlanID(vlanID string) *anexiaConfig {
	b.VlanID.Value = vlanID
	return b
}

func (b *anexiaConfig) WithCPUs(cpus int) *anexiaConfig {
	b.CPUs = cpus
	return b
}

func (b *anexiaConfig) WithMemory(memory int) *anexiaConfig {
	b.Memory = memory
	return b
}

func (b *anexiaConfig) WithDiskSize(diskSize int) *anexiaConfig {
	b.DiskSize = diskSize
	return b
}

func (b *anexiaConfig) AddDisk(size int, performanceType string) *anexiaConfig {
	if b.Disks == nil {
		b.Disks = []anexia.RawDisk{}
	}

	b.Disks = append(b.Disks, anexia.RawDisk{
		Size:            size,
		PerformanceType: providerconfig.ConfigVarString{Value: performanceType},
	})

	return b
}

func CompleteAnexiaProviderSpec(config *anexia.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecAnexia) (*anexia.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Anexia == nil {
		return nil, fmt.Errorf("cannot use cluster to create Anexia cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &anexia.RawConfig{}
	}

	if datacenter != nil {
		if config.LocationID.Value == "" {
			config.LocationID.Value = datacenter.LocationID
		}
	}

	if config.DiskSize > 0 && len(config.Disks) > 0 {
		return nil, anexia.ErrConfigDiskSizeAndDisks
	}

	return config, nil
}
