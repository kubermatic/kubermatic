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
	"k8c.io/machine-controller/sdk/cloudprovider/equinixmetal"
	"k8c.io/machine-controller/sdk/providerconfig"
)

type equinixmetalConfig struct {
	equinixmetal.RawConfig
}

func NewEquinixMetalConfig() *equinixmetalConfig {
	return &equinixmetalConfig{}
}

func (b *equinixmetalConfig) Build() equinixmetal.RawConfig {
	return b.RawConfig
}

func (b *equinixmetalConfig) WithInstanceType(instanceType string) *equinixmetalConfig {
	b.InstanceType.Value = instanceType
	return b
}

func (b *equinixmetalConfig) WithMetro(metro string) *equinixmetalConfig {
	b.Metro.Value = metro
	return b
}

func (b *equinixmetalConfig) WithFacility(facility string) *equinixmetalConfig {
	if b.Facilities == nil {
		b.Facilities = []providerconfig.ConfigVarString{}
	}

	b.Facilities = append(b.Facilities, providerconfig.ConfigVarString{Value: facility})

	return b
}

func (b *equinixmetalConfig) WithProjectID(projectID string) *equinixmetalConfig {
	b.ProjectID.Value = projectID
	return b
}

func (b *equinixmetalConfig) WithBillingCycle(billingCycle string) *equinixmetalConfig {
	b.BillingCycle.Value = billingCycle
	return b
}

func (b *equinixmetalConfig) WithTag(tag string) *equinixmetalConfig {
	b.Tags = addTagToSlice(b.Tags, tag)
	return b
}

//nolint:staticcheck // Deprecated Packet provider is still used for backward compatibility until v2.29
func CompleteEquinixMetalProviderSpec(config *equinixmetal.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecPacket) (*equinixmetal.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Packet == nil {
		return nil, fmt.Errorf("cannot use cluster to create Equinix Metal cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &equinixmetal.RawConfig{}
	}

	tags := []string{"kubernetes"}
	if cluster != nil {
		tags = append(tags,
			fmt.Sprintf("kubernetes-cluster-%s", cluster.Name),
			fmt.Sprintf("system/cluster:%s", cluster.Name),
		)

		if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
			tags = append(tags, fmt.Sprintf("system/project:%s", projectID))
		}
	}
	config.Tags = mergeTags(config.Tags, tags)

	if datacenter != nil {
		if config.Metro.Value == "" {
			config.Metro.Value = datacenter.Metro
		}

		if len(config.Facilities) == 0 {
			if config.Facilities == nil {
				config.Facilities = []providerconfig.ConfigVarString{}
			}

			for _, facility := range datacenter.Facilities {
				config.Facilities = append(config.Facilities, providerconfig.ConfigVarString{Value: facility})
			}
		}
	}

	// TODO: Move this into the validation function
	// if len(config.Facilities) == 0 && config.Metro.Value == "" {
	// 	return nil, errors.New("metro or facilities must be specified")
	// }

	return config, nil
}
