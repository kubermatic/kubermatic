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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	alibaba "k8c.io/machine-controller/sdk/cloudprovider/alibaba"
	anexia "k8c.io/machine-controller/sdk/cloudprovider/anexia"
	aws "k8c.io/machine-controller/sdk/cloudprovider/aws"
	azure "k8c.io/machine-controller/sdk/cloudprovider/azure"
	baremetal "k8c.io/machine-controller/sdk/cloudprovider/baremetal"
	digitalocean "k8c.io/machine-controller/sdk/cloudprovider/digitalocean"
	gce "k8c.io/machine-controller/sdk/cloudprovider/gce"
	hetzner "k8c.io/machine-controller/sdk/cloudprovider/hetzner"
	kubevirt "k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	nutanix "k8c.io/machine-controller/sdk/cloudprovider/nutanix"
	openstack "k8c.io/machine-controller/sdk/cloudprovider/openstack"
	vmwareclouddirector "k8c.io/machine-controller/sdk/cloudprovider/vmwareclouddirector"
	vsphere "k8c.io/machine-controller/sdk/cloudprovider/vsphere"
	"k8c.io/machine-controller/sdk/net"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/operating-system-manager/pkg/providerconfig/amzn2"
	"k8c.io/operating-system-manager/pkg/providerconfig/flatcar"
	"k8c.io/operating-system-manager/pkg/providerconfig/rhel"
	"k8c.io/operating-system-manager/pkg/providerconfig/rockylinux"
	"k8c.io/operating-system-manager/pkg/providerconfig/ubuntu"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
)

func EncodeAsRawExtension(value interface{}) (runtime.RawExtension, error) {
	ext := runtime.RawExtension{}

	b, err := json.Marshal(value)
	if err != nil {
		return ext, err
	}

	ext.Raw = b
	return ext, nil
}

func CreateProviderConfig(cloudProvider kubermaticv1.ProviderType, cloudProviderSpec interface{}, osSpec interface{}, networkConfig *providerconfig.NetworkConfig, sshPubKeys []string) (*providerconfig.Config, error) {
	mcCloudProvider, err := MachineControllerProviderName(cloudProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to determine cloud provider from cluster: %w", err)
	}

	operatingSystem, err := OperatingSystemFromSpec(osSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to determine operating system: %w", err)
	}

	cloudProviderSpecExt, err := EncodeAsRawExtension(cloudProviderSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to encode cloud provider spec: %w", err)
	}

	osSpecExt, err := EncodeAsRawExtension(osSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to encode operating system spec: %w", err)
	}

	return &providerconfig.Config{
		CloudProvider:       mcCloudProvider,
		CloudProviderSpec:   cloudProviderSpecExt,
		OperatingSystem:     operatingSystem,
		OperatingSystemSpec: osSpecExt,
		SSHPublicKeys:       sshPubKeys,
		Network:             networkConfig,
	}, nil
}

func CreateProviderSpec(providerConfig *providerconfig.Config) (*clusterv1alpha1.ProviderSpec, error) {
	encodedConfig, err := EncodeAsRawExtension(providerConfig)
	if err != nil {
		return nil, err
	}

	return &clusterv1alpha1.ProviderSpec{
		Value: &encodedConfig,
	}, nil
}

// MachineControllerProviderName translates the KKP cloud provider name into
// the machine-controller's name. Most providers are named identically, but some
// are different (like gcp vs. gce).
func MachineControllerProviderName(kkpName kubermaticv1.ProviderType) (providerconfig.CloudProvider, error) {
	provider := providerconfig.CloudProvider(kkpName)

	switch kkpName {
	case kubermaticv1.GCPCloudProvider:
		provider = providerconfig.CloudProviderGoogle
	case kubermaticv1.VMwareCloudDirectorCloudProvider:
		provider = providerconfig.CloudProviderVMwareCloudDirector
	}

	for _, allowed := range providerconfig.AllCloudProviders {
		if allowed == provider {
			return provider, nil
		}
	}

	return "", fmt.Errorf("unknown cloud provider %q given", kkpName)
}

