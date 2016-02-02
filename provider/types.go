package provider

type Node interface {
	ID() string
	PublicIP() string
}

type NodeSpec interface{}

type NodeProvider interface {
	CreateNode(NodeSpec) error
}

type Metadata struct {
	Name string `json:"name"`
	Uid  string `json:"uid"`
}

type ClusterSpec struct {
	Dc string `json:"dc"`
}

type Cluster struct {
	Metadata Metadata    `json:"metadata"`
	Spec     ClusterSpec `json:"spec"`
}

type ClusterProvider interface {
	NewCluster(cluster string, spec ClusterSpec) (*Cluster, error)
	Cluster(dc string, cluster string) (*Cluster, error)
	Clusters(dc string) ([]*Cluster, error)
}
