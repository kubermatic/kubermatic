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
	"k8c.io/machine-controller/sdk/cloudprovider/digitalocean"

	"k8s.io/utils/ptr"
)

type digitaloceanConfig struct {
	digitalocean.RawConfig
}

func NewDigitaloceanConfig() *digitaloceanConfig {
	return &digitaloceanConfig{}
}

func (b *digitaloceanConfig) Build() digitalocean.RawConfig {
	return b.RawConfig
}

func (b *digitaloceanConfig) WithRegion(region string) *digitaloceanConfig {
	b.Region.Value = region
	return b
}

func (b *digitaloceanConfig) WithSize(size string) *digitaloceanConfig {
	b.Size.Value = size
	return b
}

func (b *digitaloceanConfig) WithIPv6(enable bool) *digitaloceanConfig {
	b.IPv6.Value = ptr.To(enable)
	return b
}

func (b *digitaloceanConfig) WithPrivateNetworking(enable bool) *digitaloceanConfig {
	b.PrivateNetworking.Value = ptr.To(enable)
	return b
}

func (b *digitaloceanConfig) WithBackups(enable bool) *digitaloceanConfig {
	b.Backups.Value = ptr.To(enable)
	return b
}

func (b *digitaloceanConfig) WithMonitoring(enable bool) *digitaloceanConfig {
	b.Monitoring.Value = ptr.To(enable)
	return b
}

func (b *digitaloceanConfig) WithTag(tag string) *digitaloceanConfig {
	b.Tags = addTagToSlice(b.Tags, tag)
	return b
}

func CompleteDigitaloceanProviderSpec(config *digitalocean.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecDigitalocean) (*digitalocean.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Digitalocean == nil {
		return nil, fmt.Errorf("cannot use cluster to create Digitalocean cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &digitalocean.RawConfig{}
	}

	if datacenter != nil {
		if config.Region.Value == "" {
			config.Region.Value = datacenter.Region
		}
	}

	if config.PrivateNetworking.Value == nil {
		config.PrivateNetworking.Value = ptr.To(true)
	}

	tags := []string{"kubernetes"}
	if cluster != nil {
		tags = append(tags,
			fmt.Sprintf("kubernetes-cluster-%s", cluster.Name),
			fmt.Sprintf("system-cluster-%s", cluster.Name),
		)

		if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
			tags = append(tags, fmt.Sprintf("system-project-%s", projectID))
		}
	}
	config.Tags = mergeTags(config.Tags, tags)

	return config, nil
}
