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
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/machine/operatingsystem"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
)

type MachineBuilder struct {
	// specify either seed+datacenterName or the datacenter directly;
	// when the datacenter is given, it takes priority and the caller
	// is responsible for making sure it's the correct datacenter
	// matching the cluster (if any).
	// If a complete cloudProviderSpec is provided, neither seed nor
	// datacenter need to be configured at all.

	seed           *kubermaticv1.Seed
	datacenterName string
	sshPubKeys     sets.Set[string]

	datacenter *kubermaticv1.Datacenter

	cluster             *kubermaticv1.Cluster
	cloudProvider       kubermaticv1.ProviderType
	cloudProviderSpec   interface{}
	operatingSystemSpec interface{}
	networkConfig       *providerconfig.NetworkConfig
}

func NewBuilder() *MachineBuilder {
	return &MachineBuilder{
		sshPubKeys: sets.New[string](),
	}
}

// WithSeed should only be used in conjunction with WithDatacenterName().
// Alternatively, use WithDatacenter() to specify the datacenter directly.
func (b *MachineBuilder) WithSeed(seed *kubermaticv1.Seed) *MachineBuilder {
	b.seed = seed
	return b
}

func (b *MachineBuilder) WithCluster(cluster *kubermaticv1.Cluster) *MachineBuilder {
	b.cluster = cluster
	return b
}

func (b *MachineBuilder) WithDatacenter(datacenter *kubermaticv1.Datacenter) *MachineBuilder {
	b.datacenter = datacenter
	return b
}

func (b *MachineBuilder) WithDatacenterName(datacenterName string) *MachineBuilder {
	b.datacenterName = datacenterName
	return b
}

func (b *MachineBuilder) WithCloudProvider(cloudProvider kubermaticv1.ProviderType) *MachineBuilder {
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

	// try to determine the target datacenter (it's optional as the caller is free to provide
	// a full providerspec themselves)
	datacenter, err := b.determineDatacenter()
	if err != nil {
		return nil, fmt.Errorf("failed to determine datacenter: %w", err)
	}

	return CompleteCloudProviderSpec(b.cloudProviderSpec, cloudProvider, b.cluster, datacenter, operatingSystem)
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

	datacenter, err := b.determineDatacenter()
	if err != nil {
		return nil, fmt.Errorf("failed to determine datacenter: %w", err)
	}

	cloudProviderSpec, err := CompleteCloudProviderSpec(b.cloudProviderSpec, cloudProvider, b.cluster, datacenter, operatingSystem)
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

func (b *MachineBuilder) determineCloudProvider() (kubermaticv1.ProviderType, error) {
	var provider kubermaticv1.ProviderType

	if b.cluster != nil {
		clusterProvider, err := kubermaticv1helper.ClusterCloudProviderName(b.cluster.Spec.Cloud)
		if err != nil {
			return "", fmt.Errorf("failed to determine cloud provider from cluster: %w", err)
		}

		provider = kubermaticv1.ProviderType(clusterProvider)
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
		specProvider, err := ProviderTypeFromSpec(b.cloudProviderSpec)
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

func (b *MachineBuilder) determineDatacenter() (*kubermaticv1.Datacenter, error) {
	if b.datacenter != nil {
		return b.datacenter, nil
	}

	// sanity check: do not silently ignore misconfigurations where only a seed, but
	// no datacenter is specified accidentally; note that the datacenter name can be
	// taken from the cluster, if given
	seedSpecified := b.seed != nil
	dcSpecified := b.datacenterName != "" || (b.cluster != nil && b.cluster.Spec.Cloud.DatacenterName != "")

	if seedSpecified != dcSpecified {
		return nil, errors.New("invalid builder configuration, must specify either both seed and datacenter name or nothing at all; datacenter name can be deduced from a Cluster object")
	}

	// it's valid to not specify if the cloud provider spec needs no further fields added to it
	if !seedSpecified && !dcSpecified {
		return nil, nil
	}

	// sanity check: make sure cluster and explicit datacenter name do not contradict each other
	if b.datacenterName != "" && b.cluster != nil && b.cluster.Spec.Cloud.DatacenterName != b.datacenterName {
		return nil, fmt.Errorf("explicit datacenter %q does not match cluster's datacenter %q", b.datacenterName, b.cluster.Spec.Cloud.DatacenterName)
	}

	datacenterName := b.datacenterName
	if datacenterName == "" && b.cluster != nil {
		datacenterName = b.cluster.Spec.Cloud.DatacenterName
	}
	// due to the checks for dcSpecified it's now guaranteed that datacenterName is not empty

	for name, dc := range b.seed.Spec.Datacenters {
		if name == datacenterName {
			return &dc, nil
		}
	}

	return nil, fmt.Errorf("unknown datacenter %q in seed %q", datacenterName, b.seed.Name)
}

func (b *MachineBuilder) completeOperatingSystemSpec(os providerconfig.OperatingSystem, cloudProvider kubermaticv1.ProviderType) (interface{}, error) {
	if b.operatingSystemSpec != nil {
		return b.operatingSystemSpec, nil
	}

	return operatingsystem.DefaultSpec(os, cloudProvider)
}
