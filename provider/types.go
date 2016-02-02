package provider

import (
	"fmt"
	"github.com/kubermatic/api"
)

const (
	DigitaloceanCloudProvider = iota
	LinodeCloudProvider
)

type CloudSpecProvider interface {
	CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error)
	Cloud(annotations map[string]string) (*api.CloudSpec, error)
}

type NodeProvider interface {
	CreateNode(cluster *api.Cluster, spec *api.NodeSpec) (*api.Node, error)
	Nodes(cluster *api.Cluster) ([]*api.Node, error)
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

func ClusterCloudProvider(c *api.Cluster) (int, error) {
	switch cloud := c.Spec.Cloud; {
	case cloud.Digitalocean != nil:
		return DigitaloceanCloudProvider, nil
	case cloud.Linode != nil:
		return LinodeCloudProvider, nil
	}

	return -1, fmt.Errorf("no cloud provider set")
}
