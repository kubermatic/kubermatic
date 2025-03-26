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
	"k8c.io/machine-controller/sdk/cloudprovider/hetzner"
	"k8c.io/machine-controller/sdk/providerconfig"
)

type hetznerConfig struct {
	hetzner.RawConfig
}

func NewHetznerConfig() *hetznerConfig {
	return &hetznerConfig{}
}

func (b *hetznerConfig) Build() hetzner.RawConfig {
	return b.RawConfig
}

func (b *hetznerConfig) WithServerType(serverType string) *hetznerConfig {
	b.ServerType.Value = serverType
	return b
}

func (b *hetznerConfig) WithDatacenter(datacenter string) *hetznerConfig {
	b.Datacenter.Value = datacenter
	return b
}

func (b *hetznerConfig) WithImage(image string) *hetznerConfig {
	b.Image.Value = image
	return b
}

func (b *hetznerConfig) WithLocation(location string) *hetznerConfig {
	b.Location.Value = location
	return b
}

func (b *hetznerConfig) WithNetwork(network string) *hetznerConfig {
	if b.Networks == nil {
		b.Networks = []providerconfig.ConfigVarString{}
	}

	b.Networks = append(b.Networks, providerconfig.ConfigVarString{Value: network})

	return b
}

func CompleteHetznerProviderSpec(config *hetzner.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecHetzner) (*hetzner.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Hetzner == nil {
		return nil, fmt.Errorf("cannot use cluster to create Hetzner cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &hetzner.RawConfig{}
	}

	if cluster != nil {
		// network configured on the cluster has priority over the datacenter-level network
		if len(config.Networks) == 0 && cluster.Spec.Cloud.Hetzner.Network != "" {
			config.Networks = []providerconfig.ConfigVarString{{
				Value: cluster.Spec.Cloud.Hetzner.Network,
			}}
		}
	}

	if datacenter != nil {
		if config.Datacenter.Value == "" {
			config.Datacenter.Value = datacenter.Datacenter
		}

		if config.Location.Value == "" {
			config.Location.Value = datacenter.Location
		}

		if len(config.Networks) == 0 && datacenter.Network != "" {
			config.Networks = []providerconfig.ConfigVarString{{
				Value: datacenter.Network,
			}}
		}
	}

	return config, nil
}
