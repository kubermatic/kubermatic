package machine

import (
	"fmt"

	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/digitalocean"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/hetzner"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"github.com/kubermatic/machine-controller/pkg/userdata/centos"
	"github.com/kubermatic/machine-controller/pkg/userdata/coreos"
	"github.com/kubermatic/machine-controller/pkg/userdata/ubuntu"

	"k8s.io/apimachinery/pkg/util/json"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// GetAPIV2OperatingSystemSpec returns the api compatible OperatingSystemSpec for the given machine
func GetAPIV2OperatingSystemSpec(machine *clusterv1alpha1.Machine) (*apiv2.OperatingSystemSpec, error) {
	decodedProviderConfig, err := providerconfig.GetConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get machine providerConfig: %v", err)
	}

	operatingSystemSpec := &apiv2.OperatingSystemSpec{}

	if decodedProviderConfig.OperatingSystem == providerconfig.OperatingSystemCoreos {
		config := &coreos.Config{}
		if err := json.Unmarshal(decodedProviderConfig.OperatingSystemSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse coreos config: %v", err)
		}
		operatingSystemSpec.ContainerLinux = &apiv2.ContainerLinuxSpec{
			DisableAutoUpdate: config.DisableAutoUpdate,
		}
	} else if decodedProviderConfig.OperatingSystem == providerconfig.OperatingSystemUbuntu {
		config := &ubuntu.Config{}
		if err := json.Unmarshal(decodedProviderConfig.OperatingSystemSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse ubuntu config: %v", err)
		}
		operatingSystemSpec.Ubuntu = &apiv2.UbuntuSpec{
			DistUpgradeOnBoot: config.DistUpgradeOnBoot,
		}
	} else if decodedProviderConfig.OperatingSystem == providerconfig.OperatingSystemCentOS {
		config := &centos.Config{}
		if err := json.Unmarshal(decodedProviderConfig.OperatingSystemSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse centos config: %v", err)
		}
		operatingSystemSpec.CentOS = &apiv2.CentOSSpec{
			DistUpgradeOnBoot: config.DistUpgradeOnBoot,
		}
	}

	return operatingSystemSpec, nil
}

// GetAPIV2NodeCloudSpec returns the api compatible NodeCloudSpec for the given machine
func GetAPIV2NodeCloudSpec(machine *clusterv1alpha1.Machine) (*apiv2.NodeCloudSpec, error) {
	decodedProviderConfig, err := providerconfig.GetConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get machine providerConfig: %v", err)
	}

	cloudSpec := &apiv2.NodeCloudSpec{}

	switch decodedProviderConfig.CloudProvider {
	case providerconfig.CloudProviderAWS:
		config := &aws.RawConfig{}
		if err := json.Unmarshal(decodedProviderConfig.CloudProviderSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse aws config: %v", err)
		}
		cloudSpec.AWS = &apiv2.AWSNodeSpec{
			Tags:         config.Tags,
			VolumeSize:   config.DiskSize,
			VolumeType:   config.DiskType.Value,
			InstanceType: config.InstanceType.Value,
			AMI:          config.AMI.Value,
		}
	case providerconfig.CloudProviderAzure:
		config := &azure.RawConfig{}
		if err := json.Unmarshal(decodedProviderConfig.CloudProviderSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse Azure config: %v", err)
		}
		cloudSpec.Azure = &apiv2.AzureNodeSpec{
			Size:           config.VMSize.Value,
			AssignPublicIP: config.AssignPublicIP.Value,
			Tags:           config.Tags,
		}
	case providerconfig.CloudProviderDigitalocean:
		config := &digitalocean.RawConfig{}
		if err := json.Unmarshal(decodedProviderConfig.CloudProviderSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse digitalocean config: %v", err)
		}
		cloudSpec.Digitalocean = &apiv2.DigitaloceanNodeSpec{
			IPv6:       config.IPv6.Value,
			Size:       config.Size.Value,
			Backups:    config.Backups.Value,
			Monitoring: config.Monitoring.Value,
		}
		for _, v := range config.Tags {
			cloudSpec.Digitalocean.Tags = append(cloudSpec.Digitalocean.Tags, v.Value)
		}
	case providerconfig.CloudProviderOpenstack:
		config := &openstack.RawConfig{}
		if err := json.Unmarshal(decodedProviderConfig.CloudProviderSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse openstack config: %v", err)
		}
		cloudSpec.Openstack = &apiv2.OpenstackNodeSpec{
			Flavor: config.Flavor.Value,
			Image:  config.Image.Value,
			Tags:   config.Tags,
		}
	case providerconfig.CloudProviderHetzner:
		config := &hetzner.RawConfig{}
		if err := json.Unmarshal(decodedProviderConfig.CloudProviderSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse hetzner config: %v", err)
		}
		cloudSpec.Hetzner = &apiv2.HetznerNodeSpec{
			Type: config.ServerType.Value,
		}
	case providerconfig.CloudProviderVsphere:
		config := &vsphere.RawConfig{}
		if err := json.Unmarshal(decodedProviderConfig.CloudProviderSpec.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to parse vsphere config: %v", err)
		}
		cloudSpec.VSphere = &apiv2.VSphereNodeSpec{
			CPUs:            int(config.CPUs),
			Memory:          int(config.MemoryMB),
			Template:        config.TemplateVMName.Value,
			TemplateNetName: config.TemplateNetName.Value,
		}
	default:
		return nil, fmt.Errorf("unknown cloud provider %q", providerconfig.CloudProviderVsphere)
	}

	return cloudSpec, nil
}
