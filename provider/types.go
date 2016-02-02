package provider

import (
	"github.com/kubermatic/api"
)

type CloudSpecProvider interface {
	CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error)
	Cloud(annotations map[string]string) (*api.CloudSpec, error)
}

type NodeProvider interface {
	CreateNode(cluster *api.Cluster, spec *api.NodeSpec) (*api.Node, error)
	Nodes(cluster *api.Cluster)
}

type CloudProvider interface {
	CloudSpecProvider
	NodeProvider
}

type KubernetesProvider interface {
	NewCluster(cluster string, spec api.ClusterSpec) (*api.Cluster, error)
	Cluster(dc string, cluster string) (*api.Cluster, error)
	Clusters(dc string) ([]*api.Cluster, error)

	Nodes(dc string, cluster string) ([]string, error)
}
