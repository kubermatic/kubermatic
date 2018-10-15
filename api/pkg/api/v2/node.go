package v2

// Node is the object representing a cluster node.
// swagger:model NodeV2
type Node struct {
	Metadata ObjectMeta `json:"metadata,omitempty"`

	// required: true
	Spec   NodeSpec   `json:"spec"`
	Status NodeStatus `json:"status"`
}

// NodeCloudSpec represents the collection of cloud provider specific settings. Only one must be set at a time.
// swagger:model NodeCloudSpecV2
type NodeCloudSpec struct {
	Digitalocean *DigitaloceanNodeSpec `json:"digitalocean,omitempty"`
	AWS          *AWSNodeSpec          `json:"aws,omitempty"`
	Azure        *AzureNodeSpec        `json:"azure,omitempty"`
	Openstack    *OpenstackNodeSpec    `json:"openstack,omitempty"`
	Hetzner      *HetznerNodeSpec      `json:"hetzner,omitempty"`
	VSphere      *VSphereNodeSpec      `json:"vsphere,omitempty"`
}

// UbuntuSpec ubuntu specific settings
// swagger:model UbuntuSpecV2
type UbuntuSpec struct {
	// do a dist-upgrade on boot and reboot it required afterwards
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// CentOSSpec contains CentOS specific settings
type CentOSSpec struct {
	// do a dist-upgrade on boot and reboot it required afterwards
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// ContainerLinuxSpec ubuntu linux specific settings
// swagger:model ContainerLinuxSpecV2
type ContainerLinuxSpec struct {
	// disable container linux auto-update feature
	DisableAutoUpdate bool `json:"disableAutoUpdate"`
}

// OperatingSystemSpec represents the collection of os specific settings. Only one must be set at a time.
// swagger:model OperatingSystemSpecV2
type OperatingSystemSpec struct {
	Ubuntu         *UbuntuSpec         `json:"ubuntu,omitempty"`
	ContainerLinux *ContainerLinuxSpec `json:"containerLinux,omitempty"`
	CentOS         *CentOSSpec         `json:"centos,omitempty"`
}

// NodeVersionInfo node version information
// swagger:model NodeVersionInfoV2
type NodeVersionInfo struct {
	Kubelet string `json:"kubelet"`
}

// NodeSpec node specification
// swagger:model NodeSpecV2
type NodeSpec struct {
	// required: true
	Cloud NodeCloudSpec `json:"cloud"`
	// required: true
	OperatingSystem OperatingSystemSpec `json:"operatingSystem"`
	// required: true
	Versions NodeVersionInfo `json:"versions,omitempty"`
}

// DigitaloceanNodeSpec digitalocean node settings
// swagger:model DigitaloceanNodeSpecV2
type DigitaloceanNodeSpec struct {
	// droplet size slug
	// required: true
	Size string `json:"size"`
	// enable backups for the droplet
	Backups bool `json:"backups"`
	// enable ipv6 for the droplet
	IPv6 bool `json:"ipv6"`
	// enable monitoring for the droplet
	Monitoring bool `json:"monitoring"`
	// additional droplet tags
	Tags []string `json:"tags"`
}

// HetznerNodeSpec Hetzner node settings
// swagger:model HetznerNodeSpecV2
type HetznerNodeSpec struct {
	// server type
	// required: true
	Type string `json:"type"`
}

// AzureNodeSpec describes settings for an Azure node
// swagger:model AzureNodeSpecV1
type AzureNodeSpec struct {
	// VM size
	// required: true
	Size string `json:"size"`
	// should the machine have a publicly accessible IP address
	// required: false
	AssignPublicIP bool `json:"assignPublicIP"`
	// Additional metadata to set
	// required: false
	Tags map[string]string `json:"tags,omitempty"`
}

// VSphereNodeSpec VSphere node settings
// swagger:model VSphereNodeSpecV2
type VSphereNodeSpec struct {
	CPUs            int    `json:"cpus"`
	Memory          int    `json:"memory"`
	Template        string `json:"template"`
	TemplateNetName string `json:"templateNetName"`
}

// OpenstackNodeSpec openstack node settings
// swagger:model OpenstackNodeSpecV2
type OpenstackNodeSpec struct {
	// instance flavor
	// required: true
	Flavor string `json:"flavor"`
	// image to use
	// required: true
	Image string `json:"image"`
	// Additional metadata to set
	// required: false
	Tags map[string]string `json:"tags,omitempty"`
}

// AWSNodeSpec aws specific node settings
// swagger:model AWSNodeSpecV2
type AWSNodeSpec struct {
	// instance type. for example: t2.micro
	// required: true
	InstanceType string `json:"instanceType"`
	// size of the volume in gb. Only one volume will be created
	// required: true
	VolumeSize int64 `json:"diskSize"`
	// type of the volume. for example: gp2, io1, st1, sc1, standard
	// required: true
	VolumeType string `json:"volumeType"`
	// ami to use. Will be defaulted to a ami for your selected operating system and region. Only set this when you know what you do.
	AMI string `json:"ami"`
	// additional instance tags
	Tags map[string]string `json:"tags"`
}

// NodeResources cpu and memory of a node
// swagger:model NodeResourcesV2
type NodeResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// NodeStatus is information about the current status of a node.
// swagger:model NodeStatusV2
type NodeStatus struct {
	// name of the actual Machine object
	MachineName string `json:"machineName"`
	// resources in total
	Capacity NodeResources `json:"capacity,omitempty"`
	// allocatable resources
	Allocatable NodeResources `json:"allocatable,omitempty"`
	// different addresses of a node
	Addresses []NodeAddress `json:"addresses,omitempty"`
	// node versions and systems info
	NodeInfo NodeSystemInfo `json:"nodeInfo,omitempty"`

	// in case of a error this will contain a short error message
	ErrorReason string `json:"errorReason,omitempty"`
	// in case of a error this will contain a detailed error explanation
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// NodeAddress contains information for the node's address.
// swagger:model NodeAddressV2
type NodeAddress struct {
	// address type. for example: ExternalIP, InternalIP, InternalDNS, ExternalDNS
	Type string `json:"type"`
	// the actual address. for example: 192.168.1.1, node1.my.dns
	Address string `json:"address"`
}

// NodeSystemInfo is a set of versions/ids/uuids to uniquely identify the node.
// swagger:model NodeSystemInfoV2
type NodeSystemInfo struct {
	KernelVersion           string `json:"kernelVersion"`
	ContainerRuntime        string `json:"containerRuntime"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	KubeletVersion          string `json:"kubeletVersion"`
	OperatingSystem         string `json:"operatingSystem"`
	Architecture            string `json:"architecture"`
}
