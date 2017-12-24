package api

import (
	"errors"
)

var (
	// ErrNotFound tells that something was not found
	ErrNotFound = errors.New("not found")
	// ErrInvalidType tells if a interface conversion failed due to invalid type
	ErrInvalidType = errors.New("invalid type")
)

// Metadata is an object storing common metadata for persistable objects.
type Metadata struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
	UID      string `json:"uid,omitempty"`

	// private fields
	// Annotations represent Annotations on Kubernetes Namespace for the respective cluster,
	// which are used to store persistent data for the cluster.
	Annotations map[string]string `json:"-"`
	User        string            `json:"-"`
}

// DigitaloceanNodeSpec specifies a digital ocean node.
type DigitaloceanNodeSpec struct {
	// Size is the size of the node (DigitalOcean node type).
	Size string `json:"size"`
	// SSHKeyFingerprints  represent the fingerprints of the keys.
	// DigitalOcean utilizes the fingerprints to identify public
	// SSHKeys stored within the DigitalOcean platform.
	SSHKeyFingerprints []string `json:"sshKeys,omitempty"`
}

// OpenstackNodeSpec specifies a open stack node.
type OpenstackNodeSpec struct {
	Flavor string `json:"flavor"`
	Image  string `json:"image"`
}

// BringYourOwnNodeSpec specifies a bring your own node
type BringYourOwnNodeSpec struct {
}

// BareMetalNodeSpec specifies a node instanciated by the bare-metal-provider
type BareMetalNodeSpec struct {
	ID       string `json:"id"`
	Memory   int    `json:"memory"`
	Space    int    `json:"space"`
	CPUs     []CPU  `json:"cpus"`
	PublicIP string `json:"public_ip"`
}

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

// ContainerLinuxSpec specifies Container Linux options
type ContainerLinuxSpec struct {
	DisableAutoUpdate bool `json:"disable_auto_update"`
}

// UbuntuSpec specifies ubuntu options
type UbuntuSpec struct {
	Version           string `json:"version"`
	DisableAutoUpdate bool   `json:"disable_auto_update"`
}

// OperatingSystemSpec specifies operations system options
type OperatingSystemSpec struct {
	ContainerLinux ContainerLinuxSpec `json:"container_linux"`
	SSHUser        string             `json:"ssh_user"`
}

// NodeSpec mutually stores data of a cloud specific node.
type NodeSpec struct {
	// OperatingSystem defines
	OperatingSystem OperatingSystemSpec `json:"operating_system"`

	Digitalocean *DigitaloceanNodeSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnNodeSpec `json:"bringyourown,omitempty"`
	Fake         *FakeNodeSpec         `json:"fake,omitempty"`
	AWS          *AWSNodeSpec          `json:"aws,omitempty"`
	BareMetal    *BareMetalNodeSpec    `json:"baremetal,omitempty"`
	Openstack    *OpenstackNodeSpec    `json:"openstack,omitempty"`
}

func (n *NodeSpec) AWSSpec() interface{} {
	return n.AWS
}

func (n *NodeSpec) FakeSpec() interface{} {
	return n.Fake
}

func (n *NodeSpec) DigitaloceanCloudSpec() interface{} {
	return n.Digitalocean
}

func (n *NodeSpec) BringYourOwnSpec() interface{} {
	return n.BringYourOwn
}

func (n *NodeSpec) BareMetalSpec() interface{} {
	return n.BareMetal
}

func (n *NodeSpec) OpenStackSpec() interface{} {
	return n.Openstack
}

// NodeCondition stores information about the node condition
type NodeCondition struct {
	Healthy     bool   `json:"healthy"`
	Description string `json:"description"`
}

// NodeStatus stores status information about a node.
type NodeStatus struct {
	Addresses NodeAddresses `json:"addresses"`
	CPU       int64         `json:"cpu"`
	Memory    string        `json:"memory"`
	Versions  *NodeVersions `json:"versions"`
	Condition NodeCondition `json:"condition"`
}

// NodeAddresses stores the IP addresses associated with a node
type NodeAddresses struct {
	Public  string `json:"public"`
	Private string `json:"private"`
}

// NodeVersions stores information about the node operating system
type NodeVersions struct {
	OS               string `json:"os,omitempty"`
	ContainerRuntime string `json:"container_runtime,omitempty"`
	Kubelet          string `json:"kubelet,omitempty"`
	KubeProxy        string `json:"kubeproxy,omitempty"`
	Kernel           string `json:"kernel,omitempty"`
}

