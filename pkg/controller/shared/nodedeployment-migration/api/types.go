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

package api

import (
	kubevirtv1 "kubevirt.io/api/core/v1"

	vcd "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
	"github.com/kubermatic/machine-controller/pkg/userdata/flatcar"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

// NodeDeployment represents a set of worker nodes that is part of a cluster.
type NodeDeployment struct {
	apiv1.ObjectMeta `json:",inline"`

	Spec NodeDeploymentSpec `json:"spec"`
}

// NodeDeploymentSpec node deployment specification.
type NodeDeploymentSpec struct {
	// required: true
	Replicas int32 `json:"replicas"`
	// required: true
	Template NodeSpec `json:"template"`
	// required: false
	Paused *bool `json:"paused,omitempty"`
	// Only supported for nodes with Kubernetes 1.23 or less.
	// required: false
	DynamicConfig *bool `json:"dynamicConfig,omitempty"`
}

// NodeSpec node specification.
type NodeSpec struct {
	// required: true
	Cloud NodeCloudSpec `json:"cloud"`
	// required: true
	OperatingSystem OperatingSystemSpec `json:"operatingSystem"`
	// required: false
	SSHUserName string `json:"sshUserName,omitempty"`
	// required: true
	Versions NodeVersionInfo `json:"versions,omitempty"`
	// Map of string keys and values that can be used to organize and categorize (scope and select) objects.
	// It will be applied to Nodes allowing users run their apps on specific Node using labelSelector.
	// required: false
	Labels map[string]string `json:"labels,omitempty"`
	// List of taints to set on new nodes
	Taints []TaintSpec `json:"taints,omitempty"`
}

// NodeCloudSpec represents the collection of cloud provider specific settings. Only one must be set at a time.
type NodeCloudSpec struct {
	Digitalocean        *DigitaloceanNodeSpec        `json:"digitalocean,omitempty"`
	AWS                 *AWSNodeSpec                 `json:"aws,omitempty"`
	Azure               *AzureNodeSpec               `json:"azure,omitempty"`
	Openstack           *OpenstackNodeSpec           `json:"openstack,omitempty"`
	Packet              *PacketNodeSpec              `json:"packet,omitempty"`
	Hetzner             *HetznerNodeSpec             `json:"hetzner,omitempty"`
	VSphere             *VSphereNodeSpec             `json:"vsphere,omitempty"`
	GCP                 *GCPNodeSpec                 `json:"gcp,omitempty"`
	Kubevirt            *KubevirtNodeSpec            `json:"kubevirt,omitempty"`
	Alibaba             *AlibabaNodeSpec             `json:"alibaba,omitempty"`
	Anexia              *AnexiaNodeSpec              `json:"anexia,omitempty"`
	Nutanix             *NutanixNodeSpec             `json:"nutanix,omitempty"`
	VMwareCloudDirector *VMwareCloudDirectorNodeSpec `json:"vmwareclouddirector,omitempty"`
}

// OperatingSystemSpec represents the collection of os specific settings. Only one must be set at a time.
type OperatingSystemSpec struct {
	Ubuntu      *UbuntuSpec      `json:"ubuntu,omitempty"`
	AmazonLinux *AmazonLinuxSpec `json:"amzn2,omitempty"`
	CentOS      *CentOSSpec      `json:"centos,omitempty"`
	RHEL        *RHELSpec        `json:"rhel,omitempty"`
	Flatcar     *FlatcarSpec     `json:"flatcar,omitempty"`
	RockyLinux  *RockyLinuxSpec  `json:"rockylinux,omitempty"`
}

// UbuntuSpec ubuntu specific settings.
type UbuntuSpec struct {
	// do a dist-upgrade on boot and reboot it required afterwards.
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// CentOSSpec contains CentOS specific settings.
type CentOSSpec struct {
	// do a dist-upgrade on boot and reboot it required afterwards.
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// FlatcarSpec contains Flatcar Linux specific settings.
type FlatcarSpec struct {
	// disable flatcar linux auto-update feature.
	DisableAutoUpdate bool `json:"disableAutoUpdate"`

	// ProvisioningUtility specifies the type of provisioning utility, allowed values are cloud-init and ignition.
	// Defaults to ignition.
	flatcar.ProvisioningUtility `json:"provisioningUtility,omitempty"`
}

// RHELSpec contains rhel specific settings.
type RHELSpec struct {
	// do a dist-upgrade on boot and reboot it required afterwards.
	DistUpgradeOnBoot               bool   `json:"distUpgradeOnBoot"`
	RHELSubscriptionManagerUser     string `json:"rhelSubscriptionManagerUser,omitempty"`
	RHELSubscriptionManagerPassword string `json:"rhelSubscriptionManagerPassword,omitempty"`
	RHSMOfflineToken                string `json:"rhsmOfflineToken,omitempty"`
}

// RockyLinuxSpec contains rocky-linux specific settings.
type RockyLinuxSpec struct {
	// do a dist-upgrade on boot and reboot it required afterwards.
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// AmazonLinuxSpec amazon linux specific settings.
type AmazonLinuxSpec struct {
	// do a dist-upgrade on boot and reboot it required afterwards.
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// NodeVersionInfo node version information.
type NodeVersionInfo struct {
	Kubelet string `json:"kubelet"`
}

// TaintSpec defines a node taint.
type TaintSpec struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// DigitaloceanNodeSpec digitalocean node settings.
type DigitaloceanNodeSpec struct {
	// droplet size slug.
	// required: true
	Size string `json:"size"`
	// enable backups for the droplet.
	Backups bool `json:"backups"`
	// DEPRECATED
	// IPv6 is enabled automatically based on IP Family of the cluster so setting this field is not needed.
	// enable ipv6 for the droplet
	IPv6 bool `json:"ipv6"`
	// enable monitoring for the droplet.
	Monitoring bool `json:"monitoring"`
	// additional droplet tags.
	Tags []string `json:"tags"`
}

// HetznerNodeSpec Hetzner node settings.
type HetznerNodeSpec struct {
	// server type
	// required: true
	Type string `json:"type"`
	// network name
	// required: false
	Network string `json:"network"`
}

// AzureNodeSpec describes settings for an Azure node.
type AzureNodeSpec struct {
	// VM size.
	// required: true
	Size string `json:"size"`
	// should the machine have a publicly accessible IP address.
	// required: false
	AssignPublicIP bool `json:"assignPublicIP"`
	// Additional metadata to set.
	// required: false
	Tags map[string]string `json:"tags,omitempty"`
	// OS disk size in GB.
	// required: false
	OSDiskSize int32 `json:"osDiskSize"`
	// Data disk size in GB.
	// required: false
	DataDiskSize int32 `json:"dataDiskSize"`
	// Zones represents the availability zones for azure vms.
	// required: false
	Zones []string `json:"zones"`
	// ImageID represents the ID of the image that should be used to run the node.
	// required: false
	ImageID string `json:"imageID"`
	// AssignAvailabilitySet is used to check if an availability set should be created and assigned to the cluster.
	AssignAvailabilitySet bool `json:"assignAvailabilitySet"`
}

// VSphereNodeSpec VSphere node settings.
type VSphereNodeSpec struct {
	CPUs       int    `json:"cpus"`
	Memory     int    `json:"memory"`
	DiskSizeGB *int64 `json:"diskSizeGB,omitempty"`
	Template   string `json:"template"`
	// Additional metadata to set.
	// required: false
	Tags []VSphereTag `json:"tags,omitempty"`
}

// VSphereTag represents vsphere tag.
type VSphereTag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// CategoryID when empty the default category will be used.
	CategoryID string `json:"categoryID,omitempty"`
}

// OpenstackNodeSpec openstack node settings.
type OpenstackNodeSpec struct {
	// instance flavor.
	// required: true
	Flavor string `json:"flavor"`
	// image to use.
	// required: true
	Image string `json:"image"`
	// Additional metadata to set.
	// required: false
	Tags map[string]string `json:"tags,omitempty"`
	// Defines whether floating ip should be used.
	// required: false
	UseFloatingIP bool `json:"useFloatingIP,omitempty"`
	// if set, the rootDisk will be a volume. If not, the rootDisk will be on ephemeral storage and its size will be derived from the flavor.
	// required: false
	RootDiskSizeGB *int `json:"diskSize"`
	// if not set, the default AZ from the Datacenter spec will be used.
	// required: false
	AvailabilityZone string `json:"availabilityZone"`
	// Period of time to check for instance ready status, i.e. 10s/1m
	// required: false
	InstanceReadyCheckPeriod string `json:"instanceReadyCheckPeriod"`
	// Max time to wait for the instance to be ready, i.e. 10s/1m
	// required: false
	InstanceReadyCheckTimeout string `json:"instanceReadyCheckTimeout"`
	// UUID of the server group, used to configure affinity or anti-affinity of the VM instances relative to hypervisor
	// required: false
	ServerGroup string `json:"serverGroup"`
}

// AWSNodeSpec aws specific node settings.
type AWSNodeSpec struct {
	// instance type. for example: t2.micro.
	// required: true
	InstanceType string `json:"instanceType"`
	// size of the volume in gb. Only one volume will be created
	// required: true
	VolumeSize int32 `json:"diskSize"`
	// type of the volume. for example: gp2, io1, st1, sc1, standard
	// required: true
	VolumeType string `json:"volumeType"`
	// ami to use. Will be defaulted to a ami for your selected operating system and region. Only set this when you know what you do.
	AMI string `json:"ami"`
	// additional instance tags
	Tags map[string]string `json:"tags"`
	// Availability zone in which to place the node. It is coupled with the subnet to which the node will belong.
	AvailabilityZone string `json:"availabilityZone"`
	// The VPC subnet to which the node shall be connected.
	SubnetID string `json:"subnetID"`
	// This flag controls a property of the AWS instance. When set the AWS instance will get a public IP address
	// assigned during launch overriding a possible setting in the used AWS subnet.
	// required: false
	AssignPublicIP *bool `json:"assignPublicIP"`
	// IsSpotInstance indicates whether the created machine is an aws ec2 spot instance or on-demand ec2 instance.
	IsSpotInstance *bool `json:"isSpotInstance"`
	// SpotInstanceMaxPrice is the maximum price you are willing to pay per instance hour. Your instance runs when
	// your maximum price is greater than the Spot Price.
	SpotInstanceMaxPrice *string `json:"spotInstanceMaxPrice"`
	// SpotInstancePersistentRequest ensures that your request will be submitted every time your Spot Instance is terminated.
	SpotInstancePersistentRequest *bool `json:"spotInstancePersistentRequest"`
	// SpotInstanceInterruptionBehavior sets the interruption behavior for the spot instance when capacity is no longer
	// available at the price you specified, if there is no capacity, or if a constraint cannot be met. Charges for EBS
	// volume storage apply when an instance is stopped.
	SpotInstanceInterruptionBehavior *string `json:"spotInstanceInterruptionBehavior"`
	// AssumeRoleARN defines the ARN for an IAM role that should be assumed when handling resources on AWS. It will be used
	// to acquire temporary security credentials using an STS AssumeRole API operation whenever creating an AWS session.
	// required: false
	AssumeRoleARN string `json:"assumeRoleARN"`
	// AssumeRoleExternalID is an arbitrary string that may be needed when calling the STS AssumeRole API operation.
	// Using an external ID can help to prevent the "confused deputy problem".
	// required: false
	AssumeRoleExternalID string `json:"assumeRoleExternalID"`
}

// PacketNodeSpec specifies packet specific node settings.
type PacketNodeSpec struct {
	// InstanceType denotes the plan to which the device will be provisioned.
	// required: true
	InstanceType string `json:"instanceType"`
	// additional instance tags.
	// required: false
	Tags []string `json:"tags"`
}

// GCPNodeSpec gcp specific node settings.
type GCPNodeSpec struct {
	Zone        string            `json:"zone"`
	MachineType string            `json:"machineType"`
	DiskSize    int64             `json:"diskSize"`
	DiskType    string            `json:"diskType"`
	Preemptible bool              `json:"preemptible"`
	Labels      map[string]string `json:"labels"`
	Tags        []string          `json:"tags"`
	CustomImage string            `json:"customImage"`
}

// KubevirtNodeSpec kubevirt specific node settings.
type KubevirtNodeSpec struct {
	// FlavorName states name of the virtual-machine flavor.
	//
	// Deprecated. In favor of Instancetype and Preference.
	FlavorName string `json:"flavorName"`
	// FlavorProfile states name of virtual-machine profile.
	//
	// Deprecated. In favor of Instancetype and Preference.
	FlavorProfile string `json:"flavorProfile"`
	// Instancetype provide a way to define a set of resource, performance and other runtime characteristics,
	// allowing users to reuse these definitions across multiple VirtualMachines.
	// Anything provided within an instancetype cannot be overridden within the VirtualMachine.
	Instancetype *kubevirtv1.InstancetypeMatcher `json:"instancetype"`
	// Preference are like Instancetype defining runtime characteristics. But unlike Instancetypes,
	// Preferences only represent the preferred values and as such can be overridden by values in the VirtualMachine.
	Preference *kubevirtv1.PreferenceMatcher `json:"preference"`
	// CPUs states how many cpus the kubevirt node will have.
	// required: true
	CPUs string `json:"cpus"`
	// Memory states the memory that kubevirt node will have.
	// required: true
	Memory string `json:"memory"`

	// PrimaryDiskOSImage states the source from which the imported image will be downloaded.
	// This field contains:
	// - a URL to download an Os Image from a HTTP source.
	// - a DataVolume Name as source for DataVolume cloning.
	// required: true
	PrimaryDiskOSImage string `json:"primaryDiskOSImage"`
	// PrimaryDiskStorageClassName states the storage class name for the provisioned PVCs.
	// required: true
	PrimaryDiskStorageClassName string `json:"primaryDiskStorageClassName"`
	// PrimaryDiskSize states the size of the provisioned pvc per node.
	// required: true
	PrimaryDiskSize string `json:"primaryDiskSize"`
	// SecondaryDisks contains list of secondary-disks.
	SecondaryDisks []SecondaryDisks `json:"secondaryDisks"`
	// PodAffinityPreset describes pod affinity scheduling rules.
	//
	// Deprecated: in favor of topology spread constraints.
	PodAffinityPreset string `json:"podAffinityPreset"`
	// PodAntiAffinityPreset describes pod anti-affinity scheduling rules.
	//
	// Deprecated: in favor of topology spread constraints
	PodAntiAffinityPreset string `json:"podAntiAffinityPreset"`
	// NodeAffinityPreset describes node affinity scheduling rules.
	NodeAffinityPreset NodeAffinityPreset `json:"nodeAffinityPreset"`
	// TopologySpreadConstraints describes topology spread constraints for VMs.
	TopologySpreadConstraints []TopologySpreadConstraint `json:"topologySpreadConstraints"`
}

type SecondaryDisks struct {
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName"`
}

type NodeAffinityPreset struct {
	Type   string
	Key    string
	Values []string
}

type TopologySpreadConstraint struct {
	// MaxSkew describes the degree to which VMs may be unevenly distributed.
	MaxSkew int `json:"maxSkew"`
	// TopologyKey is the key of infra-node labels.
	TopologyKey string `json:"topologyKey"`
	// WhenUnsatisfiable indicates how to deal with a VM if it doesn't satisfy.
	// the spread constraint.
	WhenUnsatisfiable string `json:"whenUnsatisfiable"`
}

// AlibabaNodeSpec alibaba specific node settings.
type AlibabaNodeSpec struct {
	InstanceType            string            `json:"instanceType"`
	DiskSize                string            `json:"diskSize"`
	DiskType                string            `json:"diskType"`
	VSwitchID               string            `json:"vSwitchID"`
	InternetMaxBandwidthOut string            `json:"internetMaxBandwidthOut"`
	Labels                  map[string]string `json:"labels"`
	ZoneID                  string            `json:"zoneID"`
}

// AnexiaDiskConfig defines a single disk for a node at anexia.
type AnexiaDiskConfig struct {
	// Disks configures this disk of each node will have.
	// required: true
	Size int64 `json:"size"`

	// PerformanceType configures the performance type this disks of each node will have.
	// Known values are something like "ENT3" or "HPC2".
	// required: false
	PerformanceType *string `json:"performanceType,omitempty"`
}

// AnexiaNodeSpec anexia specific node settings.
type AnexiaNodeSpec struct {
	// VlanID Instance vlanID.
	// required: true
	VlanID string `json:"vlanID"`
	// TemplateID instance template
	// required: true
	TemplateID string `json:"templateID"`
	// CPUs states how many cpus the node will have.
	// required: true
	CPUs int `json:"cpus"`
	// Memory states the memory that node will have.
	// required: true
	Memory int64 `json:"memory"`

	// DiskSize states the disk size that node will have.
	// Deprecated: please use the new Disks attribute instead.
	// required: false
	DiskSize *int64 `json:"diskSize"`

	// Disks configures the disks each node will have.
	// required: false
	Disks []AnexiaDiskConfig `json:"disks"`
}

// NutanixNodeSpec nutanix specific node settings.
type NutanixNodeSpec struct {
	SubnetName     string            `json:"subnetName"`
	ImageName      string            `json:"imageName"`
	Categories     map[string]string `json:"categories"`
	CPUs           int64             `json:"cpus"`
	CPUCores       *int64            `json:"cpuCores"`
	CPUPassthrough *bool             `json:"cpuPassthrough"`
	MemoryMB       int64             `json:"memoryMB"`
	DiskSize       *int64            `json:"diskSize"`
}

// VMwareCloudDirectorNodeSpec VMware Cloud Director node settings.
type VMwareCloudDirectorNodeSpec struct {
	CPUs             int                  `json:"cpus"`
	CPUCores         int                  `json:"cpuCores"`
	MemoryMB         int                  `json:"memoryMB"`
	DiskSizeGB       *int64               `json:"diskSizeGB,omitempty"`
	DiskIOPS         *int64               `json:"diskIOPS,omitempty"`
	Template         string               `json:"template"`
	Catalog          string               `json:"catalog"`
	StorageProfile   string               `json:"storageProfile"`
	IPAllocationMode vcd.IPAllocationMode `json:"ipAllocationMode,omitempty"`
	VApp             string               `json:"vapp,omitempty"`
	Network          string               `json:"network,omitempty"`
	// Additional metadata to set.
	// required: false
	Metadata map[string]string `json:"metadata,omitempty"`
}
