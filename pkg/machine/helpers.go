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
	alibaba "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/alibaba/types"
	anexia "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/anexia/types"
	aws "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	azure "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure/types"
	digitalocean "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/digitalocean/types"
	equinixmetal "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/equinixmetal/types"
	gce "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce/types"
	hetzner "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/hetzner/types"
	kubevirt "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/kubevirt/types"
	nutanix "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/nutanix/types"
	openstack "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	vmwareclouddirector "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/util"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/api/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	"k8c.io/operating-system-manager/pkg/providerconfig/amzn2"
	"k8c.io/operating-system-manager/pkg/providerconfig/centos"
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

func CreateProviderConfig(cloudProvider kubermaticv1.CloudProvider, cloudProviderSpec interface{}, osSpec interface{}, networkConfig *providerconfig.NetworkConfig, sshPubKeys []string) (*providerconfig.Config, error) {
	mcCloudProvider, err := kubermaticv1helper.CloudProviderToMachineController(cloudProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to translate cloud provider %q: %w", cloudProvider, err)
	}

	operatingSystem, err := OperatingSystemFromSpec(osSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to determine operating system: %w", err)
	}

	mcOperatingSystem, err := kubermaticv1helper.OperatingSystemToMachineController(operatingSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to translate operating system %q: %w", operatingSystem, err)
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
		CloudProvider:       providerconfig.CloudProvider(mcCloudProvider),
		CloudProviderSpec:   cloudProviderSpecExt,
		OperatingSystem:     providerconfig.OperatingSystem(mcOperatingSystem),
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

// OperatingSystemFromSpec returns the OS name for the given OS spec.
func OperatingSystemFromSpec(osSpec interface{}) (kubermaticv1.OperatingSystem, error) {
	switch osSpec.(type) {
	case centos.Config:
		return kubermaticv1.OperatingSystemCentOS, nil
	case rhel.Config:
		return kubermaticv1.OperatingSystemRHEL, nil
	case rockylinux.Config:
		return kubermaticv1.OperatingSystemRockyLinux, nil
	case ubuntu.Config:
		return kubermaticv1.OperatingSystemUbuntu, nil
	case amzn2.Config:
		return kubermaticv1.OperatingSystemAmazonLinux2, nil
	case flatcar.Config:
		return kubermaticv1.OperatingSystemFlatcar, nil
	}

	return "", fmt.Errorf("cannot determine OS from the given osSpec (%T)", osSpec)
}

func CloudProviderFromSpec(cloudProviderSpec interface{}) (kubermaticv1.CloudProvider, error) {
	switch cloudProviderSpec.(type) {
	case alibaba.RawConfig:
		return kubermaticv1.CloudProviderAlibaba, nil
	case anexia.RawConfig:
		return kubermaticv1.CloudProviderAnexia, nil
	case aws.RawConfig:
		return kubermaticv1.CloudProviderAWS, nil
	case azure.RawConfig:
		return kubermaticv1.CloudProviderAzure, nil
	case digitalocean.RawConfig:
		return kubermaticv1.CloudProviderDigitalocean, nil
	case gce.RawConfig:
		return kubermaticv1.CloudProviderGCP, nil
	case hetzner.RawConfig:
		return kubermaticv1.CloudProviderHetzner, nil
	case kubevirt.RawConfig:
		return kubermaticv1.CloudProviderKubeVirt, nil
	case nutanix.RawConfig:
		return kubermaticv1.CloudProviderNutanix, nil
	case openstack.RawConfig:
		return kubermaticv1.CloudProviderOpenStack, nil
	case equinixmetal.RawConfig:
		return kubermaticv1.CloudProviderPacket, nil
	case vmwareclouddirector.RawConfig:
		return kubermaticv1.CloudProviderVMwareCloudDirector, nil
	case vsphere.RawConfig:
		return kubermaticv1.CloudProviderVSphere, nil
	default:
		return "", fmt.Errorf("cannot handle unknown cloud provider %T", cloudProviderSpec)
	}
}

func DecodeCloudProviderSpec(cloudProvider kubermaticv1.CloudProvider, pconfig providerconfig.Config) (interface{}, error) {
	switch cloudProvider {
	case kubermaticv1.CloudProviderAlibaba:
		return alibaba.GetConfig(pconfig)
	case kubermaticv1.CloudProviderAnexia:
		return anexia.GetConfig(pconfig)
	case kubermaticv1.CloudProviderAWS:
		return aws.GetConfig(pconfig)
	case kubermaticv1.CloudProviderAzure:
		return azure.GetConfig(pconfig)
	case kubermaticv1.CloudProviderDigitalocean:
		return digitalocean.GetConfig(pconfig)
	case kubermaticv1.CloudProviderGCP:
		return gce.GetConfig(pconfig)
	case kubermaticv1.CloudProviderHetzner:
		return hetzner.GetConfig(pconfig)
	case kubermaticv1.CloudProviderKubeVirt:
		return kubevirt.GetConfig(pconfig)
	case kubermaticv1.CloudProviderNutanix:
		return nutanix.GetConfig(pconfig)
	case kubermaticv1.CloudProviderOpenStack:
		return openstack.GetConfig(pconfig)
	case kubermaticv1.CloudProviderPacket:
		return equinixmetal.GetConfig(pconfig)
	case kubermaticv1.CloudProviderVMwareCloudDirector:
		return vmwareclouddirector.GetConfig(pconfig)
	case kubermaticv1.CloudProviderVSphere:
		return vsphere.GetConfig(pconfig)
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
func CompleteCloudProviderSpec(cloudProviderSpec interface{}, cloudProvider kubermaticv1.CloudProvider, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter, os kubermaticv1.OperatingSystem) (interface{}, error) {
	// make it so that in the following lines we do not have to do one nil check per each provider
	if datacenter == nil {
		datacenter = &kubermaticv1.Datacenter{}
	}

	switch cloudProvider {
	case kubermaticv1.CloudProviderAlibaba:
		return provider.CompleteAlibabaProviderSpec(assert[alibaba.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Alibaba)
	case kubermaticv1.CloudProviderAnexia:
		return provider.CompleteAnexiaProviderSpec(assert[anexia.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Anexia)
	case kubermaticv1.CloudProviderAWS:
		return provider.CompleteAWSProviderSpec(assert[aws.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.AWS, os)
	case kubermaticv1.CloudProviderAzure:
		return provider.CompleteAzureProviderSpec(assert[azure.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Azure)
	case kubermaticv1.CloudProviderDigitalocean:
		return provider.CompleteDigitaloceanProviderSpec(assert[digitalocean.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Digitalocean)
	case kubermaticv1.CloudProviderGCP:
		return provider.CompleteGCPProviderSpec(assert[gce.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.GCP)
	case kubermaticv1.CloudProviderHetzner:
		return provider.CompleteHetznerProviderSpec(assert[hetzner.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Hetzner)
	case kubermaticv1.CloudProviderKubeVirt:
		return provider.CompleteKubevirtProviderSpec(assert[kubevirt.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.KubeVirt)
	case kubermaticv1.CloudProviderNutanix:
		return provider.CompleteNutanixProviderSpec(assert[nutanix.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Nutanix, os)
	case kubermaticv1.CloudProviderOpenStack:
		return provider.CompleteOpenStackProviderSpec(assert[openstack.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.OpenStack, os)
	case kubermaticv1.CloudProviderPacket:
		return provider.CompleteEquinixMetalProviderSpec(assert[equinixmetal.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.Packet)
	case kubermaticv1.CloudProviderVMwareCloudDirector:
		return provider.CompleteVMwareCloudDirectorProviderSpec(assert[vmwareclouddirector.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.VMwareCloudDirector, os)
	case kubermaticv1.CloudProviderVSphere:
		return provider.CompleteVSphereProviderSpec(assert[vsphere.RawConfig](cloudProviderSpec), cluster, datacenter.Spec.VSphere, os)
	default:
		return nil, fmt.Errorf("cannot handle unknown cloud provider %q", cloudProvider)
	}
}

func CompleteNetworkConfig(config *providerconfig.NetworkConfig, cluster *kubermaticv1.Cluster) (*providerconfig.NetworkConfig, error) {
	if config == nil {
		config = &providerconfig.NetworkConfig{}
	}

	if cluster != nil {
		var ipFamily util.IPFamily

		switch {
		case cluster.Spec.ClusterNetwork.IsIPv4Only():
			ipFamily = util.IPFamilyIPv4
		case cluster.Spec.ClusterNetwork.IsIPv6Only():
			ipFamily = util.IPFamilyIPv6
		case cluster.Spec.ClusterNetwork.IsDualStack():
			ipFamily = util.IPFamilyIPv4IPv6
		default:
			ipFamily = util.IPFamilyUnspecified
		}

		config.IPFamily = ipFamily
	}

	return config, nil
}
