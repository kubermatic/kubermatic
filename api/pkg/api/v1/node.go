package v1

import "time"

// NodeSpec mutually stores data of a cloud specific node.
type NodeSpec struct {
	Digitalocean *DigitaloceanNodeSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnNodeSpec `json:"bringyourown,omitempty"`
	Fake         *FakeNodeSpec         `json:"fake,omitempty"`
	AWS          *AWSNodeSpec          `json:"aws,omitempty"`
	Openstack    *OpenstackNodeSpec    `json:"openstack,omitempty"`
}

// DigitaloceanNodeSpec specifies a digital ocean node.
type DigitaloceanNodeSpec struct {
	Size string `json:"size"`
}

// OpenstackNodeSpec specifies a open stack node.
type OpenstackNodeSpec struct {
	Flavor string `json:"flavor"`
	Image  string `json:"image"`
}

// BringYourOwnNodeSpec specifies a bring your own node
type BringYourOwnNodeSpec struct{}

// FakeNodeSpec specifies a fake node.
type FakeNodeSpec struct {
	Type string `json:"type"`
	OS   string `json:"os"`
}

// AWSNodeSpec specifies an aws node.
type AWSNodeSpec struct {
	RootSize     int64  `json:"root_size"`
	InstanceType string `json:"instance_type"`
	VolumeType   string `json:"volume_type"`
	AMI          string `json:"ami"`
}

// ResourceList represents node resources
type ResourceList struct {
	CPU    string
	Memory string
}

// NodeCondition contains condition information for a node.
type NodeCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastHeartbeatTime  time.Time `json:"lastHeartbeatTime,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// NodeStatus is information about the current status of a node.
type NodeStatus struct {
	Capacity    ResourceList    `json:"capacity,omitempty"`
	Allocatable ResourceList    `json:"allocatable,omitempty"`
	Conditions  []NodeCondition `json:"conditions,omitempty"`
	Addresses   []NodeAddress   `json:"addresses,omitempty"`
	NodeInfo    NodeSystemInfo  `json:"nodeInfo,omitempty"`
}

// NodeAddress contains information for the node's address.
type NodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// NodeSystemInfo is a set of ids/uuids to uniquely identify the node.
type NodeSystemInfo struct {
	MachineID               string `json:"machineID"`
	SystemUUID              string `json:"systemUUID"`
	BootID                  string `json:"bootID"`
	KernelVersion           string `json:"kernelVersion"`
	OSImage                 string `json:"osImage"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	KubeletVersion          string `json:"kubeletVersion"`
	KubeProxyVersion        string `json:"kubeProxyVersion"`
	OperatingSystem         string `json:"operatingSystem"`
	Architecture            string `json:"architecture"`
}
