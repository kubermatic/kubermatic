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
	"k8c.io/machine-controller/sdk/cloudprovider/vsphere"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/utils/ptr"
)

type vsphereConfig struct {
	vsphere.RawConfig
}

func NewVSphereConfig() *vsphereConfig {
	return &vsphereConfig{}
}

func (b *vsphereConfig) Build() vsphere.RawConfig {
	return b.RawConfig
}

func (b *vsphereConfig) WithCPUs(cpus int) *vsphereConfig {
	b.CPUs = int32(cpus)
	return b
}

func (b *vsphereConfig) WithMemoryMB(memory int) *vsphereConfig {
	b.MemoryMB = int64(memory)
	return b
}

func (b *vsphereConfig) WithDiskSizeGB(diskSize int) *vsphereConfig {
	b.DiskSizeGB = ptr.To[int64](int64(diskSize))
	return b
}

func (b *vsphereConfig) WithDatacenter(dc string) *vsphereConfig {
	b.Datacenter.Value = dc
	return b
}

func (b *vsphereConfig) WithDatastore(ds string) *vsphereConfig {
	b.Datastore.Value = ds
	return b
}

func (b *vsphereConfig) WithFolder(folder string) *vsphereConfig {
	b.Folder.Value = folder
	return b
}

func (b *vsphereConfig) WithTemplateVMName(templateName string) *vsphereConfig {
	b.TemplateVMName.Value = templateName
	return b
}

func (b *vsphereConfig) WithAllowInsecure(allow bool) *vsphereConfig {
	b.AllowInsecure.Value = ptr.To(allow)
	return b
}

func CompleteVSphereProviderSpec(config *vsphere.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecVSphere, os providerconfig.OperatingSystem) (*vsphere.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.VSphere == nil {
		return nil, fmt.Errorf("cannot use cluster to create VSphere cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &vsphere.RawConfig{}
	}

	var datastore = ""
	// If `DatastoreCluster` is not specified we use either the Datastore
	// specified at `Cluster` or the one specified at `Datacenter` level.
	if cluster != nil {
		if cluster.Spec.Cloud.VSphere.DatastoreCluster == "" {
			datastore = cluster.Spec.Cloud.VSphere.Datastore
			if datastore == "" && datacenter != nil {
				datastore = datacenter.DefaultDatastore
			}
		}
	}

	if config.Datastore.Value == "" {
		config.Datastore.Value = datastore
	}

	if datacenter != nil {
		if config.Datacenter.Value == "" {
			config.Datacenter.Value = datacenter.Datacenter
		}

		if config.AllowInsecure.Value == nil {
			config.AllowInsecure.Value = ptr.To(datacenter.AllowInsecure)
		}

		if config.Cluster.Value == "" {
			config.Cluster.Value = datacenter.Cluster
		}

		if config.TemplateVMName.Value == "" && os != "" {
			templaate, ok := datacenter.Templates[os]
			if !ok {
				return nil, fmt.Errorf("no template VM configured for operating system %q", os)
			}

			config.TemplateVMName.Value = templaate
		}
	}

	if cluster != nil {
		//nolint:staticcheck
		//lint:ignore SA1019: config.VMNetName is deprecated: use networks instead.
		if config.VMNetName.Value == "" && config.Networks == nil {
			// Both Networks and VMNetName can't exist at the same time.
			if len(cluster.Spec.Cloud.VSphere.Networks) > 0 {
				for _, network := range cluster.Spec.Cloud.VSphere.Networks {
					config.Networks = append(config.Networks, providerconfig.ConfigVarString{Value: network})
				}
			} else if cluster.Spec.Cloud.VSphere.VMNetName != "" {
				config.Networks = append(config.Networks, providerconfig.ConfigVarString{Value: cluster.Spec.Cloud.VSphere.VMNetName})
			}
		}

		if config.DatastoreCluster.Value == "" {
			config.DatastoreCluster.Value = cluster.Spec.Cloud.VSphere.DatastoreCluster
		}

		if config.Folder.Value == "" {
			config.Folder.Value = cluster.Spec.Cloud.VSphere.Folder

			if config.Folder.Value == "" && datacenter != nil {
				config.Folder.Value = fmt.Sprintf("%s/%s", datacenter.RootPath, cluster.Name)
			}
		}

		if config.ResourcePool.Value == "" {
			config.ResourcePool.Value = cluster.Spec.Cloud.VSphere.ResourcePool
		}

		for i, tag := range config.Tags {
			if tag.CategoryID == "" {
				config.Tags[i].CategoryID = cluster.Spec.Cloud.VSphere.Tags.CategoryID
			}
		}
	}

	return config, nil
}
