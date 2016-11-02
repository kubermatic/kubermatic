package api

import (
	"time"
)

// Metadata is an object storing common metadata for persistable objects.
type Metadata struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
	UID      string `json:"uid,omitempty"`

	// private fields
	// Annotations are kubernetes Annotations from the namespace.
	// Used to store data persistend.
	Annotations map[string]string `json:"-"`
	User        string            `json:"-"`
}

// DigitaloceanNodeSpec specifies a digital ocean node.
type DigitaloceanNodeSpec struct {
	// Type specifies the type of the image used to create the node.
	Type string `json:"type"`
	// Size is the size of the node.
	Size string `json:"size"`
	// SSHKeyNames represent the fingerprints of the keys.
	// Digitalocean utilizes the fingerprints to identify SSHKeys stored at digitalocean.
	SSHKeyNames []string `json:"sshKeys,omitempty"`
}

// BringYourOwnNodeSpec specifies a bring your own node
type BringYourOwnNodeSpec struct {
}

// FakeNodeSpec specifies a fake node.
type FakeNodeSpec struct {
	Type string `json:"type"`
	OS   string `json:"os"`
}

// NodeSpec mutually stores data of a cloud specific node.
type NodeSpec struct {
	// DC contains the name of the datacenter the node is located in.
	DC string `json:"dc"`

	Digitalocean *DigitaloceanNodeSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnNodeSpec `json:"bringyourown,omitempty"`
	Fake         *FakeNodeSpec         `json:"fake,omitempty"`
}

// NodeStatus stores status informations about a node.
type NodeStatus struct {
	Hostname  string            `json:"hostname"`
	Addresses map[string]string `json:"addresses"`
}

// Node is the object representing a cluster node.
type Node struct {
	Metadata Metadata   `json:"metadata"`
	Spec     NodeSpec   `json:"spec"`
	Status   NodeStatus `json:"status,omitempty"`
}

// DigitaloceanCloudSpec specifies access data to digital ocean.
type DigitaloceanCloudSpec struct {
	// Token grants access to the digitalocean api.
	Token string `json:"token"`
	// SSHKeys are real SSH keys ("id-alg .....") used by the cloud-init.
	SSHKeys []string `json:"sshKeys"`
}

// BringYourOwnCloudSpec specifies access data for a bring your own cluster.
type BringYourOwnCloudSpec struct {
	PrivateIntf   string  `json:"privateInterface"`
	ClientKeyCert KeyCert `json:"clientKeyCert"`
}

// FakeCloudSpec specifies access data for a fake cloud.
type FakeCloudSpec struct {
	Token string `json:"token,omitempty"`
}

// FlannelNetworkSpec specifies a deployed flannel network.
type FlannelNetworkSpec struct {
	// CIDR is the subnet used by flannel.
	// It need's to be provided in CIDR notation.
	// See RFC: 4632, eg. "127.1.0.0/16"
	CIDR string
}

// NetworkSpec specifies the deployed network.
type NetworkSpec struct {
	// FlannelNetworkSpec holds the informations needed for running a flannel network.
	Flannel FlannelNetworkSpec
}

// CloudSpec mutually stores access data to a cloud provider.
type CloudSpec struct {
	// The datacenter the users 'cloud' lives in.
	DC string `json:"dc"`
	// Network holds the network specifications.
	Network NetworkSpec `json:"-"`

	Fake         *FakeCloudSpec         `json:"fake,omitempty"`
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnCloudSpec `json:"bringyourown,omitempty"`
}

// ClusterHealthStatus stores health information of the components of a cluster.
type ClusterHealthStatus struct {
	Apiserver  bool   `json:"apiserver"`
	Scheduler  bool   `json:"scheduler"`
	Controller bool   `json:"controller"`
	Etcd       []bool `json:"etcd"`
}

// ClusterHealth stores health information of a cluster and the timestamp of the last change.
type ClusterHealth struct {
	ClusterHealthStatus `json:",inline"`
	LastTransitionTime  time.Time `json:"lastTransitionTime"`
}

// ClusterPhase is the life cycle phase of a cluster.
type ClusterPhase string

const (
	// UnknownClusterStatusPhase means that the phase label is missing on the Namespace.
	UnknownClusterStatusPhase ClusterPhase = "Unknown"

	// PendingClusterStatusPhase means that the cluster controller hasn't picked the cluster up.
	PendingClusterStatusPhase ClusterPhase = "Pending"

	// LaunchingClusterStatusPhase means that the cluster controller starts up the cluster.
	LaunchingClusterStatusPhase ClusterPhase = "Launching"

	// FailedClusterStatusPhase means that the cluster controller time out launching the cluster.
	FailedClusterStatusPhase ClusterPhase = "Failed"

	// RunningClusterStatusPhase means that the cluster is cluster is up and running.
	RunningClusterStatusPhase ClusterPhase = "Running"

	// PausedClusterStatusPhase means that the cluster was paused after the idle time.
	PausedClusterStatusPhase ClusterPhase = "Paused"

	// DeletingClusterStatusPhase means that the cluster controller is deleting the cluster.
	DeletingClusterStatusPhase ClusterPhase = "Deleting"
)

type (
	// Bytes stores a byte slices and ecnodes as base64 in JSON.
	Bytes []byte
)

// KeyCert is a pair of key and cert.
type KeyCert struct {
	Key  Bytes `json:"key"`
	Cert Bytes `json:"cert"`
}

// SecretKeyCert is a pair of key and cert where the key is not published to the API client.
type SecretKeyCert struct {
	Key  Bytes `json:"-"`
	Cert Bytes `json:"cert"`
}

// ClusterStatus stores status informations about a cluster.
type ClusterStatus struct {
	LastTransitionTime time.Time      `json:"lastTransitionTime"`
	Phase              ClusterPhase   `json:"phase,omitempty"`
	Health             *ClusterHealth `json:"health,omitempty"`

	RootCA       SecretKeyCert `json:"rootCA"`
	ApiserverSSH string        `json:"apiserverSSH"`
}

// ClusterSpec specifies the data for a new cluster.
type ClusterSpec struct {
	Cloud *CloudSpec `json:"cloud,omitempty"`
	// The name choosen by the user.
	HumanReadableName string `json:"humanReadableName"`

	Dev bool `json:"-"` // a cluster used in development, compare --dev flag.
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	URL     string `json:"url"`
	EtcdURL string `json:"etcdURL"`
	Token   string `json:"token"`
}

// Cluster is the object representating a cluster.
type Cluster struct {
	Metadata Metadata        `json:"metadata"`
	Spec     ClusterSpec     `json:"spec"`
	Address  *ClusterAddress `json:"address,omitempty"`
	Status   ClusterStatus   `json:"status,omitempty"`
}

// DigitialoceanDatacenterSpec specifies a data center of digital ocean.
type DigitialoceanDatacenterSpec struct {
	Region string `json:"region"`
}

// BringYourOwnDatacenterSpec specifies a data center with bring-your-own nodes.
type BringYourOwnDatacenterSpec struct {
}

// DatacenterSpec specifies the data for a datacenter.
type DatacenterSpec struct {
	Country      string                       `json:"country,omitempty"`
	Location     string                       `json:"location,omitempty"`
	Provider     string                       `json:"provider,omitempty"`
	Digitalocean *DigitialoceanDatacenterSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnDatacenterSpec  `json:"bringyourown,omitempty"`
}

// Datacenter is the object representing a Kubernetes infra datacenter.
type Datacenter struct {
	Metadata Metadata       `json:"metadata"`
	Spec     DatacenterSpec `json:"spec"`
	Seed     bool           `json:"seed,omitempty"`
}
