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
	"strconv"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/alibaba"
)

const (
	alibabaDefaultDiskType                = "cloud"
	alibabaDefaultDiskSize                = 40
	alibabaDefaultInternetMaxBandwidthOut = 10
)

type alibabaConfig struct {
	alibaba.RawConfig
}

func NewAlibabaConfig() *alibabaConfig {
	return &alibabaConfig{}
}

func (b *alibabaConfig) Build() alibaba.RawConfig {
	return b.RawConfig
}

func (b *alibabaConfig) WithInstanceType(instanceType string) *alibabaConfig {
	b.InstanceType.Value = instanceType
	return b
}

func (b *alibabaConfig) WithDiskType(diskType string) *alibabaConfig {
	b.DiskType.Value = diskType
	return b
}

func (b *alibabaConfig) WithDiskSize(size int) *alibabaConfig {
	b.DiskSize.Value = strconv.Itoa(size)
	return b
}

func (b *alibabaConfig) WithVSwitchID(switchID string) *alibabaConfig {
	b.VSwitchID.Value = switchID
	return b
}

func (b *alibabaConfig) WithInternetMaxBandwidthOut(mbps int) *alibabaConfig {
	b.InternetMaxBandwidthOut.Value = strconv.Itoa(mbps)
	return b
}

func (b *alibabaConfig) WithRegion(region string) *alibabaConfig {
	b.RegionID.Value = region
	return b
}

func (b *alibabaConfig) WithZone(zone string) *alibabaConfig {
	b.ZoneID.Value = zone
	return b
}

func CompleteAlibabaProviderSpec(config *alibaba.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecAlibaba) (*alibaba.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Alibaba == nil {
		return nil, fmt.Errorf("cannot use cluster to create Alibaba cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &alibaba.RawConfig{}
	}

	if config.DiskType.Value == "" {
		config.DiskType.Value = alibabaDefaultDiskType
	}

	if config.DiskSize.Value == "" {
		config.DiskSize.Value = strconv.Itoa(alibabaDefaultDiskSize)
	}

	if config.InternetMaxBandwidthOut.Value == "" {
		config.InternetMaxBandwidthOut.Value = strconv.Itoa(alibabaDefaultInternetMaxBandwidthOut)
	}

	if datacenter != nil {
		if config.RegionID.Value == "" {
			config.RegionID.Value = datacenter.Region
		}
	}

	if config.ZoneID.Value == "" && config.RegionID.Value != "" {
		config.ZoneID.Value = fmt.Sprintf("%sa", config.RegionID.Value)
	}

	return config, nil
}
