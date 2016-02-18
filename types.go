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
	Annotations map[string]string `json:"-"`
	User        string            `json:"-"`
}

// NodeSpec specifies a node.
type NodeSpec struct {
	Type string `json:"type"`
	OS   string `json:"os"`
}

// NodeStatus stores status informations about a node.
type NodeStatus struct {
	Online    bool              `json:"online"`
	Hostname  string            `json:"hostname"`
	Addresses map[string]string `json:"addresses"`
}

// Node is the object representing a cluster node.
type Node struct {
	Metadata Metadata   `json:"metadata"`
	Spec     NodeSpec   `json:"spec"`
	Status   NodeStatus `json:"status,omitempty"`
}

// LinodeCloudSpec specifies access data to digital ocean.
type LinodeCloudSpec struct {
	Token string `json:"token,omitempty"`
	Dc    string `json:"dc,omitempty"`
}

// DigitaloceanCloudSpec specifies access data to digital ocean.
type DigitaloceanCloudSpec struct {
	Region  string   `json:"region"`
	Token   string   `json:"token"`
	SSHKeys []string `json:"sshKeys,omitempty"`
}

// FakeCloudSpec specifies access data for a fake cloud.
type FakeCloudSpec struct {
	Token  string `json:"token,omitempty"`
	Region string `json:"region,omitempty"`
	Dc     string `json:"dc,omitempty"`
}

// CloudSpec mutually stores access data to a cloud provider.
type CloudSpec struct {
	Fake         *FakeCloudSpec         `json:"fake,omitempty"`
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	Linode       *LinodeCloudSpec       `json:"linode,omitempty"`
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

	// FailedClusterStatusPhase means that the cluster controller time out launching the cluster.
	FailedClusterStatusPhase ClusterPhase = "Failed"

	// RunningClusterStatusPhase means that the cluster is cluster is up and running.
	RunningClusterStatusPhase ClusterPhase = "Running"

	// PausedClusterStatusPhase means that the cluster was paused after the idle time.
	PausedClusterStatusPhase ClusterPhase = "Paused"

	// DeletingClusterStatusPhase means that the cluster controller is deleting the cluster.
	DeletingClusterStatusPhase ClusterPhase = "Deleting"
)

// ClusterStatus stores status informations about a cluster.
type ClusterStatus struct {
	LastTransitionTime time.Time      `json:"lastTransitionTime"`
	Phase              ClusterPhase   `json:"phase,omitempty"`
	Health             *ClusterHealth `json:"health,omitempty"`
}

// ClusterSpec specifies the data for a new cluster.
type ClusterSpec struct {
	Cloud             *CloudSpec `json:"cloud,omitempty"`
	HumanReadableName string     `json:"humanReadableName"`
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// Cluster is the object representating a cluster.
type Cluster struct {
	Metadata Metadata        `json:"metadata"`
	Spec     ClusterSpec     `json:"spec"`
	Address  *ClusterAddress `json:"address,omitempty"`
	Status   ClusterStatus   `json:"status,omitempty"`
}

// DatacenterSpec specifies the data for a datacenter.
type DatacenterSpec struct {
	Description string `json:"description,omitempty"`
	Country     string `json:"country,omitempty"`
	Provider    string `json:"provider,omitempty"`
}

// Datacenter is the object representing a Kubernetes infra datacenter.
type Datacenter struct {
	Metadata Metadata       `json:"metadata"`
	Spec     DatacenterSpec `json:"spec"`
}
