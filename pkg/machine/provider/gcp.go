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
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/gce"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

type gcpConfig struct {
	gce.RawConfig
}

func NewGCPConfig() *gcpConfig {
	return &gcpConfig{}
}

func (b *gcpConfig) Build() gce.RawConfig {
	return b.RawConfig
}

func (b *gcpConfig) WithZone(zone string) *gcpConfig {
	b.Zone.Value = zone
	return b
}

func (b *gcpConfig) WithMachineType(machineType string) *gcpConfig {
	b.MachineType.Value = machineType
	return b
}

func (b *gcpConfig) WithDiskSize(diskSize int) *gcpConfig {
	b.DiskSize = int64(diskSize)
	return b
}

func (b *gcpConfig) WithDiskType(diskType string) *gcpConfig {
	b.DiskType.Value = diskType
	return b
}

func (b *gcpConfig) WithNetwork(network string) *gcpConfig {
	b.Network.Value = network
	return b
}

func (b *gcpConfig) WithPreemptible(preemptible bool) *gcpConfig {
	b.Preemptible.Value = ptr.To(preemptible)
	return b
}

func (b *gcpConfig) WithRegional(regional bool) *gcpConfig {
	b.Regional.Value = ptr.To(regional)
	return b
}

func (b *gcpConfig) WithMultiZone(multiZone bool) *gcpConfig {
	b.MultiZone.Value = ptr.To(multiZone)
	return b
}

func (b *gcpConfig) WithAssignPublicIPAddress(assign bool) *gcpConfig {
	if b.AssignPublicIPAddress == nil {
		b.AssignPublicIPAddress = &providerconfig.ConfigVarBool{}
	}
	b.AssignPublicIPAddress.Value = ptr.To(assign)
	return b
}

func (b *gcpConfig) WithTag(tag string) *gcpConfig {
	b.Tags = sets.List(sets.New(b.Tags...).Insert(tag))
	return b
}

func CompleteGCPProviderSpec(config *gce.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecGCP) (*gce.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.GCP == nil {
		return nil, fmt.Errorf("cannot use cluster to create GCP cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &gce.RawConfig{}
	}

	if config.AssignPublicIPAddress == nil || config.AssignPublicIPAddress.Value == nil {
		config.AssignPublicIPAddress = &providerconfig.ConfigVarBool{Value: ptr.To(true)}
	}

	if config.MultiZone.Value == nil {
		config.MultiZone.Value = ptr.To(false)
	}

	if config.Regional.Value == nil {
		config.Regional.Value = ptr.To(false)
	}

	if cluster != nil {
		if config.Network.Value == "" {
			config.Network.Value = cluster.Spec.Cloud.GCP.Network
		}

		if config.Subnetwork.Value == "" {
			config.Subnetwork.Value = cluster.Spec.Cloud.GCP.Subnetwork
		}

		tags := sets.New(config.Tags...)
		tags.Insert(
			fmt.Sprintf("kubernetes-cluster-%s", cluster.Name),
			fmt.Sprintf("system-cluster-%s", cluster.Name),
		)

		if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
			tags.Insert(fmt.Sprintf("system-project-%s", projectID))
		}

		config.Tags = sets.List(tags)
	}

	if datacenter != nil {
		if config.Regional.Value == nil {
			config.Regional.Value = &datacenter.Regional
		}

		if config.Zone.Value == "" {
			if datacenter.Region == "" {
				return nil, errors.New("no region configured in GCP datacenter, cannot construct zone")
			}

			if len(datacenter.ZoneSuffixes) == 0 {
				return nil, errors.New("no zone suffixes configured in GCP datacenter, cannot construct zone")
			}

			config.Zone.Value = datacenter.Region + "-" + datacenter.ZoneSuffixes[0]
		}
	}

	return config, nil
}
