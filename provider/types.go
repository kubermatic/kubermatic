package provider

import (
	"golang.org/x/net/context"

	"github.com/kubermatic/api"
)

// User represents an API user that is used for authentication.
type User struct {
	Name  string
	Roles map[string]struct{}
}

// CloudSpecProvider declares methods for converting a cloud spec to/from annotations.
type CloudSpecProvider interface {
	PrepareCloudSpec(*api.Cluster) error
	CreateAnnotations(*api.CloudSpec) (map[string]string, error)
	Cloud(annotations map[string]string) (*api.CloudSpec, error)
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
}

// KubernetesProvider declares the set of methods for interacting with a Kubernetes cluster.
type KubernetesProvider interface {
	NewCluster(user User, spec *api.ClusterSpec) (*api.Cluster, error)
	Cluster(user User, cluster string) (*api.Cluster, error)
	SetCloud(user User, cluster string, cloud *api.CloudSpec) (*api.Cluster, error)
	Clusters(user User) ([]*api.Cluster, error)
	DeleteCluster(user User, cluster string) error
}
