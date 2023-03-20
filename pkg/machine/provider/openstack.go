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
	"time"

	openstack "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"

	"k8s.io/utils/pointer"
)

type openStackConfig struct {
	openstack.RawConfig
}

func NewOpenStackConfig() *openStackConfig {
	return &openStackConfig{}
}

func (b *openStackConfig) Build() openstack.RawConfig {
	return b.RawConfig
}

func (b *openStackConfig) WithImage(image string) *openStackConfig {
	b.Image.Value = image
	return b
}

func (b *openStackConfig) WithFlavor(flavor string) *openStackConfig {
	b.Flavor.Value = flavor
	return b
}

func (b *openStackConfig) WithRegion(region string) *openStackConfig {
	b.Region.Value = region
	return b
}

func (b *openStackConfig) WithInstanceReadyCheckPeriod(period time.Duration) *openStackConfig {
	b.InstanceReadyCheckPeriod.Value = period.String()
	return b
}

func (b *openStackConfig) WithInstanceReadyCheckTimeout(timeout time.Duration) *openStackConfig {
	b.InstanceReadyCheckTimeout.Value = timeout.String()
	return b
}

func (b *openStackConfig) WithTrustDevicePath(trust bool) *openStackConfig {
	b.TrustDevicePath.Value = pointer.Bool(trust)
	return b
}

func (b *openStackConfig) WithTag(tagKey string, tagValue string) *openStackConfig {
	if b.Tags == nil {
		b.Tags = map[string]string{}
	}
	b.Tags[tagKey] = tagValue
	return b
}

func CompleteOpenStackProviderSpec(config *openstack.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecOpenStack, os kubermaticv1.OperatingSystem) (*openstack.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.OpenStack == nil {
		return nil, fmt.Errorf("cannot use cluster to create Openstack cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &openstack.RawConfig{}
	}

	if datacenter != nil {
		if config.Image.Value == "" && os != "" {
			image, ok := datacenter.Images[os]
			if !ok {
				return nil, fmt.Errorf("no disk image configured for operating system %q", os)
			}

			config.Image.Value = image
		}

		if config.AvailabilityZone.Value == "" {
			config.AvailabilityZone.Value = datacenter.AvailabilityZone
		}

		if config.Region.Value == "" {
			config.Region.Value = datacenter.Region
		}

		if config.IdentityEndpoint.Value == "" {
			config.IdentityEndpoint.Value = datacenter.AuthURL
		}

		if config.TrustDevicePath.Value == nil {
			config.TrustDevicePath.Value = datacenter.TrustDevicePath

			if config.TrustDevicePath.Value == nil {
				config.TrustDevicePath.Value = pointer.Bool(false)
			}
		}
	}

	if cluster != nil {
		if config.FloatingIPPool.Value == "" {
			config.FloatingIPPool.Value = cluster.Spec.Cloud.OpenStack.FloatingIPPool
		}

		if config.Network.Value == "" {
			config.Network.Value = cluster.Spec.Cloud.OpenStack.Network
		}

		if config.Subnet.Value == "" {
			config.Subnet.Value = cluster.Spec.Cloud.OpenStack.SubnetID
		}

		if len(config.SecurityGroups) == 0 && len(cluster.Spec.Cloud.OpenStack.SecurityGroups) > 0 {
			config.SecurityGroups = []providerconfig.ConfigVarString{{Value: cluster.Spec.Cloud.OpenStack.SecurityGroups}}
		}

		if config.Tags == nil {
			config.Tags = map[string]string{}
		}

		config.Tags["kubernetes-cluster"] = cluster.Name
		config.Tags["system-cluster"] = cluster.Name

		if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
			config.Tags["system-project"] = projectID
		}
	}

	return config, nil
}
