package provider

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/kubermatic/api"
)

// Constants defining known cloud providers.
const (
	FakeCloudProvider         = "fake"
	DigitaloceanCloudProvider = "digitalocean"
	BringYourOwnCloudProvider = "bringyourown"
	LinodeCloudProvider       = "linode"
)

// CloudSpecProvider declares methods for converting a cloud spec to/from annotations.
type CloudSpecProvider interface {
	CreateAnnotations(*api.CloudSpec) (map[string]string, error)
	Cloud(annotations map[string]string) (*api.CloudSpec, error)
}

// NodeProvider declares methods for creating/listing nodes.
type NodeProvider interface {
	CreateNode(context.Context, *api.Cluster, *api.NodeSpec) (*api.Node, error)
	Nodes(context.Context, *api.Cluster) ([]*api.Node, error)
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
	NewCluster(user, cluster string, spec *api.ClusterSpec) (*api.Cluster, error)
	Cluster(user, cluster string) (*api.Cluster, error)
	Clusters(user string) ([]*api.Cluster, error)
	DeleteCluster(user, cluster string) error

	Nodes(user, cluster string) ([]string, error)
}

// clusterCloudProviderName returns the provider name for the given cluster where
// one of Cluster.Spec.Cloud.* is set
func clusterCloudProviderName(c *api.Cluster) (string, error) {
	if c.Spec.Cloud == nil {
		return "", nil
	}

	switch cloud := c.Spec.Cloud; {
	case cloud.Fake != nil:
		return FakeCloudProvider, nil
	case cloud.Digitalocean != nil:
		return DigitaloceanCloudProvider, nil
	case cloud.BringYourOwn != nil:
		return BringYourOwnCloudProvider, nil
	case cloud.Linode != nil:
		return LinodeCloudProvider, nil
	}

	return "", nil
}

// ClusterCloudProvider returns the provider for the given cluster where
// one of Cluster.Spec.Cloud.* is set
func ClusterCloudProvider(cps map[string]CloudProvider, c *api.Cluster) (CloudProvider, error) {
	name, err := clusterCloudProviderName(c)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}

	cp, found := cps[name]
	if !found {
		return nil, fmt.Errorf("unsupported cloud provider %q", name)
	}

	return cp, nil
}
