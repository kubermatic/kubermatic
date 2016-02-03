package api

import (
	"time"
)

// Metadata is an object storing common metadata for persistable objects.
type Metadata struct {
	Name        string            `json:"name"`
	Revision    uint64            `json:"revision"`
	UID         string            `json:"uid"`
	Annotations map[string]string `json:"-"` // only for internal use
}

// NodeSpec specifies a node.
type NodeSpec struct {
	Type string `json:"type"`
	OS   string `json:"os"`
}

// NodeStatus stores status informations about a node.
type NodeStatus struct {
	Online bool `json:"online"`
}

// Node is the object representing a cluster node.
type Node struct {
	Metadata Metadata   `json:"metadata"`
	Spec     NodeSpec   `json:"spec"`
	Status   NodeStatus `json:"status,omitempty"`
}

// FakeCloudSpec specifies access data for a fake cloud.
type FakeCloudSpec struct {
	Token  string `json:"token,omitempty"`
	Region string `json:"region,omitempty"`
	Dc     string `json:"dc,omitempty"`
}

// DigitaloceanCloudSpec specifies access data to digital ocean.
type DigitaloceanCloudSpec struct {
	Token  string `json:"token,omitempty"`
	Region string `json:"region,omitempty"`
	Dc     string `json:"dc,omitempty"`
}

// LinodeCloudSpec specifies access data to digital ocean.
type LinodeCloudSpec struct {
	Token string `json:"token,omitempty"`
	Dc    string `json:"dc,omitempty"`
}

// CloudSpec mutually stores access data to a cloud provider.
type CloudSpec struct {
	Fake         *FakeCloudSpec         `json:"fake,omitempty"`
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	Linode       *LinodeCloudSpec       `json:"linode,omitempty"`
}

// ClusterHealth stores health information of a cluster.
type ClusterHealth struct {
	Timestamp  time.Time `json:"timestamp"`
	Apiserver  bool      `json:"apiserver"`
	Scheduler  bool      `json:"scheduler"`
	Controller bool      `json:"controller"`
	Etcd       bool      `json:"etcd"`
}

// ClusterStatus stores status informations about a cluster.
type ClusterStatus struct {
	Health ClusterHealth `json:"health"`
}

// ClusterSpec specifies the data for a new cluster.
type ClusterSpec struct {
	Cloud *CloudSpec `json:"cloud,omitempty"`
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// Cluster ist the object representating a cluster.
type Cluster struct {
	Metadata Metadata        `json:"metadata"`
	Spec     ClusterSpec     `json:"spec"`
	Address  *ClusterAddress `json:"address,omitempty"`
	Status   *ClusterStatus  `json:"status,omitempty"`
}
