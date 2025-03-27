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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/openstack"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/utils/ptr"
)

type openstackConfig struct {
	openstack.RawConfig
}

func NewOpenstackConfig() *openstackConfig {
	return &openstackConfig{}
}

func (b *openstackConfig) Build() openstack.RawConfig {
	return b.RawConfig
}

func (b *openstackConfig) WithImage(image string) *openstackConfig {
	b.Image.Value = image
	return b
}

func (b *openstackConfig) WithFlavor(flavor string) *openstackConfig {
	b.Flavor.Value = flavor
	return b
}

func (b *openstackConfig) WithRegion(region string) *openstackConfig {
	b.Region.Value = region
	return b
}

func (b *openstackConfig) WithInstanceReadyCheckPeriod(period time.Duration) *openstackConfig {
	b.InstanceReadyCheckPeriod.Value = period.String()
	return b
}

func (b *openstackConfig) WithInstanceReadyCheckTimeout(timeout time.Duration) *openstackConfig {
	b.InstanceReadyCheckTimeout.Value = timeout.String()
	return b
}

func (b *openstackConfig) WithTrustDevicePath(trust bool) *openstackConfig {
	b.TrustDevicePath.Value = ptr.To(trust)
	return b
}

func (b *openstackConfig) WithTag(tagKey string, tagValue string) *openstackConfig {
	if b.Tags == nil {
		b.Tags = map[string]string{}
	}
	b.Tags[tagKey] = tagValue
	return b
}

func CompleteOpenstackProviderSpec(config *openstack.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecOpenstack, os providerconfig.OperatingSystem) (*openstack.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Openstack == nil {
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
				config.TrustDevicePath.Value = ptr.To(false)
			}
		}

		if config.ConfigDrive.Value == nil && datacenter.EnableConfigDrive != nil {
			config.ConfigDrive.Value = datacenter.EnableConfigDrive
		}
	}

	if cluster != nil {
		if config.Network.Value == "" {
			config.Network.Value = cluster.Spec.Cloud.Openstack.Network
		}

		if config.Subnet.Value == "" {
			config.Subnet.Value = cluster.Spec.Cloud.Openstack.SubnetID
		}

		if len(config.SecurityGroups) == 0 && len(cluster.Spec.Cloud.Openstack.SecurityGroups) > 0 {
			config.SecurityGroups = []providerconfig.ConfigVarString{{Value: cluster.Spec.Cloud.Openstack.SecurityGroups}}
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
