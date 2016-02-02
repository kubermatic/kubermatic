package api

import (
	"net/url"
	"time"
)

type Metadata struct {
	Name     string `json:"name"`
	Revision uint64 `json:"revision"`
	Uid      string `json:"uid"`
}

type NodeSpec struct {
	Type string `json:"type"`
	OS   string `json:"os"`
}

type NodeStatus struct {
	Online bool `json:"online"`
}

type Node struct {
	Metadata Metadata   `json:"metadata"`
	Spec     NodeSpec   `json:"spec"`
	Status   NodeStatus `json:"status,omitempty"`
}

type DigitaloceanCloudSpec struct {
	Token string `json:"token,omitempty"`
	Dc    string `json:"dc,omitempty"`
}

type LinodeCloudSpec struct {
	Token string `json:"token,omitempty"`
	Dc    string `json:"dc,omitempty"`
}

type CloudSpec struct {
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	Linode       *LinodeCloudSpec       `json:"linode,omitempty"`
}

type ClusterHealth struct {
	Timestamp  time.Time `json:"timestamp"`
	Apiserver  bool      `json:"apiserver"`
	Scheduler  bool      `json:"scheduler"`
	Controller bool      `json:"controller"`
	Etcd       bool      `json:"etcd"`
}

type ClusterStatus struct {
	Health *ClusterHealth `json:"health,omitempty"`
}

type ClusterSpec struct {
	Dc     string         `json:"dc"`
	Cloud  *CloudSpec     `json:"cloud,omitempty"`
	Status *ClusterStatus `json:"cloud,omitempty"`
}

type ClusterAddress struct {
	Url   url.URL `json:"url"`
	Token string  `json:"token"`
}

type Cluster struct {
	Metadata Metadata       `json:"metadata"`
	Spec     ClusterSpec    `json:"spec"`
	Address  ClusterAddress `json:"address"`
}
