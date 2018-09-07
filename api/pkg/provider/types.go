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
	AzureCloudProvider        = "azure"
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

// ClusterUpdater defines a function to persist an update to a cluster
type ClusterUpdater func(string, func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error)

// CloudSpecProvider converts both a cloud spec and is able to create/retrieve nodes
// on a cloud provider.
type CloudSpecProvider interface {
	InitializeCloudProvider(*kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	CleanUpCloudProvider(*kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	DefaultCloudSpec(spec kubermaticv1.CloudSpec) error
	ValidateCloudSpec(spec kubermaticv1.CloudSpec) error
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

// ClusterListOptions allows to set filters that will be applied to filter the result.
type ClusterListOptions struct {
	// ClusterSpecName gets the clusters with the given name in the spec
	ClusterSpecName string

	// SortBy sorts the result by the given key name,
	// for example setting "metadata.creationTimestamp" will sort the result by creation timestamp
	SortBy string
}

// ClusterGetOptions allows to check the status of the cluster
type ClusterGetOptions struct {
	// CheckInitStatus if set to true will check if cluster is initialized. The call will return error if
	// not all cluster components are running
	CheckInitStatus bool
}

// ProjectGetOptions allows to check the status of the Project
type ProjectGetOptions struct {
	// IncludeUninitialized if set to true will skip the check if project is initialized. By default the call will return
	// an  error if not all project components are active
	IncludeUninitialized bool
}

// NewClusterProvider declares the set of methods for interacting with clusters
// This provider is Project and RBAC compliant
type NewClusterProvider interface {
	// New creates a brand new cluster that is bound to the given project
	New(project *kubermaticv1.Project, user *kubermaticv1.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error)

	// List gets all clusters that belong to the given project
	// If you want to filter the result please take a look at ClusterListOptions
	//
	// Note:
	// After we get the list of clusters we could try to get each cluster individually using unprivileged account to see if the user have read access,
	// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
	List(project *kubermaticv1.Project, options *ClusterListOptions) ([]*kubermaticv1.Cluster, error)

	// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to
	Get(user *kubermaticv1.User, project *kubermaticv1.Project, clusterName string, options *ClusterGetOptions) (*kubermaticv1.Cluster, error)

	// Update updates a cluster
	Update(user *kubermaticv1.User, project *kubermaticv1.Project, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// Delete deletes the given cluster
	Delete(user *kubermaticv1.User, project *kubermaticv1.Project, clusterName string) error

	// GetAdminKubeconfigForCustomerCluster returns the admin kubeconfig for the given cluster
	GetAdminKubeconfigForCustomerCluster(cluster *kubermaticv1.Cluster) (*clientcmdapi.Config, error)

	// GetMachineClientForCustomerCluster returns a client to interact with machine resources in the given cluster
	//
	// Note that the client you will get has admin privileges
	GetMachineClientForCustomerCluster(cluster *kubermaticv1.Cluster) (machineclientset.Interface, error)

	// GetClientForCustomerCluster returns a client to interact with the given cluster
	//
	// Note that the client you will get has admin privileges
	GetKubernetesClientForCustomerCluster(cluster *kubermaticv1.Cluster) (kubernetes.Interface, error)
}

// SSHKeyListOptions allows to set filters that will be applied to filter the result.
type SSHKeyListOptions struct {
	// ClusterName gets the keys that are being used by the given cluster name
	ClusterName string

	// SortBy sorts the result by the given key name,
	// for example setting "metadata.creationTimestamp" will sort the result by creation timestamp
	SortBy string
}

// NewSSHKeyProvider declares the set of methods for interacting with ssh keys
// This provider is Project and RBAC compliant
type NewSSHKeyProvider interface {
	// List gets a list of ssh keys, by default it will get all the keys that belong to the given project.
	// If you want to filter the result please take a look at SSHKeyListOptions
	//
	// Note:
	// After we get the list of the keys we could try to get each individually using unprivileged account to see if the user have read access,
	List(user *kubermaticv1.User, project *kubermaticv1.Project, options *SSHKeyListOptions) ([]*kubermaticv1.UserSSHKey, error)

	// Create creates a ssh key that belongs to the given project
	Create(user *kubermaticv1.User, project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error)

	// Delete deletes the given ssh key
	Delete(user *kubermaticv1.User, project *kubermaticv1.Project, keyName string) error

	// Get returns a key with the given name
	Get(user *kubermaticv1.User, project *kubermaticv1.Project, keyName string) (*kubermaticv1.UserSSHKey, error)

	// Update simply updates the given key
	Update(user *kubermaticv1.User, project *kubermaticv1.Project, newKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error)
}

// UserProvider declares the set of methods for interacting with kubermatic users
type UserProvider interface {
	UserByEmail(email string) (*kubermaticv1.User, error)
	UserByID(id string) (*kubermaticv1.User, error)
	CreateUser(id, name, email string) (*kubermaticv1.User, error)
	Update(*kubermaticv1.User) (*kubermaticv1.User, error)
	ListByProject(projectName string) ([]*kubermaticv1.User, error)
}

// ProjectProvider declares the set of method for interacting with kubermatic's project
type ProjectProvider interface {
	// New creates a brand new project in the system with the given name
	// Note that a user cannot own more than one project with the given name
	New(user *kubermaticv1.User, name string) (*kubermaticv1.Project, error)

	// Delete deletes the given project as the given user
	//
	// Note:
	// Before deletion project's status.phase is set to ProjectTerminating
	Delete(user *kubermaticv1.User, projectInternalName string) error

	// Get returns the project with the given name
	Get(user *kubermaticv1.User, projectInternalName string, options *ProjectGetOptions) (*kubermaticv1.Project, error)
}

// ClusterCloudProviderName returns the provider name for the given CloudSpec.
func ClusterCloudProviderName(spec kubermaticv1.CloudSpec) (string, error) {
	var clouds []string
	if spec.AWS != nil {
		clouds = append(clouds, AWSCloudProvider)
	}
	if spec.Azure != nil {
		clouds = append(clouds, AzureCloudProvider)
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
	if spec.Azure != nil {
		clouds = append(clouds, AzureCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in DatacenterSpec: %+v", spec)
	}
	return clouds[0], nil
}
