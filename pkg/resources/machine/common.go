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

	"github.com/aws/aws-sdk-go/service/ec2"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	alibaba "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/alibaba/types"
	aws "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	azure "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure/types"
	digitalocean "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/digitalocean/types"
	gce "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce/types"
	hetzner "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/hetzner/types"
	kubevirt "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/kubevirt/types"
	openstack "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	packet "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/packet/types"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"github.com/kubermatic/machine-controller/pkg/userdata/centos"
	"github.com/kubermatic/machine-controller/pkg/userdata/coreos"
	"github.com/kubermatic/machine-controller/pkg/userdata/flatcar"
	"github.com/kubermatic/machine-controller/pkg/userdata/rhel"
	"github.com/kubermatic/machine-controller/pkg/userdata/sles"
	"github.com/kubermatic/machine-controller/pkg/userdata/ubuntu"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
)

func getOsName(nodeSpec apiv1.NodeSpec) (providerconfig.OperatingSystem, error) {
	if nodeSpec.OperatingSystem.CentOS != nil {
		return providerconfig.OperatingSystemCentOS, nil
	}
	if nodeSpec.OperatingSystem.Ubuntu != nil {
		return providerconfig.OperatingSystemUbuntu, nil
	}
	if nodeSpec.OperatingSystem.ContainerLinux != nil {
		return providerconfig.OperatingSystemCoreos, nil
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

	return "", errors.New("unknown operating system")
}

func getAWSProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	osName, err := getOsName(nodeSpec)
	if err != nil {
		return nil, err
	}
	ami := dc.Spec.AWS.Images[osName]
	if nodeSpec.Cloud.AWS.AMI != "" {
		ami = nodeSpec.Cloud.AWS.AMI
	}

	config := aws.RawConfig{
		// If the node spec doesn't provide a subnet ID, AWS will just pick the AZ's default subnet.
		SubnetID:         providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.SubnetID},
		VpcID:            providerconfig.ConfigVarString{Value: c.Spec.Cloud.AWS.VPCID},
		SecurityGroupIDs: []providerconfig.ConfigVarString{{Value: c.Spec.Cloud.AWS.SecurityGroupID}},
		Region:           providerconfig.ConfigVarString{Value: dc.Spec.AWS.Region},
		AvailabilityZone: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.AvailabilityZone},
		InstanceProfile:  providerconfig.ConfigVarString{Value: c.Spec.Cloud.AWS.InstanceProfileName},
		InstanceType:     providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.InstanceType},
		DiskType:         providerconfig.ConfigVarString{Value: nodeSpec.Cloud.AWS.VolumeType},
		DiskSize:         nodeSpec.Cloud.AWS.VolumeSize,
		AMI:              providerconfig.ConfigVarString{Value: ami},
		AssignPublicIP:   nodeSpec.Cloud.AWS.AssignPublicIP,
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

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getAzureProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config := azure.RawConfig{
		Location:          providerconfig.ConfigVarString{Value: dc.Spec.Azure.Location},
		ResourceGroup:     providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.ResourceGroup},
		VMSize:            providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Azure.Size},
		OSDiskSize:        nodeSpec.Cloud.Azure.OSDiskSize,
		DataDiskSize:      nodeSpec.Cloud.Azure.DataDiskSize,
		VNetName:          providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.VNetName},
		SubnetName:        providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.SubnetName},
		RouteTableName:    providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.RouteTableName},
		AvailabilitySet:   providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.AvailabilitySet},
		SecurityGroupName: providerconfig.ConfigVarString{Value: c.Spec.Cloud.Azure.SecurityGroup},
		Zones:             nodeSpec.Cloud.Azure.Zones,
		ImageID:           providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Azure.ImageID},

		// https://github.com/kubermatic/kubermatic/issues/5013#issuecomment-580357280
		AssignPublicIP: providerconfig.ConfigVarBool{Value: nodeSpec.Cloud.Azure.AssignPublicIP},
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

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getVSphereProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	var datastore = ""
	// If `DatastoreCluster` is not specified we use either the Datastore
	// specified at `Cluster` or the one specified at `Datacenter` level.
	if c.Spec.Cloud.VSphere.DatastoreCluster == "" {
		datastore = defaultIfEmpty(c.Spec.Cloud.VSphere.Datastore, dc.Spec.VSphere.DefaultDatastore)
	}
	config := vsphere.RawConfig{
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
		AllowInsecure:    providerconfig.ConfigVarBool{Value: dc.Spec.VSphere.AllowInsecure},
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getOpenstackProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config := openstack.RawConfig{
		Image:            providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Openstack.Image},
		Flavor:           providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Openstack.Flavor},
		AvailabilityZone: providerconfig.ConfigVarString{Value: dc.Spec.Openstack.AvailabilityZone},
		Region:           providerconfig.ConfigVarString{Value: dc.Spec.Openstack.Region},
		IdentityEndpoint: providerconfig.ConfigVarString{Value: dc.Spec.Openstack.AuthURL},
		Network:          providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.Network},
		Subnet:           providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.SubnetID},
		SecurityGroups:   []providerconfig.ConfigVarString{{Value: c.Spec.Cloud.Openstack.SecurityGroups}},
	}

	if nodeSpec.Cloud.Openstack.UseFloatingIP || dc.Spec.Openstack.EnforceFloatingIP {
		config.FloatingIPPool = providerconfig.ConfigVarString{Value: c.Spec.Cloud.Openstack.FloatingIPPool}
	}

	if nodeSpec.Cloud.Openstack.RootDiskSizeGB != nil && *nodeSpec.Cloud.Openstack.RootDiskSizeGB > 0 {
		config.RootDiskSizeGB = nodeSpec.Cloud.Openstack.RootDiskSizeGB
	}

	if dc.Spec.Openstack.TrustDevicePath != nil {
		config.TrustDevicePath = providerconfig.ConfigVarBool{Value: *dc.Spec.Openstack.TrustDevicePath}
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
	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getHetznerProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config := hetzner.RawConfig{
		Datacenter: providerconfig.ConfigVarString{Value: dc.Spec.Hetzner.Datacenter},
		Location:   providerconfig.ConfigVarString{Value: dc.Spec.Hetzner.Location},
		ServerType: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Hetzner.Type},
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getDigitaloceanProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config := digitalocean.RawConfig{
		Region:            providerconfig.ConfigVarString{Value: dc.Spec.Digitalocean.Region},
		Backups:           providerconfig.ConfigVarBool{Value: nodeSpec.Cloud.Digitalocean.Backups},
		IPv6:              providerconfig.ConfigVarBool{Value: nodeSpec.Cloud.Digitalocean.IPv6},
		Monitoring:        providerconfig.ConfigVarBool{Value: nodeSpec.Cloud.Digitalocean.Monitoring},
		Size:              providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Digitalocean.Size},
		PrivateNetworking: providerconfig.ConfigVarBool{Value: true},
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

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getPacketProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config := packet.RawConfig{
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

	facilities := sets.NewString(dc.Spec.Packet.Facilities...)
	config.Facilities = make([]providerconfig.ConfigVarString, len(facilities.List()))
	for i, facility := range facilities.List() {
		config.Facilities[i].Value = facility
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getGCPProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config := gce.CloudProviderSpec{
		Zone:                  providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.Zone},
		MachineType:           providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.MachineType},
		DiskSize:              nodeSpec.Cloud.GCP.DiskSize,
		DiskType:              providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.DiskType},
		Preemptible:           providerconfig.ConfigVarBool{Value: nodeSpec.Cloud.GCP.Preemptible},
		Network:               providerconfig.ConfigVarString{Value: c.Spec.Cloud.GCP.Network},
		Subnetwork:            providerconfig.ConfigVarString{Value: c.Spec.Cloud.GCP.Subnetwork},
		AssignPublicIPAddress: &providerconfig.ConfigVarBool{Value: true},
		CustomImage:           providerconfig.ConfigVarString{Value: nodeSpec.Cloud.GCP.CustomImage},
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

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getKubevirtProviderSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := kubevirt.RawConfig{
		CPUs:             providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.CPUs},
		PVCSize:          providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.PVCSize},
		StorageClassName: providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.StorageClassName},
		SourceURL:        providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.SourceURL},
		Namespace:        providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.Namespace},
		Memory:           providerconfig.ConfigVarString{Value: nodeSpec.Cloud.Kubevirt.Memory},
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getAlibabaProviderSpec(c *kubermaticv1.Cluster, nodeSpec apiv1.NodeSpec, dc *kubermaticv1.Datacenter) (*runtime.RawExtension, error) {
	config := alibaba.RawConfig{
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

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getCentOSOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := centos.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.CentOS.DistUpgradeOnBoot,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getCoreosOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := coreos.Config{
		DisableAutoUpdate: nodeSpec.OperatingSystem.ContainerLinux.DisableAutoUpdate,
		// We manage CoreOS updates via the CoreOS update operator which requires locksmithd
		// to be disabled: https://github.com/coreos/container-linux-update-operator#design
		DisableLocksmithD: true,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getUbuntuOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := ubuntu.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.Ubuntu.DistUpgradeOnBoot,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getSLESOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := sles.Config{
		DistUpgradeOnBoot: nodeSpec.OperatingSystem.SLES.DistUpgradeOnBoot,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getRHELOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := rhel.Config{
		DistUpgradeOnBoot:               nodeSpec.OperatingSystem.RHEL.DistUpgradeOnBoot,
		RHELSubscriptionManagerUser:     nodeSpec.OperatingSystem.RHEL.RHELSubscriptionManagerUser,
		RHELSubscriptionManagerPassword: nodeSpec.OperatingSystem.RHEL.RHELSubscriptionManagerPassword,
		RHSMOfflineToken:                nodeSpec.OperatingSystem.RHEL.RHSMOfflineToken,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

func getFlatcarOperatingSystemSpec(nodeSpec apiv1.NodeSpec) (*runtime.RawExtension, error) {
	config := flatcar.Config{
		DisableAutoUpdate: nodeSpec.OperatingSystem.Flatcar.DisableAutoUpdate,
		// We manage Flatcar updates via the CoreOS update operator which requires locksmithd
		// to be disabled: https://github.com/coreos/container-linux-update-operator#design
		DisableLocksmithD: true,
	}

	ext := &runtime.RawExtension{}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	ext.Raw = b
	return ext, nil
}

// defaultIfEmpty returns the given value if not empty or the default value
// otherwise.
func defaultIfEmpty(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}
