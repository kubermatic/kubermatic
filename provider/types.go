package provider

import (
	"fmt"

	"github.com/kubermatic/api"
)

// Constants defining known cloud providers.
const (
	DigitaloceanCloudProvider = iota
	LinodeCloudProvider
)

// CloudSpecProvider declares methods for converting a cloud spec to/from annotations.
type CloudSpecProvider interface {
	CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error)
	Cloud(annotations map[string]string) (*api.CloudSpec, error)
}

// NodeProvider declares methods for creating/listing nodes.
type NodeProvider interface {
	CreateNode(cluster *api.Cluster, spec *api.NodeSpec) (*api.Node, error)
	Nodes(cluster *api.Cluster) ([]*api.Node, error)
}

// CloudProvider converts both a cloud spec and is able to create/retrieve nodes
// on a cloud provider.
type CloudProvider interface {
	CloudSpecProvider
	NodeProvider
}

// KubernetesProvider declares the set of methods for interacting with a Kubernetes cluster.
type KubernetesProvider interface {
	NewCluster(cluster string, spec api.ClusterSpec) (*api.Cluster, error)
	Cluster(dc string, cluster string) (*api.Cluster, error)
	Clusters(dc string) ([]*api.Cluster, error)

	Nodes(dc string, cluster string) ([]string, error)
}

// ClusterCloudProvider returns the provider index for the given cluster.
func ClusterCloudProvider(c *api.Cluster) (int, error) {
	switch cloud := c.Spec.Cloud; {
	case cloud.Digitalocean != nil:
		return DigitaloceanCloudProvider, nil
	case cloud.Linode != nil:
		return LinodeCloudProvider, nil
	}

	return -1, fmt.Errorf("no cloud provider set")
}
