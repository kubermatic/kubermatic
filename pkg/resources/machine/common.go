/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"

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
	vcd "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"github.com/kubermatic/machine-controller/pkg/userdata/amzn2"
	"github.com/kubermatic/machine-controller/pkg/userdata/centos"
	"github.com/kubermatic/machine-controller/pkg/userdata/flatcar"
	"github.com/kubermatic/machine-controller/pkg/userdata/rhel"
	"github.com/kubermatic/machine-controller/pkg/userdata/rockylinux"
	"github.com/kubermatic/machine-controller/pkg/userdata/sles"
	"github.com/kubermatic/machine-controller/pkg/userdata/ubuntu"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	nutanixprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/nutanix"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

func getOsName(nodeSpec apiv1.NodeSpec) (providerconfig.OperatingSystem, error) {
	if nodeSpec.OperatingSystem.CentOS != nil {
		return providerconfig.OperatingSystemCentOS, nil
	}
	if nodeSpec.OperatingSystem.Ubuntu != nil {
		return providerconfig.OperatingSystemUbuntu, nil
	}
	if nodeSpec.OperatingSystem.SLES != nil {
		return providerconfig.OperatingSystemSLES, nil
	}
	if nodeSpec.OperatingSystem.RHEL != nil {
		return providerconfig.OperatingSystemRHEL, nil
	}
	if nodeSpec.OperatingSystem.Flatcar != nil {
		return providerconfig.OperatingSystemFlatcar, nil
	}
	if nodeSpec.OperatingSystem.RockyLinux != nil {
		return providerconfig.OperatingSystemRockyLinux, nil
	}
	if nodeSpec.OperatingSystem.AmazonLinux != nil {
		return providerconfig.OperatingSystemAmazonLinux2, nil
	}

	return "", errors.New("unknown operating system")
}

func EncodeAsRawExtension(providerConfig interface{}) (*runtime.RawExtension, error) {
	ext := &runtime.RawExtension{}
	b, err := json.Marshal(providerConfig)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func GetAWSProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*aws.RawConfig, error) {
	osName, err := getOsName(nodeSpec)
	if err != nil {
		return nil, err
	}
	ami := dc.Spec.AWS.Images[osName]
	if nodeSpec.Cloud.AWS.AMI != "" {
		ami = nodeSpec.Cloud.AWS.AMI
	}

	spotConfig := &aws.SpotInstanceConfig{}
	if nodeSpec.Cloud.AWS.IsSpotInstance != nil && *nodeSpec.Cloud.AWS.IsSpotInstance {
		if nodeSpec.Cloud.AWS.SpotInstanceMaxPrice != nil {
			spotConfig.MaxPrice = providerconfig.ConfigVarString{Value: *nodeSpec.Cloud.AWS.SpotInstanceMaxPrice}
		}

		if nodeSpec.Cloud.AWS.SpotInstancePersistentRequest != nil {
			spotConfig.PersistentRequest = providerconfig.ConfigVarBool{Value: nodeSpec.Cloud.AWS.SpotInstancePersistentRequest}
		}

		if nodeSpec.Cloud.AWS.SpotInstanceInterruptionBehavior != nil {
			spotConfig.InterruptionBehavior = providerconfig.ConfigVarString{Value: *nodeSpec.Cloud.AWS.SpotInstanceInterruptionBehavior}
		}
	}

	config := &aws.RawConfig{
		// If the node spec doesn't provide a subnet ID, AWS will just pick the AZ's default subnet.
		SubnetID:             providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.SubnetID},
		VpcID:                providerconfig.ConfigVarString{Value: c.Spec.Cloud.AWS.VPCID},
		SecurityGroupIDs:     []providerconfig.ConfigVarString{{Value: c.Spec.Cloud.AWS.SecurityGroupID}},
		Region:               providerconfig.ConfigVarString{Value: dc.Spec.AWS.Region},
		AvailabilityZone:     providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.AvailabilityZone},
		InstanceProfile:      providerconfig.ConfigVarString{Value: c.Spec.Cloud.AWS.InstanceProfileName},
		InstanceType:         providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.InstanceType},
		DiskType:             providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.VolumeType},
		DiskSize:             nodeSpec.Cloud.AWS.VolumeSize,
		AMI:                  providerconfig.ConfigVarString{Value: ami},
		AssignPublicIP:       nodeSpec.Cloud.AWS.AssignPublicIP,
		IsSpotInstance:       nodeSpec.Cloud.AWS.IsSpotInstance,
		SpotInstanceConfig:   spotConfig,
		AssumeRoleARN:        providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.AssumeRoleARN},
		AssumeRoleExternalID: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.AssumeRoleExternalID},
		EBSVolumeEncrypted:   providerconfig.ConfigVarBool{Value: pointer.Bool(false)},
	}
	if config.DiskType.Value == "" {
		config.DiskType.Value = ec2.VolumeTypeGp2
	}
	if config.DiskSize == 0 {
		config.DiskSize = 25
	}

	config.Tags = map[string]string{}
	for key, value := range nodeSpec.Cloud.AWS.Tags {
		config.Tags[key] = value
	}
	config.Tags["kubernetes.io/cluster/"+c.Name] = ""
	config.Tags["system/cluster"] = c.Name
	projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		config.Tags["system/project"] = projectID
	}

	return config, nil
}

func getAWSProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetAWSProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetAzureProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*azure.RawConfig, error) {
	config := &azure.RawConfig{
		Location:              providerconfig.ConfigVarString{Value: dc.Spec.Azure.Location},
		ResourceGroup:         providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.ResourceGroup},
		VNetResourceGroup:     providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.VNetResourceGroup},
		VMSize:                providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Azure.Size},
		OSDiskSize:            nodeSpec.Cloud.Azure.OSDiskSize,
		DataDiskSize:          nodeSpec.Cloud.Azure.DataDiskSize,
		VNetName:              providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.VNetName},
		SubnetName:            providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.SubnetName},
		RouteTableName:        providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.RouteTableName},
		AvailabilitySet:       providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.AvailabilitySet},
		AssignAvailabilitySet: c.Spec.Cloud.Azure.AssignAvailabilitySet,
		SecurityGroupName:     providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.SecurityGroup},
		Zones:                 nodeSpec.Cloud.Azure.Zones,
		ImageID:               providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Azure.ImageID},

		// https://github.com/kubermatic/kubermatic/issues/5013#issuecomment-580357280
		AssignPublicIP: providerconfig.ConfigVarBool{Value: pointer.Bool(nodeSpec.Cloud.Azure.AssignPublicIP)},
	}
	config.Tags = map[string]string{}
	for key, value := range nodeSpec.Cloud.Azure.Tags {
		config.Tags[key] = value
	}
	config.Tags["KubernetesCluster"] = c.Name
	config.Tags["system-cluster"] = c.Name
	projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		config.Tags["system-project"] = projectID
	}

	return config, nil
}

func getAzureProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetAzureProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetVSphereProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*vsphere.RawConfig, error) {
	var datastore = ""
	// If `DatastoreCluster` is not specified we use either the Datastore
	// specified at `Cluster` or the one specified at `Datacenter` level.
	if c.Spec.Cloud.VSphere.DatastoreCluster == "" {
		datastore = defaultIfEmpty(c.Spec.Cloud.VSphere.Datastore, dc.Spec.VSphere.DefaultDatastore)
	}

	config := &vsphere.RawConfig{
		TemplateVMName:   providerconfig.ConfigVarString{Value: nodeSpec.Cloud.VSphere.Template},
		VMNetName:        providerconfig.ConfigVarString{Value: c.Spec.Cloud.VSphere.VMNetName},
		CPUs:             int32(nodeSpec.Cloud.VSphere.CPUs),
		MemoryMB:         int64(nodeSpec.Cloud.VSphere.Memory),
		DiskSizeGB:       nodeSpec.Cloud.VSphere.DiskSizeGB,
		Datacenter:       providerconfig.ConfigVarString{Value: dc.Spec.VSphere.Datacenter},
		Datastore:        providerconfig.ConfigVarString{Value: datastore},
		DatastoreCluster: providerconfig.ConfigVarString{Value: c.Spec.Cloud.VSphere.DatastoreCluster},
		Cluster:          providerconfig.ConfigVarString{Value: dc.Spec.VSphere.Cluster},
		Folder:           providerconfig.ConfigVarString{Value: c.Spec.Cloud.VSphere.Folder},
		AllowInsecure:    providerconfig.ConfigVarBool{Value: pointer.Bool(dc.Spec.VSphere.AllowInsecure)},
		ResourcePool:     providerconfig.ConfigVarString{Value: c.Spec.Cloud.VSphere.ResourcePool},
	}

	config.Tags = []vsphere.Tag{}
	for _, tag := range nodeSpec.Cloud.VSphere.Tags {
		vsphereTag := vsphere.Tag{
			Description: tag.Description,
			Name:        tag.Name,
			CategoryID:  tag.CategoryID,
		}
		// Set default category if empty
		if tag.CategoryID == "" {
			vsphereTag.CategoryID = c.Spec.Cloud.VSphere.TagCategoryID
		}
		config.Tags = append(config.Tags, vsphereTag)
	}

	return config, nil
}

func getVSphereProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetVSphereProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetVMwareCloudDirectorProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*vcd.RawConfig, error) {
	catalogName := defaultIfEmpty(nodeSpec.Cloud.VMwareCloudDirector.Catalog, dc.Spec.VMwareCloudDirector.DefaultCatalog)
	storageProfile := defaultIfEmpty(nodeSpec.Cloud.VMwareCloudDirector.StorageProfile, dc.Spec.VMwareCloudDirector.DefaultStorageProfile)

	config := &vcd.RawConfig{
		VApp:             providerconfig.ConfigVarString{Value: c.Spec.Cloud.VMwareCloudDirector.VApp},
		Template:         providerconfig.ConfigVarString{Value: nodeSpec.Cloud.VMwareCloudDirector.Template},
		Catalog:          providerconfig.ConfigVarString{Value: catalogName},
		Network:          providerconfig.ConfigVarString{Value: c.Spec.Cloud.VMwareCloudDirector.OVDCNetwork},
		CPUs:             int64(nodeSpec.Cloud.VMwareCloudDirector.CPUs),
		CPUCores:         int64(nodeSpec.Cloud.VMwareCloudDirector.CPUCores),
		MemoryMB:         int64(nodeSpec.Cloud.VMwareCloudDirector.MemoryMB),
		IPAllocationMode: nodeSpec.Cloud.VMwareCloudDirector.IPAllocationMode,
		AllowInsecure:    providerconfig.ConfigVarBool{Value: pointer.Bool(dc.Spec.VMwareCloudDirector.AllowInsecure)},
	}

	if storageProfile != "" {
		config.StorageProfile = &storageProfile
	}

	if nodeSpec.Cloud.VMwareCloudDirector.DiskIOPS != nil && *nodeSpec.Cloud.VMwareCloudDirector.DiskIOPS >= 0 {
		config.DiskIOPS = nodeSpec.Cloud.VMwareCloudDirector.DiskIOPS
	}

	if nodeSpec.Cloud.VMwareCloudDirector.DiskSizeGB != nil && *nodeSpec.Cloud.VMwareCloudDirector.DiskSizeGB > 4 {
		config.DiskSizeGB = nodeSpec.Cloud.VMwareCloudDirector.DiskSizeGB
	}

	if nodeSpec.Cloud.VMwareCloudDirector.Metadata != nil {
		config.Metadata = &nodeSpec.Cloud.VMwareCloudDirector.Metadata
	}

	return config, nil
}

func getVMwareCloudDirectorProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetVMwareCloudDirectorProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetOpenstackProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*openstack.RawConfig, error) {
	config := &openstack.RawConfig{
		Image:                     providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Openstack.Image},
		Flavor:                    providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Openstack.Flavor},
		AvailabilityZone:          providerconfig.ConfigVarString{Value: dc.Spec.Openstack.AvailabilityZone},
		Region:                    providerconfig.ConfigVarString{Value: dc.Spec.Openstack.Region},
		IdentityEndpoint:          providerconfig.ConfigVarString{Value: dc.Spec.Openstack.AuthURL},
		Network:                   providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.Network},
		Subnet:                    providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.SubnetID},
		SecurityGroups:            []providerconfig.ConfigVarString{{Value: c.Spec.Cloud.Openstack.SecurityGroups}},
		InstanceReadyCheckPeriod:  providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Openstack.InstanceReadyCheckPeriod},
		InstanceReadyCheckTimeout: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Openstack.InstanceReadyCheckTimeout},
		TrustDevicePath:           providerconfig.ConfigVarBool{Value: pointer.Bool(false)},
	}

	if nodeSpec.Cloud.Openstack.UseFloatingIP || dc.Spec.Openstack.EnforceFloatingIP {
		config.FloatingIPPool = providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.FloatingIPPool}
	}

	if nodeSpec.Cloud.Openstack.RootDiskSizeGB != nil && *nodeSpec.Cloud.Openstack.RootDiskSizeGB > 0 {
		config.RootDiskSizeGB = nodeSpec.Cloud.Openstack.RootDiskSizeGB
	}

	if dc.Spec.Openstack.TrustDevicePath != nil {
		config.TrustDevicePath = providerconfig.ConfigVarBool{Value: dc.Spec.Openstack.TrustDevicePath}
	}

	// Use the nodeDeployment spec AvailabilityZone if set, otherwise we stick to the default from the datacenter
	if nodeSpec.Cloud.Openstack.AvailabilityZone != "" {
		config.AvailabilityZone = providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Openstack.AvailabilityZone}
	}

	config.Tags = map[string]string{}
	for key, value := range nodeSpec.Cloud.Openstack.Tags {
		config.Tags[key] = value
	}
	config.Tags["kubernetes-cluster"] = c.Name
	config.Tags["system-cluster"] = c.Name
	projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		config.Tags["system-project"] = projectID
	}

	return config, nil
}

func getOpenstackProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetOpenstackProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetHetznerProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*hetzner.RawConfig, error) {
	network := nodeSpec.Cloud.Hetzner.Network
	// fall back to network defined in cluster spec
	if network == "" {
		network = c.Spec.Cloud.Hetzner.Network
	}
	// fall back to network defined in datacenter spec
	if network == "" {
		network = dc.Spec.Hetzner.Network
	}

	networks := []providerconfig.ConfigVarString{}

	if network != "" {
		networks = append(networks, providerconfig.ConfigVarString{Value: network})
	}

	config := &hetzner.RawConfig{
		Datacenter: providerconfig.ConfigVarString{Value: dc.Spec.Hetzner.Datacenter},
		Location:   providerconfig.ConfigVarString{Value: dc.Spec.Hetzner.Location},
		Networks:   networks,
		ServerType: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Hetzner.Type},
	}

	return config, nil
}

func getHetznerProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetHetznerProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetDigitaloceanProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*digitalocean.RawConfig, error) {
	config := &digitalocean.RawConfig{
		Region:            providerconfig.ConfigVarString{Value: dc.Spec.Digitalocean.Region},
		Backups:           providerconfig.ConfigVarBool{Value: pointer.Bool(nodeSpec.Cloud.Digitalocean.Backups)},
		IPv6:              providerconfig.ConfigVarBool{Value: pointer.Bool(nodeSpec.Cloud.Digitalocean.IPv6)},
		Monitoring:        providerconfig.ConfigVarBool{Value: pointer.Bool(nodeSpec.Cloud.Digitalocean.Monitoring)},
		Size:              providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Digitalocean.Size},
		PrivateNetworking: providerconfig.ConfigVarBool{Value: pointer.Bool(true)},
	}

	tags := sets.NewString(nodeSpec.Cloud.Digitalocean.Tags...)
	tags.Insert("kubernetes", fmt.Sprintf("kubernetes-cluster-%s", c.Name), fmt.Sprintf("system-cluster-%s", c.Name))
	projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		tags.Insert(fmt.Sprintf("system-project-%s", projectID))
	}

	config.Tags = make([]providerconfig.ConfigVarString, len(tags.List()))
	for i, tag := range tags.List() {
		config.Tags[i].Value = tag
	}

	return config, nil
}

func getDigitaloceanProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetDigitaloceanProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetPacketProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*equinixmetal.RawConfig, error) {
	config := &equinixmetal.RawConfig{
		InstanceType: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Packet.InstanceType},
	}

	tags := sets.NewString(nodeSpec.Cloud.Packet.Tags...)
	tags.Insert("kubernetes", fmt.Sprintf("kubernetes-cluster-%s", c.Name), fmt.Sprintf("system/cluster:%s", c.Name))
	projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		tags.Insert(fmt.Sprintf("system/project:%s", projectID))
	}
	config.Tags = make([]providerconfig.ConfigVarString, len(tags.List()))
	for i, tag := range tags.List() {
		config.Tags[i].Value = tag
	}

	var facilities = sets.String{}
	if dc.Spec.Packet.Facilities != nil {
		facilities = sets.NewString(dc.Spec.Packet.Facilities...)
		config.Facilities = make([]providerconfig.ConfigVarString, len(facilities.List()))
		for i, facility := range facilities.List() {
			config.Facilities[i].Value = facility
		}
	}

	if len(facilities) < 1 && dc.Spec.Packet.Metro == "" {
		return nil, errors.New("equinixmetal metro or facilities must be specified")
	}

	config.Metro.Value = dc.Spec.Packet.Metro

	return config, nil
}

func getPacketProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetPacketProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetGCPProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*gce.CloudProviderSpec, error) {
	config := &gce.CloudProviderSpec{
		Zone:                  providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.Zone},
		MachineType:           providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.MachineType},
		DiskSize:              nodeSpec.Cloud.GCP.DiskSize,
		DiskType:              providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.DiskType},
		Preemptible:           providerconfig.ConfigVarBool{Value: pointer.Bool(nodeSpec.Cloud.GCP.Preemptible)},
		Network:               providerconfig.ConfigVarString{Value: c.Spec.Cloud.GCP.Network},
		Subnetwork:            providerconfig.ConfigVarString{Value: c.Spec.Cloud.GCP.Subnetwork},
		AssignPublicIPAddress: &providerconfig.ConfigVarBool{Value: pointer.Bool(true)},
		CustomImage:           providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.CustomImage},
		MultiZone:             providerconfig.ConfigVarBool{Value: pointer.Bool(false)},
		Regional:              providerconfig.ConfigVarBool{Value: pointer.Bool(false)},
	}

	tags := sets.NewString(nodeSpec.Cloud.GCP.Tags...)
	tags.Insert(fmt.Sprintf("kubernetes-cluster-%s", c.Name), fmt.Sprintf("system-cluster-%s", c.Name))
	projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		tags.Insert(fmt.Sprintf("system-project-%s", projectID))
	}
	config.Tags = tags.List()

	config.Labels = map[string]string{}
	for key, value := range nodeSpec.Cloud.GCP.Labels {
		config.Labels[key] = value
	}

	return config, nil
}

func getGCPProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetGCPProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetKubevirtProviderConfig(cluster *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*kubevirt.RawConfig, error) {
	config := &kubevirt.RawConfig{
		VirtualMachine: kubevirt.VirtualMachine{
			Flavor: kubevirt.Flavor{
				Name:    providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.FlavorName},
				Profile: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.FlavorProfile},
			},
			Template: kubevirt.Template{
				CPUs:   providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.CPUs},
				Memory: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.Memory},
				PrimaryDisk: kubevirt.PrimaryDisk{
					Disk: kubevirt.Disk{
						Size:             providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.PrimaryDiskSize},
						StorageClassName: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.PrimaryDiskStorageClassName},
					},
					OsImage: providerconfig.ConfigVarString{Value: extractKubeVirtOsImageURLOrDataVolumeNsName(cluster.Status.NamespaceName, nodeSpec.Cloud.Kubevirt.PrimaryDiskOSImage)},
				},
			},
			DNSPolicy: providerconfig.ConfigVarString{Value: dc.Spec.Kubevirt.DNSPolicy},
			DNSConfig: dc.Spec.Kubevirt.DNSConfig.DeepCopy(),
		},
		Affinity: kubevirt.Affinity{
			PodAffinityPreset:     providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.PodAffinityPreset},
			PodAntiAffinityPreset: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.PodAntiAffinityPreset},
			NodeAffinityPreset: kubevirt.NodeAffinityPreset{
				Type: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.NodeAffinityPreset.Type},
				Key:  providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.NodeAffinityPreset.Key},
			},
		},
	}
	config.VirtualMachine.Template.SecondaryDisks = make([]kubevirt.SecondaryDisks, 0, len(nodeSpec.Cloud.Kubevirt.SecondaryDisks))
	for _, sd := range nodeSpec.Cloud.Kubevirt.SecondaryDisks {
		secondaryDisk := kubevirt.SecondaryDisks{Disk: kubevirt.Disk{
			Size:             providerconfig.ConfigVarString{Value: sd.Size},
			StorageClassName: providerconfig.ConfigVarString{Value: sd.StorageClassName},
		}}
		config.VirtualMachine.Template.SecondaryDisks = append(config.VirtualMachine.Template.SecondaryDisks, secondaryDisk)
	}
	config.Affinity.NodeAffinityPreset.Values = make([]providerconfig.ConfigVarString, 0, len(nodeSpec.Cloud.Kubevirt.NodeAffinityPreset.Values))
	for _, val := range nodeSpec.Cloud.Kubevirt.NodeAffinityPreset.Values {
		config.Affinity.NodeAffinityPreset.Values = append(config.Affinity.NodeAffinityPreset.Values, providerconfig.ConfigVarString{Value: val})
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

func getKubevirtProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetKubevirtProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetAlibabaProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*alibaba.RawConfig, error) {
	config := &alibaba.RawConfig{
		InstanceType:            providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Alibaba.InstanceType},
		DiskSize:                providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Alibaba.DiskSize},
		DiskType:                providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Alibaba.DiskType},
		VSwitchID:               providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Alibaba.VSwitchID},
		RegionID:                providerconfig.ConfigVarString{Value: dc.Spec.Alibaba.Region},
		InternetMaxBandwidthOut: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Alibaba.InternetMaxBandwidthOut},
		ZoneID:                  providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Alibaba.ZoneID},
	}

	config.Labels = map[string]string{}
	for key, value := range nodeSpec.Cloud.Alibaba.Labels {
		config.Labels[key] = value
	}

	return config, nil
}

func getAlibabaProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetAlibabaProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetAnexiaProviderConfig(_ *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*anexia.RawConfig, error) {
	config := &anexia.RawConfig{
		VlanID:     providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Anexia.VlanID},
		TemplateID: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Anexia.TemplateID},
		CPUs:       nodeSpec.Cloud.Anexia.CPUs,
		Memory:     int(nodeSpec.Cloud.Anexia.Memory),
		DiskSize:   int(nodeSpec.Cloud.Anexia.DiskSize),
		LocationID: providerconfig.ConfigVarString{Value: dc.Spec.Anexia.LocationID},
	}

	return config, nil
}

