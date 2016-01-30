package cloud

type Node interface {
	ID() string
	PublicIP() string
}

type Cluster interface {
	ID() string
	Nodes() []Node
}

type ClusterSpec interface{}

type Provider interface {
	NewCluster(s ClusterSpec) (Cluster, error)
	Clusters() ([]Cluster, error)
}
