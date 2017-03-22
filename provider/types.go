package provider

import (
	"fmt"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"golang.org/x/net/context"
)

// Constants defining known cloud providers.
const (
	FakeCloudProvider         = "fake"
	DigitaloceanCloudProvider = "digitalocean"
	BringYourOwnCloudProvider = "bringyourown"
	AWSCloudProvider          = "aws"
	BareMetalCloudProvider    = "baremetal"
)

// User represents an API user that is used for authentication.
type User struct {
	Name  string
	Roles map[string]struct{}
}

// CloudSpecProvider declares methods for converting a cloud spec to/from annotations.
type CloudSpecProvider interface {
	InitializeCloudSpec(*api.Cluster) error
	MarshalCloudSpec(*api.CloudSpec) (annotations map[string]string, err error)
	UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error)
}

// NodeProvider declares methods for creating/listing nodes.
type NodeProvider interface {
	CreateNodes(context.Context, *api.Cluster, *api.NodeSpec, int) ([]*api.Node, error)
	Nodes(context.Context, *api.Cluster) ([]*api.Node, error)
	DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error
}

// CloudProvider converts both a cloud spec and is able to create/retrieve nodes
// on a cloud provider.
type CloudProvider interface {
	CloudSpecProvider
	NodeProvider

	CleanUp(*api.Cluster) error
}

// KubernetesProvider declares the set of methods for interacting with a Kubernetes cluster.
type KubernetesProvider interface {
	// NewCluster creates a cluster for the provided user using the given ClusterSpec.
	// Deprecated in favor of NewClusterWithCloud
	NewCluster(user User, spec *api.ClusterSpec) (*api.Cluster, error)

	// NewClusterWithCloud creates a cluster for the provided user using the given ClusterSpec and given CloudSpec.
	NewClusterWithCloud(user User, spec *api.ClusterSpec, cloud *api.CloudSpec) (*api.Cluster, error)

	// Cluster return a Cluster struct, given the user and cluster.
	Cluster(user User, cluster string) (*api.Cluster, error)

	// SetCloud updates CloudSpec settings on the given cluster for the given user
	// Deprecated in favor of NewClusterWithCloud
	SetCloud(user User, cluster string, cloud *api.CloudSpec) (*api.Cluster, error)

	// Cluster returns all clusters for a given user.
	Clusters(user User) ([]*api.Cluster, error)

	// DeleteCluster deletes a Cluster from a user by it's name.
	DeleteCluster(user User, cluster string) error

	// CreateAddon creates an addon ThirdPartyResource
	CreateAddon(user User, cluster string, addonName string) (*extensions.ClusterAddon, error)

	// CreateNode creates a node ThirdPartyResource
	CreateNode(user User, cluster string, node *api.Node) (*extensions.ClNode, error)
}

// ClusterCloudProviderName returns the provider name for the given CloudSpec.
func ClusterCloudProviderName(spec *api.CloudSpec) (string, error) {
	if spec == nil {
		return "", nil
	}

	if spec.Name != "" {
		return spec.Name, nil
	}

	clouds := []string{}
	if spec.AWS != nil {
		clouds = append(clouds, AWSCloudProvider)
	}
	if spec.BringYourOwn != nil {
		clouds = append(clouds, BringYourOwnCloudProvider)
	}
	if spec.Digitalocean != nil {
		clouds = append(clouds, DigitaloceanCloudProvider)
	}
	if spec.Fake != nil {
		clouds = append(clouds, FakeCloudProvider)
	}
	if spec.BareMetal != nil {
		clouds = append(clouds, BareMetalCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in CloudSpec: %+v", spec)
	}
	return clouds[0], nil
}

// ClusterCloudProvider returns the provider for the given cluster where
// one of Cluster.Spec.Cloud.* is set.
func ClusterCloudProvider(cps map[string]CloudProvider, c *api.Cluster) (string, CloudProvider, error) {
	name, err := ClusterCloudProviderName(c.Spec.Cloud)
	if err != nil {
		return "", nil, err
	}
	if name == "" {
		return "", nil, nil
	}

	cp, found := cps[name]
	if !found {
		return "", nil, fmt.Errorf("unsupported cloud provider %q", name)
	}

	return name, cp, nil
}

// NodeCloudProviderName returns the provider name for the given node where
// one of NodeSpec.Cloud.* is set.
func NodeCloudProviderName(spec *api.NodeSpec) (string, error) {
	if spec == nil {
		return "", nil
	}
	clouds := []string{}
	if spec.BringYourOwn != nil {
		clouds = append(clouds, BringYourOwnCloudProvider)
	}
	if spec.Digitalocean != nil {
		clouds = append(clouds, DigitaloceanCloudProvider)
	}
	if spec.AWS != nil {
		clouds = append(clouds, AWSCloudProvider)
	}
	if spec.Fake != nil {
		clouds = append(clouds, FakeCloudProvider)
	}
	if spec.BareMetal != nil {
		clouds = append(clouds, BareMetalCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in NodeSpec: %+v", spec)
	}
	return clouds[0], nil
}

// DatacenterCloudProviderName returns the provider name for the given Datacenter.
func DatacenterCloudProviderName(spec *DatacenterSpec) (string, error) {
	if spec == nil {
		return "", nil
	}
	clouds := []string{}
	if spec.BringYourOwn != nil {
		clouds = append(clouds, BringYourOwnCloudProvider)
	}
	if spec.Digitalocean != nil {
		clouds = append(clouds, DigitaloceanCloudProvider)
	}
	if spec.AWS != nil {
		clouds = append(clouds, AWSCloudProvider)
	}
	if spec.BareMetal != nil {
		clouds = append(clouds, BareMetalCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in DatacenterSpec: %+v", spec)
	}
	return clouds[0], nil
}
