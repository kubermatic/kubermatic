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
	nutanixprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/nutanix"
	"k8c.io/machine-controller/sdk/cloudprovider/nutanix"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/utils/ptr"
)

type nutanixConfig struct {
	nutanix.RawConfig
}

func NewNutanixConfig() *nutanixConfig {
	return &nutanixConfig{}
}

func (b *nutanixConfig) Build() nutanix.RawConfig {
	return b.RawConfig
}

func (b *nutanixConfig) WithClusterName(clusterName string) *nutanixConfig {
	b.ClusterName.Value = clusterName
	return b
}

func (b *nutanixConfig) WithProjectName(projectName string) *nutanixConfig {
	b.ProjectName = &providerconfig.ConfigVarString{Value: projectName}
	return b
}

func (b *nutanixConfig) WithSubnetName(subnetName string) *nutanixConfig {
	b.SubnetName.Value = subnetName
	return b
}

func (b *nutanixConfig) WithImageName(imageName string) *nutanixConfig {
	b.ImageName.Value = imageName
	return b
}

func (b *nutanixConfig) WithCPUs(cpus int) *nutanixConfig {
	b.CPUs = int64(cpus)
	return b
}

func (b *nutanixConfig) WithMemoryMB(memoryMB int) *nutanixConfig {
	b.MemoryMB = int64(memoryMB)
	return b
}

func (b *nutanixConfig) WithDiskSize(diskSize int) *nutanixConfig {
	b.DiskSize = ptr.To[int64](int64(diskSize))
	return b
}

func (b *nutanixConfig) WithAllowInsecure(allow bool) *nutanixConfig {
	b.AllowInsecure.Value = ptr.To(allow)
	return b
}

func (b *nutanixConfig) WithCategory(catKey string, catValue string) *nutanixConfig {
	if b.Categories == nil {
		b.Categories = map[string]string{}
	}
	b.Categories[catKey] = catValue
	return b
}

func CompleteNutanixProviderSpec(config *nutanix.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecNutanix, os providerconfig.OperatingSystem) (*nutanix.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Nutanix == nil {
		return nil, fmt.Errorf("cannot use cluster to create Nutanix cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &nutanix.RawConfig{}
	}

	if cluster != nil {
		if config.ProjectName == nil || config.ProjectName.Value == "" {
			// Nutanix does not like us specifying the default project, so if the cluster uses that one, we have to omit it from the provider spec
			if cluster.Spec.Cloud.Nutanix.ProjectName != "" && cluster.Spec.Cloud.Nutanix.ProjectName != nutanixprovider.DefaultProject {
				config.ProjectName = &providerconfig.ConfigVarString{Value: cluster.Spec.Cloud.Nutanix.ProjectName}
			}
		}

		if config.Categories == nil {
			config.Categories = map[string]string{}
		}

		config.Categories[nutanixprovider.ClusterCategoryName] = nutanixprovider.CategoryValue(cluster.Name)

		if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
			config.Categories[nutanixprovider.ProjectCategoryName] = projectID
		}
	}

	if datacenter != nil {
		if config.AllowInsecure.Value == nil {
			config.AllowInsecure.Value = &datacenter.AllowInsecure
		}

		if config.ImageName.Value == "" && os != "" {
			image, ok := datacenter.Images[os]
			if !ok {
				return nil, fmt.Errorf("no disk image configured for operating system %q", os)
			}

			config.ImageName.Value = image
		}
	}

	return config, nil
}
