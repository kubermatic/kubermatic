package provider

type Node interface {
	ID() string
	PublicIP() string
}

type Cluster interface {
	ID() string
	Nodes() []Node
}

type NodeSpec interface{}

type NodeProvider interface {
	CreateNode(NodeSpec) error
}

type ClusterSpec interface{}

type ClusterProvider interface {
	NewCluster(ClusterSpec) (Cluster, error)
	Clusters() ([]Cluster, error)
}
