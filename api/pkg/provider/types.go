package provider

import (
	"errors"
	"fmt"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	// ErrNotFound tells that the requests resource was not found
	ErrNotFound = errors.New("the given resource was not found")
	// ErrAlreadyExists tells that the given resource already exists
	ErrAlreadyExists = errors.New("the given resource already exists")
)

// Constants defining known cloud providers.
const (
	FakeCloudProvider         = "fake"
	DigitaloceanCloudProvider = "digitalocean"
	BringYourOwnCloudProvider = "bringyourown"
	AWSCloudProvider          = "aws"
	OpenstackCloudProvider    = "openstack"
	HetznerCloudProvider      = "hetzner"
	VSphereCloudProvider      = "vsphere"

	DefaultSSHPort     = 22
	DefaultKubeletPort = 10250
)

// CloudProvider declares a set of methods for interacting with a cloud provider
type CloudProvider interface {
	CloudSpecProvider
}

// CloudSpecProvider converts both a cloud spec and is able to create/retrieve nodes
// on a cloud provider.
type CloudSpecProvider interface {
	InitializeCloudProvider(*kubermaticv1.CloudSpec, string) (*kubermaticv1.CloudSpec, error)
	ValidateCloudSpec(*kubermaticv1.CloudSpec) error
	CleanUpCloudProvider(*kubermaticv1.CloudSpec) error
}

// ClusterProvider declares the set of methods for storing and loading clusters.
type ClusterProvider interface {
	// NewCluster creates a cluster for the provided user using the given ClusterSpec
	NewCluster(user apiv1.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error)

	// Cluster return a Cluster struct, given the user and cluster.
	Cluster(user apiv1.User, name string) (*kubermaticv1.Cluster, error)

	// Clusters returns all clusters for a given user.
	Clusters(user apiv1.User) ([]*kubermaticv1.Cluster, error)

	// DeleteCluster deletes a Cluster from a user by it's name.
	DeleteCluster(user apiv1.User, name string) error

	// InitiateClusterUpgrade upgrades a Cluster to a specific version
	InitiateClusterUpgrade(user apiv1.User, name, version string) (*kubermaticv1.Cluster, error)

	// UpdateCluster updates a cluster
	UpdateCluster(user apiv1.User, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	GetClient(*kubermaticv1.Cluster) (kubernetes.Interface, error)

	GetMachineClient(*kubermaticv1.Cluster) (machineclientset.Interface, error)

	GetAdminKubeconfig(c *kubermaticv1.Cluster) (*clientcmdapi.Config, error)
}

// SSHKeyProvider declares the set of methods for interacting with ssh keys
type SSHKeyProvider interface {
	// SSHKey returns a ssh key by name
	SSHKey(user apiv1.User, name string) (*kubermaticv1.UserSSHKey, error)
	// SSHKeys returns the user ssh keys
	SSHKeys(user apiv1.User) ([]*kubermaticv1.UserSSHKey, error)
	// AssignSSHKeysToCluster assigns a ssh key to a cluster
	AssignSSHKeysToCluster(user apiv1.User, names []string, cluster string) error
	// ClusterSSHKeys returns the ssh keys of a cluster
	ClusterSSHKeys(user apiv1.User, cluster string) ([]*kubermaticv1.UserSSHKey, error)
	// CreateSSHKey creates a ssh key
	CreateSSHKey(name, pubkey string, user apiv1.User) (*kubermaticv1.UserSSHKey, error)
	// DeleteSSHKey deletes a ssh key
	DeleteSSHKey(name string, user apiv1.User) error
}

// UserProvider declares the set of methods for interacting with kubermatic users
type UserProvider interface {
	UserByEmail(email string) (*kubermaticv1.User, error)
	CreateUser(id, name, email string) (*kubermaticv1.User, error)
}

// ClusterCloudProviderName returns the provider name for the given CloudSpec.
func ClusterCloudProviderName(spec *kubermaticv1.CloudSpec) (string, error) {
	if spec == nil {
		return "", nil
	}

	var clouds []string
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
	if spec.Openstack != nil {
		clouds = append(clouds, OpenstackCloudProvider)
	}
	if spec.Hetzner != nil {
		clouds = append(clouds, HetznerCloudProvider)
	}
	if spec.VSphere != nil {
		clouds = append(clouds, VSphereCloudProvider)
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
func ClusterCloudProvider(cps map[string]CloudProvider, c *kubermaticv1.Cluster) (string, CloudProvider, error) {
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

// DatacenterCloudProviderName returns the provider name for the given Datacenter.
func DatacenterCloudProviderName(spec *DatacenterSpec) (string, error) {
	if spec == nil {
		return "", nil
	}
	var clouds []string
	if spec.BringYourOwn != nil {
		clouds = append(clouds, BringYourOwnCloudProvider)
	}
	if spec.Digitalocean != nil {
		clouds = append(clouds, DigitaloceanCloudProvider)
	}
	if spec.AWS != nil {
		clouds = append(clouds, AWSCloudProvider)
	}
	if spec.Openstack != nil {
		clouds = append(clouds, OpenstackCloudProvider)
	}
	if spec.Hetzner != nil {
		clouds = append(clouds, HetznerCloudProvider)
	}
	if spec.VSphere != nil {
		clouds = append(clouds, VSphereCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in DatacenterSpec: %+v", spec)
	}
	return clouds[0], nil
}
