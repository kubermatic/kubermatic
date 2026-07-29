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
	"net/url"
	"strconv"
	"strings"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	"k8c.io/machine-controller/sdk/providerconfig"
)

type kubevirtConfig struct {
	kubevirt.RawConfig
}

func NewKubevirtConfig() *kubevirtConfig {
	return &kubevirtConfig{}
}

func (b *kubevirtConfig) Build() kubevirt.RawConfig {
	return b.RawConfig
}

func (b *kubevirtConfig) WithCPUs(cpus int) *kubevirtConfig {
	b.VirtualMachine.Template.CPUs.Value = strconv.Itoa(cpus)
	return b
}

func (b *kubevirtConfig) WithVCPUs(vCPUs int) *kubevirtConfig {
	b.VirtualMachine.Template.VCPUs.Cores = vCPUs
	return b
}

func (b *kubevirtConfig) WithMemory(memory string) *kubevirtConfig {
	b.VirtualMachine.Template.Memory.Value = memory
	return b
}

func (b *kubevirtConfig) WithPrimaryDiskOSImage(image string) *kubevirtConfig {
	b.VirtualMachine.Template.PrimaryDisk.OsImage.Value = image
	return b
}

func (b *kubevirtConfig) WithPrimaryDiskSize(size string) *kubevirtConfig {
	b.VirtualMachine.Template.PrimaryDisk.Size.Value = size
	return b
}

func (b *kubevirtConfig) WithPrimaryDiskStorageClassName(className string) *kubevirtConfig {
	b.VirtualMachine.Template.PrimaryDisk.StorageClassName.Value = className
	return b
}

func (b *kubevirtConfig) WithDNSPolicy(dnsPolicy string) *kubevirtConfig {
	b.VirtualMachine.DNSPolicy.Value = dnsPolicy
	return b
}

func (b *kubevirtConfig) WithClusterName(clusterName string) *kubevirtConfig {
	b.ClusterName = providerconfig.ConfigVarString{Value: clusterName}
	return b
}

func (b *kubevirtConfig) WithInstancetype(name string) *kubevirtConfig {
	b.VirtualMachine.Instancetype = &kubevirtcorev1.InstancetypeMatcher{Name: name}
	return b
}

func CompleteKubevirtProviderSpec(config *kubevirt.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecKubevirt) (*kubevirt.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Kubevirt == nil {
		return nil, fmt.Errorf("cannot use cluster to create Kubevirt cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &kubevirt.RawConfig{}
	}

	if datacenter != nil {
		if config.VirtualMachine.DNSPolicy.Value == "" {
			config.VirtualMachine.DNSPolicy.Value = datacenter.DNSPolicy
		}

		if config.VirtualMachine.DNSConfig == nil {
			config.VirtualMachine.DNSConfig = datacenter.DNSConfig.DeepCopy()
		}

		if config.VirtualMachine.EvictionStrategy == "" {
			config.VirtualMachine.EvictionStrategy = string(datacenter.VMEvictionStrategy)
		}

		// Instance types already determine CPU and memory, so the defaults only apply
		// when no instance type is selected.
		if datacenter.NodeDefaults != nil && config.VirtualMachine.Instancetype == nil {
			if config.VirtualMachine.Template.CPUs.Value == "" && config.VirtualMachine.Template.VCPUs.Cores == 0 {
				config.VirtualMachine.Template.CPUs.Value = datacenter.NodeDefaults.CPUs
			}

			if config.VirtualMachine.Template.Memory.Value == "" {
				config.VirtualMachine.Template.Memory.Value = datacenter.NodeDefaults.Memory
			}
		}

		if datacenter.NodeDefaults != nil && config.VirtualMachine.Template.PrimaryDisk.Size.Value == "" {
			config.VirtualMachine.Template.PrimaryDisk.Size.Value = datacenter.NodeDefaults.PrimaryDiskSize
		}
	}

	if cluster != nil {
		kubeVirtInfraNamespace := cluster.Status.NamespaceName
		if datacenter != nil && datacenter.NamespacedMode != nil && datacenter.NamespacedMode.Enabled {
			kubeVirtInfraNamespace = datacenter.NamespacedMode.Namespace
		}
		config.ClusterName = providerconfig.ConfigVarString{Value: cluster.Name}
		config.VirtualMachine.Template.PrimaryDisk.OsImage.Value = extractKubeVirtOsImageURLOrDataVolumeNsName(kubeVirtInfraNamespace, config.VirtualMachine.Template.PrimaryDisk.OsImage.Value)
	}

	return config, nil
}

func extractKubeVirtOsImageURLOrDataVolumeNsName(namespace string, osImage string) string {
	// config.VirtualMachine.Template.PrimaryDisk.OsImage.Value contains:
	// - a URL
	// - or a DataVolume name
	// If config.VirtualMachine.Template.PrimaryDisk.OsImage.Value is a DataVolume, we need to add the namespace prefix
	if _, err := url.ParseRequestURI(osImage); err == nil {
		return osImage
	}
	// It's a DataVolume
	// If it's already a ns/name keep it.
	if nameSpaceAndName := strings.Split(osImage, "/"); len(nameSpaceAndName) >= 2 {
		return osImage
	}
	return fmt.Sprintf("%s/%s", namespace, osImage)
}
