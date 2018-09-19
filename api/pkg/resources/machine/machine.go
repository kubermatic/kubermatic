package machine

import (
	"errors"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go/service/ec2"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"

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

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// Machine returns a machine object for the given spec
func Machine(c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey) (*clusterv1alpha1.Machine, error) {
	m := clusterv1alpha1.Machine{}
	m.Name = fmt.Sprintf("machine-%s", node.Metadata.Name)
	m.Namespace = "kube-system"
	m.Spec.Name = node.Metadata.Name

	m.Spec.Versions.Kubelet = node.Spec.Versions.Kubelet

	config := providerconfig.Config{}
	config.SSHPublicKeys = make([]string, len(keys))
	for i, key := range keys {
		config.SSHPublicKeys[i] = key.Spec.PublicKey
	}

	config.ContainerRuntimeInfo.Name = node.Spec.Versions.ContainerRuntime.Name
	config.ContainerRuntimeInfo.Version = node.Spec.Versions.ContainerRuntime.Version

	var (
		err      error
		cloudExt *runtime.RawExtension
	)
	// Cloud specifics
	switch {
	case node.Spec.Cloud.AWS != nil:
		config.CloudProvider = providerconfig.CloudProviderAWS
		cloudExt, err = getAWSProviderSpec(c, node, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Azure != nil:
		config.CloudProvider = providerconfig.CloudProviderAzure
		cloudExt, err = getAzureProviderSpec(c, node, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.VSphere != nil:
		config.CloudProvider = providerconfig.CloudProviderVsphere

		// We use OverwriteCloudConfig for Vsphere to ensure we always
		// use the credentials passed in via frontend for the cloud-provider
		// functionality
		templateData := resources.NewTemplateData(c, &dc, "", nil, nil, nil, "", "", "", resource.Quantity{}, "", false, false, "", nil)
		overwriteCloudConfig, err := cloudconfig.CloudConfig(templateData)
		if err != nil {
			return nil, err
		}
		config.OverwriteCloudConfig = &overwriteCloudConfig

		cloudExt, err = getVSphereProviderSpec(c, node, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Openstack != nil:
		config.CloudProvider = providerconfig.CloudProviderOpenstack
		cloudExt, err = getOpenstackProviderSpec(c, node, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Hetzner != nil:
		config.CloudProvider = providerconfig.CloudProviderHetzner
		cloudExt, err = getHetznerProviderSpec(c, node, dc)
		if err != nil {
			return nil, err
		}
	case node.Spec.Cloud.Digitalocean != nil:
		config.CloudProvider = providerconfig.CloudProviderDigitalocean
		cloudExt, err = getDigitaloceanProviderSpec(c, node, dc)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown cloud provider")
	}
	config.CloudProviderSpec = *cloudExt

	var osExt *runtime.RawExtension

	// OS specifics
	switch {
	case node.Spec.OperatingSystem.ContainerLinux != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCoreos
		osExt, err = getCoreosOperatingSystemSpec(c, node)
		if err != nil {
			return nil, err
		}
	case node.Spec.OperatingSystem.Ubuntu != nil:
		config.OperatingSystem = providerconfig.OperatingSystemUbuntu
		osExt, err = getUbuntuOperatingSystemSpec(c, node)
		if err != nil {
			return nil, err
		}
	case node.Spec.OperatingSystem.CentOS != nil:
		config.OperatingSystem = providerconfig.OperatingSystemCentOS
		osExt, err = getCentOSOperatingSystemSpec(c, node)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown OS")
	}
	config.OperatingSystemSpec = *osExt

	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	m.Spec.ProviderConfig.Value = &runtime.RawExtension{Raw: b}

	return &m, nil
}

func getAWSProviderSpec(c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta) (*runtime.RawExtension, error) {
	config := aws.RawConfig{
		SubnetID:         providerconfig.ConfigVarString{Value: c.Spec.Cloud.AWS.SubnetID},
		VpcID:            providerconfig.ConfigVarString{Value: c.Spec.Cloud.AWS.VPCID},
		SecurityGroupIDs: []providerconfig.ConfigVarString{{Value: c.Spec.Cloud.AWS.SecurityGroupID}},
		Region:           providerconfig.ConfigVarString{Value: dc.Spec.AWS.Region},
		AvailabilityZone: providerconfig.ConfigVarString{Value: dc.Spec.AWS.Region + dc.Spec.AWS.ZoneCharacter},
		InstanceProfile:  providerconfig.ConfigVarString{Value: c.Spec.Cloud.AWS.InstanceProfileName},
		InstanceType:     providerconfig.ConfigVarString{Value: node.Spec.Cloud.AWS.InstanceType},
		DiskType:         providerconfig.ConfigVarString{Value: node.Spec.Cloud.AWS.VolumeType},
		DiskSize:         node.Spec.Cloud.AWS.VolumeSize,
		AMI:              providerconfig.ConfigVarString{Value: node.Spec.Cloud.AWS.AMI},
	}
	if config.DiskType.Value == "" {
		config.DiskType.Value = ec2.VolumeTypeGp2
	}
	if config.DiskSize == 0 {
		config.DiskSize = 25
	}

	config.Tags = map[string]string{}
	for key, value := range node.Spec.Cloud.AWS.Tags {
		config.Tags[key] = value
	}
	config.Tags["kubernetes.io/cluster/"+c.Name] = ""

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getAzureProviderSpec(c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta) (*runtime.RawExtension, error) {
	config := azure.RawConfig{
		SubscriptionID: providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.SubscriptionID},
		TenantID:       providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.TenantID},
		ClientID:       providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.ClientID},
		ClientSecret:   providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.ClientSecret},

		Location:        providerconfig.ConfigVarString{Value: dc.Spec.Azure.Location},
		ResourceGroup:   providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.ResourceGroup},
		VMSize:          providerconfig.ConfigVarString{Value: node.Spec.Cloud.Azure.Size},
		VNetName:        providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.VNetName},
		SubnetName:      providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.SubnetName},
		RouteTableName:  providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.RouteTableName},
		AvailabilitySet: providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.AvailabilitySet},

		// Revisit when we have the DNAT topic complete and we can use private machines. Then we can use: node.Spec.Cloud.Azure.AssignPublicIP
		AssignPublicIP: providerconfig.ConfigVarBool{Value: true},
	}
	config.Tags = map[string]string{}
	for key, value := range node.Spec.Cloud.Azure.Tags {
		config.Tags[key] = value
	}
	config.Tags["KubernetesCluster"] = c.Name

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getVSphereProviderSpec(c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta) (*runtime.RawExtension, error) {
	folderPath := path.Join(dc.Spec.VSphere.RootPath, c.ObjectMeta.Name)

	config := vsphere.RawConfig{
		TemplateVMName:  providerconfig.ConfigVarString{Value: node.Spec.Cloud.VSphere.Template},
		TemplateNetName: providerconfig.ConfigVarString{Value: node.Spec.Cloud.VSphere.TemplateNetName},
		VMNetName:       providerconfig.ConfigVarString{Value: c.Spec.Cloud.VSphere.VMNetName},
		CPUs:            int32(node.Spec.Cloud.VSphere.CPUs),
		MemoryMB:        int64(node.Spec.Cloud.VSphere.Memory),
		Datacenter:      providerconfig.ConfigVarString{Value: dc.Spec.VSphere.Datacenter},
		Datastore:       providerconfig.ConfigVarString{Value: dc.Spec.VSphere.Datastore},
		Cluster:         providerconfig.ConfigVarString{Value: dc.Spec.VSphere.Cluster},
		Folder:          providerconfig.ConfigVarString{Value: folderPath},
		AllowInsecure:   providerconfig.ConfigVarBool{Value: dc.Spec.VSphere.AllowInsecure},
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getOpenstackProviderSpec(c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta) (*runtime.RawExtension, error) {
	config := openstack.RawConfig{
		Image:            providerconfig.ConfigVarString{Value: node.Spec.Cloud.Openstack.Image},
		Flavor:           providerconfig.ConfigVarString{Value: node.Spec.Cloud.Openstack.Flavor},
		AvailabilityZone: providerconfig.ConfigVarString{Value: dc.Spec.Openstack.AvailabilityZone},
		Region:           providerconfig.ConfigVarString{Value: dc.Spec.Openstack.Region},
		IdentityEndpoint: providerconfig.ConfigVarString{Value: dc.Spec.Openstack.AuthURL},
		Network:          providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.Network},
		FloatingIPPool:   providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.FloatingIPPool},
		Subnet:           providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.SubnetID},
		SecurityGroups:   []providerconfig.ConfigVarString{{Value: c.Spec.Cloud.Openstack.SecurityGroups}},
	}

	config.Tags = map[string]string{}
	for key, value := range node.Spec.Cloud.Openstack.Tags {
		config.Tags[key] = value
	}
	config.Tags["kubermatic-cluster"] = c.Name

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getHetznerProviderSpec(c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta) (*runtime.RawExtension, error) {
	config := hetzner.RawConfig{
		Datacenter: providerconfig.ConfigVarString{Value: dc.Spec.Hetzner.Datacenter},
		Location:   providerconfig.ConfigVarString{Value: dc.Spec.Hetzner.Location},
		ServerType: providerconfig.ConfigVarString{Value: node.Spec.Cloud.Hetzner.Type},
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getDigitaloceanProviderSpec(c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta) (*runtime.RawExtension, error) {
	config := digitalocean.RawConfig{
		Region:            providerconfig.ConfigVarString{Value: dc.Spec.Digitalocean.Region},
		Backups:           providerconfig.ConfigVarBool{Value: node.Spec.Cloud.Digitalocean.Backups},
		IPv6:              providerconfig.ConfigVarBool{Value: node.Spec.Cloud.Digitalocean.IPv6},
		Monitoring:        providerconfig.ConfigVarBool{Value: node.Spec.Cloud.Digitalocean.Monitoring},
		Size:              providerconfig.ConfigVarString{Value: node.Spec.Cloud.Digitalocean.Size},
		PrivateNetworking: providerconfig.ConfigVarBool{Value: true},
	}

	tags := sets.NewString(node.Spec.Cloud.Digitalocean.Tags...)
	tags.Insert("kubermatic", "kubermatic-cluster-"+c.Name)
	config.Tags = make([]providerconfig.ConfigVarString, len(tags.List()))
	for i, tag := range tags.List() {
		config.Tags[i].Value = tag
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getCentOSOperatingSystemSpec(c *kubermaticv1.Cluster, node *apiv2.Node) (*runtime.RawExtension, error) {
	config := centos.Config{
		DistUpgradeOnBoot: node.Spec.OperatingSystem.CentOS.DistUpgradeOnBoot,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getCoreosOperatingSystemSpec(c *kubermaticv1.Cluster, node *apiv2.Node) (*runtime.RawExtension, error) {
	config := coreos.Config{
		DisableAutoUpdate: node.Spec.OperatingSystem.ContainerLinux.DisableAutoUpdate,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getUbuntuOperatingSystemSpec(c *kubermaticv1.Cluster, node *apiv2.Node) (*runtime.RawExtension, error) {
	config := ubuntu.Config{
		DistUpgradeOnBoot: node.Spec.OperatingSystem.Ubuntu.DistUpgradeOnBoot,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}