// KubermaticProviderType is the inverse of MachineControllerProviderName.
func KubermaticProviderType(mcName providerconfig.CloudProvider) (kubermaticv1.ProviderType, error) {
	provider := kubermaticv1.ProviderType(mcName)

	switch mcName {
	case providerconfig.CloudProviderGoogle:
		provider = kubermaticv1.GCPCloudProvider
	case providerconfig.CloudProviderVMwareCloudDirector:
		provider = kubermaticv1.VMwareCloudDirectorCloudProvider
	}

	for _, allowed := range kubermaticv1.SupportedProviders {
		if allowed == provider {
			return provider, nil
		}
	}

	return "", fmt.Errorf("unknown/unsupported cloud provider %q given", mcName)
}

// OperatingSystemFromSpec returns the OS name for the given OS spec.
func OperatingSystemFromSpec(osSpec interface{}) (providerconfig.OperatingSystem, error) {
	switch osSpec.(type) {
	case rhel.Config:
		return providerconfig.OperatingSystemRHEL, nil
	case rockylinux.Config:
		return providerconfig.OperatingSystemRockyLinux, nil
	case ubuntu.Config:
		return providerconfig.OperatingSystemUbuntu, nil
	case amzn2.Config:
		return providerconfig.OperatingSystemAmazonLinux2, nil
	case flatcar.Config:
		return providerconfig.OperatingSystemFlatcar, nil
	}

	return "", fmt.Errorf("cannot determine OS from the given osSpec (%T)", osSpec)
}

func ProviderTypeFromSpec(cloudProviderSpec interface{}) (kubermaticv1.ProviderType, error) {
	switch cloudProviderSpec.(type) {
	case alibaba.RawConfig, *alibaba.RawConfig:
		return kubermaticv1.AlibabaCloudProvider, nil
	case anexia.RawConfig, *anexia.RawConfig:
		return kubermaticv1.AnexiaCloudProvider, nil
	case aws.RawConfig, *aws.RawConfig:
		return kubermaticv1.AWSCloudProvider, nil
	case azure.RawConfig, *azure.RawConfig:
		return kubermaticv1.AzureCloudProvider, nil
	case digitalocean.RawConfig, *digitalocean.RawConfig:
		return kubermaticv1.DigitaloceanCloudProvider, nil
	case gce.RawConfig, *gce.RawConfig:
		return kubermaticv1.GCPCloudProvider, nil
	case hetzner.RawConfig, *hetzner.RawConfig:
		return kubermaticv1.HetznerCloudProvider, nil
	case kubevirt.RawConfig, *kubevirt.RawConfig:
		return kubermaticv1.KubevirtCloudProvider, nil
	case nutanix.RawConfig, *nutanix.RawConfig:
		return kubermaticv1.NutanixCloudProvider, nil
	case openstack.RawConfig, *openstack.RawConfig:
		return kubermaticv1.OpenstackCloudProvider, nil
	case vmwareclouddirector.RawConfig, *vmwareclouddirector.RawConfig:
		return kubermaticv1.VMwareCloudDirectorCloudProvider, nil
	case vsphere.RawConfig, *vsphere.RawConfig:
		return kubermaticv1.VSphereCloudProvider, nil
	case baremetal.RawConfig, *baremetal.RawConfig:
		return kubermaticv1.BaremetalCloudProvider, nil
	default:
		return "", fmt.Errorf("cannot handle unknown cloud provider %T", cloudProviderSpec)
	}
}

func DecodeCloudProviderSpec(cloudProvider kubermaticv1.ProviderType, pconfig providerconfig.Config) (interface{}, error) {
	switch cloudProvider {
	case kubermaticv1.AlibabaCloudProvider:
		return alibaba.GetConfig(pconfig)
	case kubermaticv1.AnexiaCloudProvider:
		return anexia.GetConfig(pconfig)
	case kubermaticv1.AWSCloudProvider:
		return aws.GetConfig(pconfig)
	case kubermaticv1.AzureCloudProvider:
		return azure.GetConfig(pconfig)
	case kubermaticv1.DigitaloceanCloudProvider:
		return digitalocean.GetConfig(pconfig)
	case kubermaticv1.GCPCloudProvider:
		return gce.GetConfig(pconfig)
	case kubermaticv1.HetznerCloudProvider:
		return hetzner.GetConfig(pconfig)
	case kubermaticv1.KubevirtCloudProvider:
		return kubevirt.GetConfig(pconfig)
	case kubermaticv1.NutanixCloudProvider:
		return nutanix.GetConfig(pconfig)
	case kubermaticv1.OpenstackCloudProvider:
		return openstack.GetConfig(pconfig)
	case kubermaticv1.VMwareCloudDirectorCloudProvider:
		return vmwareclouddirector.GetConfig(pconfig)
	case kubermaticv1.VSphereCloudProvider:
		return vsphere.GetConfig(pconfig)
	case kubermaticv1.BaremetalCloudProvider:
		return baremetal.GetConfig(pconfig)
	default:
		return nil, fmt.Errorf("cannot handle unknown cloud provider %q", cloudProvider)
	}
}