func getAnexiaProviderSpec(nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetAnexiaProviderConfig(nil, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func GetNutanixProviderConfig(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*nutanix.RawConfig, error) {
	config := &nutanix.RawConfig{
		SubnetName: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Nutanix.SubnetName},
		ImageName:  providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Nutanix.ImageName},

		Categories: nodeSpec.Cloud.Nutanix.Categories,

		CPUs:           nodeSpec.Cloud.Nutanix.CPUs,
		CPUCores:       nodeSpec.Cloud.Nutanix.CPUCores,
		CPUPassthrough: nodeSpec.Cloud.Nutanix.CPUPassthrough,

		MemoryMB: nodeSpec.Cloud.Nutanix.MemoryMB,
		DiskSize: nodeSpec.Cloud.Nutanix.DiskSize,
	}

	if c.Spec.Cloud.Nutanix.ProjectName != "" && c.Spec.Cloud.Nutanix.ProjectName != nutanixprovider.DefaultProject {
		config.ProjectName = &providerconfig.ConfigVarString{Value: c.Spec.Cloud.Nutanix.ProjectName}
	}

	config.Categories = map[string]string{}
	for key, value := range nodeSpec.Cloud.Nutanix.Categories {
		config.Categories[key] = value
	}

	config.Categories[nutanixprovider.ClusterCategoryName] = nutanixprovider.CategoryValue(c.Name)

	if projectID, ok := c.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
		config.Categories[nutanixprovider.ProjectCategoryName] = projectID
	}

	return config, nil
}

func getNutanixProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config, err := GetNutanixProviderConfig(c, nodeSpec, dc)
	if err != nil {
		return nil, err
	}

	return EncodeAsRawExtension(config)
}

func getCentOSOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	return EncodeAsRawExtension(centos.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.CentOS.DistUpgradeOnBoot,
	})
}

func getUbuntuOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	return EncodeAsRawExtension(ubuntu.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.Ubuntu.DistUpgradeOnBoot,
	})
}

func getSLESOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	return EncodeAsRawExtension(sles.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.SLES.DistUpgradeOnBoot,
	})
}

func getRHELOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	return EncodeAsRawExtension(rhel.Config{
		DistUpgradeOnBoot:               nodeSpec.OperatingSystem.RHEL.DistUpgradeOnBoot,
		RHELSubscriptionManagerUser:     nodeSpec.OperatingSystem.RHEL.RHELSubscriptionManagerUser,
		RHELSubscriptionManagerPassword: nodeSpec.OperatingSystem.RHEL.RHELSubscriptionManagerPassword,
		RHSMOfflineToken:                nodeSpec.OperatingSystem.RHEL.RHSMOfflineToken,
	})
}

func getFlatcarOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := flatcar.Config{
		DisableAutoUpdate: nodeSpec.OperatingSystem.Flatcar.DisableAutoUpdate,
		// We manage Flatcar updates via the CoreOS update operator which requires locksmithd
		// to be disabled: https://github.com/coreos/container-linux-update-operator#design
		DisableLocksmithD: true,

		ProvisioningUtility: flatcar.Ignition,
	}
	// Force cloud-init on Anexia since it doesn't have support for ignition
	if nodeSpec.Cloud.Anexia != nil {
		config.ProvisioningUtility = flatcar.CloudInit
	}

	return EncodeAsRawExtension(config)
}

func getRockyLinuxOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	return EncodeAsRawExtension(rockylinux.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.RockyLinux.DistUpgradeOnBoot,
	})
}

func getAmazonLinuxOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	return EncodeAsRawExtension(amzn2.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.AmazonLinux.DistUpgradeOnBoot,
	})
}

// defaultIfEmpty returns the given value if not empty or the default value
// otherwise.
func defaultIfEmpty(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}
