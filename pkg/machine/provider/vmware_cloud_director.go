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
	"k8c.io/machine-controller/sdk/cloudprovider/vmwareclouddirector"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/utils/ptr"
)

type vcdConfig struct {
	vmwareclouddirector.RawConfig
}

func NewVMwareCloudDirectorConfig() *vcdConfig {
	return &vcdConfig{}
}

func (b *vcdConfig) Build() vmwareclouddirector.RawConfig {
	return b.RawConfig
}

func (b *vcdConfig) WithCatalog(catalog string) *vcdConfig {
	b.Catalog.Value = catalog
	return b
}

func (b *vcdConfig) WithTemplate(template string) *vcdConfig {
	b.Template.Value = template
	return b
}

func (b *vcdConfig) WithCPUs(cpus int) *vcdConfig {
	b.CPUs = int64(cpus)
	return b
}

func (b *vcdConfig) WithCPUCores(cpuCores int) *vcdConfig {
	b.CPUCores = int64(cpuCores)
	return b
}

func (b *vcdConfig) WithMemoryMB(memoryMB int) *vcdConfig {
	b.MemoryMB = int64(memoryMB)
	return b
}

func (b *vcdConfig) WithDiskSizeGB(diskSizeGB int) *vcdConfig {
	b.DiskSizeGB = ptr.To[int64](int64(diskSizeGB))
	return b
}

func (b *vcdConfig) WithIPAllocationMode(mode vmwareclouddirector.IPAllocationMode) *vcdConfig {
	b.IPAllocationMode = mode
	return b
}

func (b *vcdConfig) WithAllowInsecure(allow bool) *vcdConfig {
	b.AllowInsecure.Value = ptr.To(allow)
	return b
}

func CompleteVMwareCloudDirectorProviderSpec(config *vmwareclouddirector.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecVMwareCloudDirector, os providerconfig.OperatingSystem) (*vmwareclouddirector.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.VMwareCloudDirector == nil {
		return nil, fmt.Errorf("cannot use cluster to create VMware Cloud Director cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &vmwareclouddirector.RawConfig{}
	}

	if datacenter != nil {
		if config.AllowInsecure.Value == nil {
			config.AllowInsecure.Value = ptr.To(datacenter.AllowInsecure)
		}

		if config.Catalog.Value == "" {
			config.Catalog.Value = datacenter.DefaultCatalog
		}

		if config.StorageProfile == nil && len(datacenter.DefaultStorageProfile) > 0 {
			config.StorageProfile = ptr.To(datacenter.DefaultStorageProfile)
		}

		if config.Template.Value == "" && os != "" {
			template := datacenter.Templates[os]
			if template == "" {
				return nil, fmt.Errorf("no template configured in VMware Cloud Director datacenter for operating system %q", os)
			}

			config.Template.Value = template
		}
	}

	if cluster != nil {
		if config.VApp.Value == "" {
			config.VApp.Value = cluster.Spec.Cloud.VMwareCloudDirector.VApp
		}

		if config.Network.Value == "" {
			if len(cluster.Spec.Cloud.VMwareCloudDirector.OVDCNetworks) > 0 {
				// As a default, we attach the first network to the VMs.
				config.Network.Value = cluster.Spec.Cloud.VMwareCloudDirector.OVDCNetworks[0]
			} else if cluster.Spec.Cloud.VMwareCloudDirector.OVDCNetwork != "" {
				config.Network.Value = cluster.Spec.Cloud.VMwareCloudDirector.OVDCNetwork
			}
		}
	}

	return config, nil
}