func assert[T any](spec interface{}) *T {
	var empty T

	if spec == nil {
		return &empty
	}

	asserted, ok := spec.(T)
	if ok {
		return &asserted
	}

	assertedPtr, ok := spec.(*T)
	if ok {
		return assertedPtr
	}

	panic(fmt.Errorf("spec is neither %T nor %T", empty, empty))
}

// CompleteCloudProviderSpec takes the given cloudProviderSpec (if any) and fills in the other required fields
// (for AWS for example the VPCID or instance profile name) based on the datacenter (static configuration)
// and the cluster object (dynamic infos that some providers write into the spec).
// The result is the cloudProviderSpec being ready to be marshalled into a MachineSpec to ultimately create
// the MachineDeployment.
func CompleteCloudProviderSpec(cloudProviderSpec interface{}, cloudProvider kubermaticv1.ProviderType, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter, os providerconfig.OperatingSystem) (interface{}, error) {
	// make it so that in the following lines we do not have to do one nil check per each provider
	if datacenter == nil {
		datacenter = &kubermaticv1.Datacenter{}
	}

	switch cloudProvider {
	case kubermaticv1.AlibabaCloudProvider:
		return provider.CompleteAlibabaProviderSpec(assert[alibaba.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Alibaba)
	case kubermaticv1.AnexiaCloudProvider:
		return provider.CompleteAnexiaProviderSpec(assert[anexia.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Anexia)
	case kubermaticv1.AWSCloudProvider:
		return provider.CompleteAWSProviderSpec(assert[aws.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.AWS, os)
	case kubermaticv1.AzureCloudProvider:
		return provider.CompleteAzureProviderSpec(assert[azure.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Azure, os)
	case kubermaticv1.DigitaloceanCloudProvider:
		return provider.CompleteDigitaloceanProviderSpec(assert[digitalocean.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Digitalocean)
	case kubermaticv1.GCPCloudProvider:
		return provider.CompleteGCPProviderSpec(assert[gce.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.GCP)
	case kubermaticv1.HetznerCloudProvider:
		return provider.CompleteHetznerProviderSpec(assert[hetzner.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Hetzner)
	case kubermaticv1.KubevirtCloudProvider:
		return provider.CompleteKubevirtProviderSpec(assert[kubevirt.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Kubevirt)
	case kubermaticv1.NutanixCloudProvider:
		return provider.CompleteNutanixProviderSpec(assert[nutanix.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Nutanix, os)
	case kubermaticv1.OpenstackCloudProvider:
		return provider.CompleteOpenstackProviderSpec(assert[openstack.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Openstack, os)
	case kubermaticv1.VMwareCloudDirectorCloudProvider:
		return provider.CompleteVMwareCloudDirectorProviderSpec(assert[vmwareclouddirector.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.VMwareCloudDirector, os)
	case kubermaticv1.VSphereCloudProvider:
		return provider.CompleteVSphereProviderSpec(assert[vsphere.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.VSphere, os)
	case kubermaticv1.BaremetalCloudProvider:
		return provider.CompleteBaremetalProviderSpec(assert[baremetal.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Baremetal)
	default:
		return nil, fmt.Errorf("cannot handle unknown cloud provider %q", cloudProvider)
	}
}

func CompleteNetworkConfig(config *providerconfig.NetworkConfig, cluster *kubermaticv1.Cluster) (*providerconfig.NetworkConfig, error) {
	if config == nil {
		config = &providerconfig.NetworkConfig{}
	}

	if cluster != nil {
		var ipFamily net.IPFamily

		switch {
		case cluster.IsIPv4Only():
			ipFamily = net.IPFamilyIPv4
		case cluster.IsIPv6Only():
			ipFamily = net.IPFamilyIPv6
		case cluster.IsDualStack():
			ipFamily = net.IPFamilyIPv4IPv6
		default:
			ipFamily = net.IPFamilyUnspecified
		}

		config.IPFamily = ipFamily
	}

	return config, nil
}