// Node is the object representing a cluster node.
type Node struct {
	Metadata Metadata   `json:"metadata"`
	Spec     NodeSpec   `json:"spec"`
	Status   NodeStatus `json:"status,omitempty"`
}

// ContainerLinuxClusterSpec specifies container linux configuration for nodes - cluster wide
type ContainerLinuxClusterSpec struct {
	AutoUpdate bool `json:"auto_update"`
}

// ContainerLinuxNodeSpec specifies container linux configuration for individual nodes
type ContainerLinuxNodeSpec struct {
	Version string `json:"version"`
}

// CPU represents the CPU resources available on a node
type CPU struct {
	Cores     int     `json:"cores"`
	Frequency float64 `json:"frequency"`
}

// DigitialoceanDatacenterSpec specifies a data center of digital ocean.
type DigitialoceanDatacenterSpec struct {
	Region string `json:"region"`
}

// BringYourOwnDatacenterSpec specifies a data center with bring-your-own nodes.
type BringYourOwnDatacenterSpec struct {
}

// AWSDatacenterSpec specifies a data center of Amazon Web Services.
type AWSDatacenterSpec struct {
	Region string `json:"region"`
}

// BareMetalDatacenterSpec specifies a generic bare metal datacenter.
type BareMetalDatacenterSpec struct {
}

// OpenstackDatacenterSpec specifies a generic bare metal datacenter.
type OpenstackDatacenterSpec struct {
	AvailabilityZone string `json:"availability_zone"`
	AuthURL          string `json:"auth_url"`
}

// DatacenterSpec specifies the data for a datacenter.
type DatacenterSpec struct {
	Country      string                       `json:"country,omitempty"`
	Location     string                       `json:"location,omitempty"`
	Provider     string                       `json:"provider,omitempty"`
	Digitalocean *DigitialoceanDatacenterSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnDatacenterSpec  `json:"bringyourown,omitempty"`
	AWS          *AWSDatacenterSpec           `json:"aws,omitempty"`
	BareMetal    *BareMetalDatacenterSpec     `json:"baremetal,omitempty"`
	Openstack    *OpenstackDatacenterSpec     `json:"openstack,omitempty"`
}

// Datacenter is the object representing a Kubernetes infra datacenter.
type Datacenter struct {
	Metadata Metadata       `json:"metadata"`
	Spec     DatacenterSpec `json:"spec"`
	Seed     bool           `json:"seed,omitempty"`
}

// MasterVersion is the object representing a Kubernetes Master version.
type MasterVersion struct {
	Name                         string            `yaml:"name"`
	ID                           string            `yaml:"id"`
	Default                      bool              `yaml:"default"`
	AllowedNodeVersions          []string          `yaml:"allowedNodeVersions"`
	EtcdOperatorDeploymentYaml   string            `yaml:"etcdOperatorDeploymentYaml"`
	EtcdClusterYaml              string            `yaml:"etcdClusterYaml"`
	ApiserverDeploymentYaml      string            `yaml:"apiserverDeploymentYaml"`
	ControllerDeploymentYaml     string            `yaml:"controllerDeploymentYaml"`
	SchedulerDeploymentYaml      string            `yaml:"schedulerDeploymentYaml"`
	NodeControllerDeploymentYaml string            `yaml:"nodeControllerDeploymentYaml"`
	AddonManagerDeploymentYaml   string            `yaml:"addonManagerDeploymentYaml"`
	Values                       map[string]string `yaml:"values"`
}

// NodeVersion is the object representing a Kubernetes Kubelet version.
type NodeVersion struct {
	Name, ID string
	Latest   bool
}

// MasterUpdate represents an update option for K8s master components
type MasterUpdate struct {
	From            string `yaml:"from"`
	To              string `yaml:"to"`
	Automatic       bool   `yaml:"automatic"`
	RollbackAllowed bool   `yaml:"rollbackAllowed"`
	Enabled         bool   `yaml:"enabled"`
	Visible         bool   `yaml:"visible"`
	Promote         bool   `yaml:"promote"`
}

// NodeUpdate represents an update option for K8s node components
type NodeUpdate struct {
	From, To                   string
	Automatic, RollbackAllowed bool
	Enabled                    bool
	Visible                    bool
	Promote                    bool
}
