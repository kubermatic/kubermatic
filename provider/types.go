package provider

import (
	"fmt"

	"github.com/kubermatic/api"
)

// Constants defining known cloud providers.
const (
	FakeCloudProvider         = "fake"
	DigitaloceanCloudProvider = "digitalocean"
	LinodeCloudProvider       = "linode"
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
	Name() string

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

// clusterCloudProviderName returns the provider name for the given cluster where
// one of Cluster.Spec.Cloud.* is set
func clusterCloudProviderName(c *api.Cluster) (string, error) {
	if c.Spec.Cloud == nil {
		return "", fmt.Errorf("no cloud provider set")
	}

	switch cloud := c.Spec.Cloud; {
	case cloud.Fake != nil:
		return FakeCloudProvider, nil
	case cloud.Digitalocean != nil:
		return DigitaloceanCloudProvider, nil
	case cloud.Linode != nil:
		return LinodeCloudProvider, nil
	}

	return "", fmt.Errorf("no cloud provider set")
}

// ClusterCloudProvider returns the provider for the given cluster where
// one of Cluster.Spec.Cloud.* is set
func ClusterCloudProvider(cps map[string]CloudProvider, c *api.Cluster) (CloudProvider, error) {
	name, err := clusterCloudProviderName(c)
	if err != nil {
		return nil, err
	}

	cp, found := cps[name]
	if !found {
		return nil, fmt.Errorf("unsupported cloud provider %q", name)
	}

	return cp, nil
}
