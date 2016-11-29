package api

import (
	"time"

	"github.com/kubermatic/api/provider/drivers"
	"github.com/kubermatic/api/provider/drivers/flag"
)

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

// Metadata is an object storing common metadata for persistable objects.
type Metadata struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
	UID      string `json:"uid,omitempty"`

	// private fields
	Annotations map[string]string `json:"-"`
	User        string            `json:"-"`
}

// NodeSpec mutually stores data of a cloud specific node.
type NodeSpec struct {
	// Dc is a reference to a datacenter which contains the information for the provider type.
	// It refers to github.com/kubermatic/provider.Datacenter.ExactName
	DcID string `json:"dc_id"`

	Patches flag.Flags `json:"patches_node"`
}

// FlannelNetworkSpec specifies a deployed flannel network.
type FlannelNetworkSpec struct {
	CIDR string `json:"-" yaml:"cidr"`
}

// NetworkSpec specifies the deployed network.
type NetworkSpec struct {
	Flannel FlannelNetworkSpec `json:"-" yaml:"flannel"`
}

// CloudSpec mutually stores access data to a cloud provider.
type CloudSpec struct {
	// Dc is a reference to a datacenter which contains the information for the provider type.
	// It refers to github.com/kubermatic/provider.Datacenter.ExactName
	DcID string `json:"dc_id"`

	// Patches respresent information used.
	Patches flag.Flags `json:"patches_credentials"`

	Network NetworkSpec `json:"-"`
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

// Bytes stores a byte slices and ecnodes as base64 in JSON.
type Bytes []byte

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
	HumanReadableName string     `json:"humanReadableName"`
	Cloud             *CloudSpec `json:"cloud,omitempty"`

	Dev bool `json:"-"` // a cluster used in development, compare --dev flag.
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	URL     string `json:"url"`
	EtcdURL string `json:"etcdURL"`
	Token   string `json:"token"`
}

// DatacenterSpec specifies the data for a datacenter.
type DatacenterSpec struct {
	// DriverType refers to a registered provider in github.com/kubermatic/provider/drivers
	DriverType string `json:"provider" yaml:"-"`

	Country  string `json:"country,omitempty" yaml:"country"`
	Location string `json:"location,omitempty" yaml:"location"`

	// Routing information
	Region    string `json:"-" yaml:"region,omitempty"`
	Zone      string `json:"-" yaml:"zone,omitempty"`
	ExactName string `json:"-" yaml:"ExactName"`

	CustomerPatches flag.Flags `json:"-" yaml:"customer-patches,omitempty"` // Patches that are applied to customer clusters.
	SeedPatches     flag.Flags `json:"-" yaml:"seed-patches,omitempty"`     // Patches that are applied to seed clusters.

	Private bool `json:"-" yaml:"private"`
}

/* Cluster, Datacenter and Node are all Serializeable units which represent the
state of a Cluster, Datacenter or node. The inner spec flieds are for creating
or bootstrapping a component. They contain user defined information which
weren't marshaled before. */

// Cluster is the object representating a cluster.
type Cluster struct {
	Metadata Metadata `json:"metadata"`

	Spec    ClusterSpec     `json:"spec"`
	Address *ClusterAddress `json:"address,omitempty"`
	Status  ClusterStatus   `json:"status,omitempty"`
}

// Node is the object representing a cluster node.
type Node struct {
	Metadata Metadata `json:"metadata"`

	Spec   NodeSpec               `json:"spec"`
	Status drivers.DriverInstance `json:"status,omitempty"`
}
