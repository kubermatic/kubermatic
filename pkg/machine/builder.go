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

package machine

import (
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/api/v3/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v3/pkg/machine/operatingsystem"

	"k8s.io/apimachinery/pkg/util/sets"
)

type MachineBuilder struct {
	// If a complete cloudProviderSpec is provided,
	// datacenter does not need to be configured at all.

	datacenter *kubermaticv1.Datacenter
	sshPubKeys sets.Set[string]

	cluster             *kubermaticv1.Cluster
	cloudProvider       kubermaticv1.CloudProvider
	cloudProviderSpec   interface{}
	operatingSystemSpec interface{}
	networkConfig       *providerconfig.NetworkConfig
}

func NewBuilder() *MachineBuilder {
	return &MachineBuilder{
		sshPubKeys: sets.New[string](),
	}
}

func (b *MachineBuilder) WithCluster(cluster *kubermaticv1.Cluster) *MachineBuilder {
	b.cluster = cluster
	return b
}

func (b *MachineBuilder) WithDatacenter(datacenter *kubermaticv1.Datacenter) *MachineBuilder {
	b.datacenter = datacenter
	return b
}

func (b *MachineBuilder) WithCloudProvider(cloudProvider kubermaticv1.CloudProvider) *MachineBuilder {
	b.cloudProvider = cloudProvider
	return b
}

func (b *MachineBuilder) WithNetworkConfig(networkConfig *providerconfig.NetworkConfig) *MachineBuilder {
	b.networkConfig = networkConfig
	return b
}

func (b *MachineBuilder) AddSSHPublicKey(pubKeys ...string) *MachineBuilder {
	b.sshPubKeys.Insert(pubKeys...).Delete("") // make sure to not add an empty key by accident
	return b
}

func (b *MachineBuilder) AddSSHKey(key *kubermaticv1.UserSSHKey) *MachineBuilder {
	return b.AddSSHPublicKey(key.Spec.PublicKey)
}

// WithOperatingSystemSpec works great when combined with the convenient [OS]Builder
// helpers in this package.
func (b *MachineBuilder) WithOperatingSystemSpec(osSpec interface{}) *MachineBuilder {
	b.operatingSystemSpec = osSpec
	return b
}

func (b *MachineBuilder) WithCloudProviderSpec(cpSpec interface{}) *MachineBuilder {
	b.cloudProviderSpec = cpSpec
	return b
}

func (b *MachineBuilder) BuildCloudProviderSpec() (interface{}, error) {
	operatingSystem, err := OperatingSystemFromSpec(b.operatingSystemSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to determine operating system: %w", err)
	}

	cloudProvider, err := b.determineCloudProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to determine cloud provider: %w", err)
	}

	return CompleteCloudProviderSpec(b.cloudProviderSpec, cloudProvider, b.cluster, b.datacenter, operatingSystem)
}

func (b *MachineBuilder) BuildProviderConfig() (*providerconfig.Config, error) {
	cloudProvider, err := b.determineCloudProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to determine cloud provider: %w", err)
	}

	operatingSystem, err := OperatingSystemFromSpec(b.operatingSystemSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to determine operating system: %w", err)
	}

	cloudProviderSpec, err := CompleteCloudProviderSpec(b.cloudProviderSpec, cloudProvider, b.cluster, b.datacenter, operatingSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to apply cluster information to the provider spec: %w", err)
	}

	operatingSystemSpec, err := b.completeOperatingSystemSpec(operatingSystem, cloudProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create operating system spec: %w", err)
	}

	networkConfig, err := CompleteNetworkConfig(b.networkConfig, b.cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to apply cluster information to the network config: %w", err)
	}

	return CreateProviderConfig(cloudProvider, cloudProviderSpec, operatingSystemSpec, networkConfig, sets.List(b.sshPubKeys))
}

func (b *MachineBuilder) BuildProviderSpec() (*clusterv1alpha1.ProviderSpec, error) {
	providerConfig, err := b.BuildProviderConfig()
	if err != nil {
		return nil, err
	}

	return CreateProviderSpec(providerConfig)
}

func (b *MachineBuilder) determineCloudProvider() (kubermaticv1.CloudProvider, error) {
	var provider kubermaticv1.CloudProvider

	if b.cluster != nil {
		clusterProvider, err := kubermaticv1helper.ClusterCloudProviderName(b.cluster.Spec.Cloud)
		if err != nil {
			return "", fmt.Errorf("failed to determine cloud provider from cluster: %w", err)
		}

		provider = clusterProvider
	}

	// was an explicit provider given (because the caller did not have a Cluster object at hand)?
	if b.cloudProvider != "" {
		if provider != "" && b.cloudProvider != provider {
			return "", fmt.Errorf("explicit cloud provider %q does not match cluster's cloud provider %q", b.cloudProvider, provider)
		}

		provider = b.cloudProvider
	}

	// all we have is a spec?
	if b.cloudProviderSpec != nil {
		specProvider, err := CloudProviderFromSpec(b.cloudProviderSpec)
		if err != nil {
			return "", fmt.Errorf("cannot determine cloud provider for given spec: %w", err)
		}

		if provider != "" && specProvider != provider {
			return "", fmt.Errorf("cloud provider %q from spec does not match cluster's cloud provider %q", specProvider, provider)
		}

		provider = specProvider
	}

	return provider, nil
}

func (b *MachineBuilder) completeOperatingSystemSpec(os kubermaticv1.OperatingSystem, cloudProvider kubermaticv1.CloudProvider) (interface{}, error) {
	if b.operatingSystemSpec != nil {
		return b.operatingSystemSpec, nil
	}

	return operatingsystem.DefaultSpec(os, cloudProvider)
}
