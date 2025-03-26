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
	"k8c.io/machine-controller/sdk/cloudprovider/azure"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/utils/ptr"
)

type azureConfig struct {
	azure.RawConfig
}

func NewAzureConfig() *azureConfig {
	return &azureConfig{}
}

func (b *azureConfig) Build() azure.RawConfig {
	return b.RawConfig
}

func (b *azureConfig) WithAssignPublicIP(assign bool) *azureConfig {
	b.AssignPublicIP.Value = ptr.To(assign)
	return b
}

func (b *azureConfig) WithLocation(location string) *azureConfig {
	b.Location.Value = location
	return b
}

func (b *azureConfig) WithResourceGroup(resourceGroup string) *azureConfig {
	b.ResourceGroup.Value = resourceGroup
	return b
}

func (b *azureConfig) WithVNetResourceGroup(vnetResourceGroup string) *azureConfig {
	b.VNetResourceGroup.Value = vnetResourceGroup
	return b
}

func (b *azureConfig) WithVMSize(vmSize string) *azureConfig {
	b.VMSize.Value = vmSize
	return b
}

func (b *azureConfig) WithVNetName(vNetName string) *azureConfig {
	b.VNetName.Value = vNetName
	return b
}

func (b *azureConfig) WithSubnetName(subnetName string) *azureConfig {
	b.SubnetName.Value = subnetName
	return b
}

func (b *azureConfig) WithLoadBalancerSku(soadBalancerSku string) *azureConfig {
	b.LoadBalancerSku.Value = soadBalancerSku
	return b
}

func (b *azureConfig) WithRouteTableName(routeTableName string) *azureConfig {
	b.RouteTableName.Value = routeTableName
	return b
}

func (b *azureConfig) WithAvailabilitySet(availabilitySet string) *azureConfig {
	b.AvailabilitySet.Value = availabilitySet
	return b
}

func (b *azureConfig) WithSecurityGroupName(securityGroupName string) *azureConfig {
	b.SecurityGroupName.Value = securityGroupName
	return b
}

func (b *azureConfig) WithTag(tagKey string, tagValue string) *azureConfig {
	if b.Tags == nil {
		b.Tags = map[string]string{}
	}
	b.Tags[tagKey] = tagValue
	return b
}

func CompleteAzureProviderSpec(config *azure.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecAzure, os providerconfig.OperatingSystem) (*azure.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Azure == nil {
		return nil, fmt.Errorf("cannot use cluster to create Azure cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &azure.RawConfig{}
	}

	if datacenter != nil {
		if config.Location.Value == "" {
			config.Location.Value = datacenter.Location
		}

		if config.ImageID.Value == "" && os != "" {
			// This can still be empty, but that's okay, the machine-controller will later, during
			// reconciliations, default the image for us.
			config.ImageID.Value = datacenter.Images[os]
		}
	}

	if cluster != nil {
		if config.AssignAvailabilitySet == nil {
			config.AssignAvailabilitySet = cluster.Spec.Cloud.Azure.AssignAvailabilitySet
		}

		if config.AvailabilitySet.Value == "" {
			config.AvailabilitySet.Value = cluster.Spec.Cloud.Azure.AvailabilitySet
		}

		if config.ResourceGroup.Value == "" {
			config.ResourceGroup.Value = cluster.Spec.Cloud.Azure.ResourceGroup
		}

		if config.VNetResourceGroup.Value == "" {
			config.VNetResourceGroup.Value = cluster.Spec.Cloud.Azure.VNetResourceGroup
		}

		if config.VNetName.Value == "" {
			config.VNetName.Value = cluster.Spec.Cloud.Azure.VNetName
		}

		if config.SubnetName.Value == "" {
			config.SubnetName.Value = cluster.Spec.Cloud.Azure.SubnetName
		}

		if config.RouteTableName.Value == "" {
			config.RouteTableName.Value = cluster.Spec.Cloud.Azure.RouteTableName
		}

		if config.SecurityGroupName.Value == "" {
			config.SecurityGroupName.Value = cluster.Spec.Cloud.Azure.SecurityGroup
		}

		if config.LoadBalancerSku.Value == "" {
			config.LoadBalancerSku.Value = string(cluster.Spec.Cloud.Azure.LoadBalancerSKU)
		}

		assignPublicIP := config.AssignPublicIP.Value != nil && *config.AssignPublicIP.Value
		if assignPublicIP && config.LoadBalancerSku.Value == string(kubermaticv1.AzureStandardLBSKU) {
			config.PublicIPSKU = ptr.To(config.LoadBalancerSku.Value)
		}

		if config.Tags == nil {
			config.Tags = map[string]string{}
		}

		config.Tags["KubernetesCluster"] = cluster.Name
		config.Tags["system-cluster"] = cluster.Name

		if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
			config.Tags["system-project"] = projectID
		}
	}

	return config, nil
}
