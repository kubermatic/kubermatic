package provider

import (
	"errors"
	"fmt"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
)

var (
	// ErrNotFound tells that the requests resource was not found
	ErrNotFound = errors.New("the given resource was not found")
)

// Constants defining known cloud providers.
const (
	FakeCloudProvider         = "fake"
	DigitaloceanCloudProvider = "digitalocean"
	BringYourOwnCloudProvider = "bringyourown"
	AWSCloudProvider          = "aws"
	BareMetalCloudProvider    = "baremetal"
	OpenstackCloudProvider    = "openstack"

	DefaultSSHPort     = 22
	DefaultKubeletPort = 10250
)

// CloudProvider declares a set of methods for interacting with a cloud provider
type CloudProvider interface {
	CloudSpecProvider
	NodeClassProvider
}

// CloudSpecProvider converts both a cloud spec and is able to create/retrieve nodes
// on a cloud provider.
type CloudSpecProvider interface {
	InitializeCloudProvider(*kubermaticv1.CloudSpec, string) (*kubermaticv1.CloudSpec, error)
	ValidateCloudSpec(*kubermaticv1.CloudSpec) error
	CleanUpCloudProvider(*kubermaticv1.CloudSpec) error
}

// NodeClassProvider declares a set of methods to manage NodeClasses
type NodeClassProvider interface {
	ValidateNodeSpec(*kubermaticv1.CloudSpec, *api.NodeSpec) error
	CreateNodeClass(*kubermaticv1.Cluster, *api.NodeSpec, []*kubermaticv1.UserSSHKey, *api.MasterVersion) (*v1alpha1.NodeClass, error)
	NodeClassName(*api.NodeSpec) string
}

// DataProvider declares the set of methods for storing kubermatic data
type DataProvider interface {
	ClusterProvider
	SSHKeyProvider
	UserProvider
}

// ClusterProvider declares the set of methods for storing and loading clusters.
type ClusterProvider interface {
	// NewClusterWithCloud creates a cluster for the provided user using the given ClusterSpec
	NewClusterWithCloud(user auth.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error)

	// Cluster return a Cluster struct, given the user and cluster.
	Cluster(user auth.User, cluster string) (*kubermaticv1.Cluster, error)

	// Clusters returns all clusters for a given user.
	Clusters(user auth.User) ([]*kubermaticv1.Cluster, error)

	// DeleteCluster deletes a Cluster from a user by it's name.
	DeleteCluster(user auth.User, cluster string) error

	// InitiateClusterUpgrade upgrades a Cluster to a specific version
	InitiateClusterUpgrade(user auth.User, cluster, version string) (*kubermaticv1.Cluster, error)
}

// SSHKeyProvider declares the set of methods for interacting with ssh keys
type SSHKeyProvider interface {
	// AssignSSHKeysToCluster assigns a ssh key to a cluster
	AssignSSHKeysToCluster(user auth.User, names []string, cluster string) error
	// ClusterSSHKeys returns the ssh keys of a cluster
	ClusterSSHKeys(user auth.User, cluster string) ([]*kubermaticv1.UserSSHKey, error)
	// SSHKeys returns the user ssh keys
	SSHKeys(user auth.User) ([]*kubermaticv1.UserSSHKey, error)
	// SSHKey returns a ssh key by name
	SSHKey(user auth.User, name string) (*kubermaticv1.UserSSHKey, error)
	// CreateSSHKey creates a ssh key
	CreateSSHKey(name, pubkey string, user auth.User) (*kubermaticv1.UserSSHKey, error)
	// DeleteSSHKey deletes a ssh key
	DeleteSSHKey(name string, user auth.User) error
}

// UserProvider declares the set of methods for interacting with kubermatic users
type UserProvider interface {
	UserByEmail(email string) (*kubermaticv1.User, error)
	CreateUser(id, name, email string) (*kubermaticv1.User, error)
}

// ClusterCloudProvider returns the provider for the given cluster where
// one of Cluster.Spec.Cloud.* is set.
func ClusterCloudProvider(cps map[string]CloudProvider, c *kubermaticv1.Cluster) (string, CloudProvider, error) {
	name, err := ProviderName(c.Spec.Cloud)
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

type Specer interface {
	FakeSpec() interface{}
	DigitaloceanSpec() interface{}
	BringYourOwnSpec() interface{}
	AWSSpec() interface{}
	BareMetalSpec() interface{}
	OpenStackSpec() interface{}
	Type() string
}

func validateSpec(spec Specer) bool {
	if spec == nil {
		return false
	}
	specs := 0
	if spec.BringYourOwnSpec() != nil {
		specs++
	}
	if spec.DigitaloceanSpec() != nil {
		specs++
	}
	if spec.AWSSpec() != nil {
		specs++
	}
	if spec.FakeSpec() != nil {
		specs++
	}
	if spec.BareMetalSpec() != nil {
		specs++
	}
	if spec.OpenStackSpec() != nil {
		specs++
	}

	return specs == 1
}

// NodeCloudProviderName returns the provider name for the given node where
// one of NodeSpec.Cloud.* is set.
func ProviderName(spec Specer) (string, error) {

	switch {
	case spec.AWSSpec() != nil:
		return AWSCloudProvider, nil
	case spec.BringYourOwnSpec() != nil:
		return BringYourOwnCloudProvider, nil
	case spec.FakeSpec() != nil:
		return FakeCloudProvider, nil
	case spec.DigitaloceanSpec() != nil:
		return DigitaloceanCloudProvider, nil
	case spec.BareMetalSpec() != nil:
		return BareMetalCloudProvider, nil
	case spec.OpenStackSpec() != nil:
		return OpenstackCloudProvider, nil
	default:
		return "", fmt.Errorf("only one cloud provider can be set in %s: %+v", spec.Type(), spec)
	}
}
