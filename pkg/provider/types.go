/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"errors"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	PacketCloudProvider       = "packet"
	HetznerCloudProvider      = "hetzner"
	VSphereCloudProvider      = "vsphere"
	GCPCloudProvider          = "gcp"
	KubevirtCloudProvider     = "kubevirt"
	AlibabaCloudProvider      = "alibaba"

	DefaultSSHPort     = 22
	DefaultKubeletPort = 10250

	DefaultKubeconfigFieldPath = "kubeconfig"
)

// CloudProvider declares a set of methods for interacting with a cloud provider
type CloudProvider interface {
	InitializeCloudProvider(*kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	CleanUpCloudProvider(*kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error
	ValidateCloudSpec(spec kubermaticv1.CloudSpec) error
	ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error
}

// ClusterUpdater defines a function to persist an update to a cluster
type ClusterUpdater func(string, func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error)

// ClusterListOptions allows to set filters that will be applied to filter the result.
type ClusterListOptions struct {
	// ClusterSpecName gets the clusters with the given name in the spec
	ClusterSpecName string
}

// ClusterGetOptions allows to check the status of the cluster
type ClusterGetOptions struct {
	// CheckInitStatus if set to true will check if cluster is initialized. The call will return error if
	// not all cluster components are running
	CheckInitStatus bool
}

// SecretKeySelectorValueFunc is used to fetch the value of a config var. Do not build your own
// implementation, use SecretKeySelectorValueFuncFactory.
type SecretKeySelectorValueFunc func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)

func SecretKeySelectorValueFuncFactory(ctx context.Context, client ctrlruntimeclient.Client) SecretKeySelectorValueFunc {
	return func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
		if configVar == nil {
			return "", errors.New("configVar is nil")
		}
		if configVar.Name == "" {
			return "", errors.New("configVar.Name is empty")
		}
		if configVar.Namespace == "" {
			return "", errors.New("configVar.Namspace is empty")
		}
		if key == "" {
			return "", errors.New("key is empty")
		}

		secret := &corev1.Secret{}
		namespacedName := types.NamespacedName{Namespace: configVar.Namespace, Name: configVar.Name}
		if err := client.Get(ctx, namespacedName, secret); err != nil {
			return "", fmt.Errorf("failed to get secret %q: %v", namespacedName.String(), err)
		}

		if _, ok := secret.Data[key]; !ok {
			return "", fmt.Errorf("secret %q has no key %q", namespacedName.String(), key)
		}

		return string(secret.Data[key]), nil
	}
}

// ProjectGetOptions allows to check the status of the Project
type ProjectGetOptions struct {
	// IncludeUninitialized if set to true will skip the check if project is initialized. By default the call will return
	// an  error if not all project components are active
	IncludeUninitialized bool
}

// ProjectListOptions allows to set filters that will be applied to the result returned form List method
type ProjectListOptions struct {
	// ProjectName list only projects with the given name
	ProjectName string

	// OwnerUID list only project that belong to this user
	OwnerUID types.UID
}

// ClusterProvider declares the set of methods for interacting with clusters
// This provider is Project and RBAC compliant
type ClusterProvider interface {
	// New creates a brand new cluster that is bound to the given project
	New(project *kubermaticv1.Project, userInfo *UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// List gets all clusters that belong to the given project
	// If you want to filter the result please take a look at ClusterListOptions
	//
	// Note:
	// After we get the list of clusters we could try to get each cluster individually using unprivileged account to see if the user have read access,
	// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
	List(project *kubermaticv1.Project, options *ClusterListOptions) (*kubermaticv1.ClusterList, error)

	// ListAll gets all clusters for the seed
	ListAll() (*kubermaticv1.ClusterList, error)

	// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to
	Get(userInfo *UserInfo, clusterName string, options *ClusterGetOptions) (*kubermaticv1.Cluster, error)

	// Update updates a cluster
	Update(project *kubermaticv1.Project, userInfo *UserInfo, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// Delete deletes the given cluster
	Delete(userInfo *UserInfo, clusterName string) error

	// GetAdminKubeconfigForCustomerCluster returns the admin kubeconfig for the given cluster
	GetAdminKubeconfigForCustomerCluster(cluster *kubermaticv1.Cluster) (*clientcmdapi.Config, error)

	// GetViewerKubeconfigForCustomerCluster returns the viewer kubeconfig for the given cluster
	GetViewerKubeconfigForCustomerCluster(cluster *kubermaticv1.Cluster) (*clientcmdapi.Config, error)

	// RevokeViewerKubeconfig revokes viewer token and kubeconfig
	RevokeViewerKubeconfig(c *kubermaticv1.Cluster) error

	// RevokeAdminKubeconfig revokes the viewer token and kubeconfig
	RevokeAdminKubeconfig(c *kubermaticv1.Cluster) error

	// GetAdminClientForCustomerCluster returns a client to interact with all resources in the given cluster
	//
	// Note that the client you will get has admin privileges
	GetAdminClientForCustomerCluster(*kubermaticv1.Cluster) (ctrlruntimeclient.Client, error)

	// GetClientForCustomerCluster returns a client to interact with all resources in the given cluster
	//
	// Note that the client doesn't use admin account instead it authn/authz as userInfo(email, group)
	GetClientForCustomerCluster(*UserInfo, *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error)

	// GetTokenForCustomerCluster returns a token for the given cluster with permissions granted to group that
	// user belongs to.
	GetTokenForCustomerCluster(userInfo *UserInfo, cluster *kubermaticv1.Cluster) (string, error)

	// IsCluster checks if cluster exist with the given name
	IsCluster(clusterName string) bool
}

// PrivilegedClusterProvider declares the set of methods for interacting with the seed clusters
// as an admin.
type PrivilegedClusterProvider interface {
	// GetSeedClusterAdminRuntimeClient returns a runtime client to interact with all resources in the seed cluster
	//
	// Note that the client you will get has admin privileges in the seed cluster
	GetSeedClusterAdminRuntimeClient() ctrlruntimeclient.Client

	// GetSeedClusterAdminClient returns a kubernetes client to interact with all resources in the seed cluster
	//
	// Note that the client you will get has admin privileges in the seed cluster
	GetSeedClusterAdminClient() kubernetes.Interface

	// GetUnsecured returns a cluster for the project and given name.
	//
	// Note that the admin privileges are used to get cluster
	GetUnsecured(project *kubermaticv1.Project, clusterName string, options *ClusterGetOptions) (*kubermaticv1.Cluster, error)

	// UpdateUnsecured updates a cluster.
	//
	// Note that the admin privileges are used to update cluster
	UpdateUnsecured(project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// DeleteUnsecured deletes a cluster.
	//
	// Note that the admin privileges are used to delete cluster
	DeleteUnsecured(cluster *kubermaticv1.Cluster) error

	// NewUnsecured creates a brand new cluster that is bound to the given project.
	//
	// Note that the admin privileges are used to create cluster
	NewUnsecured(project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, userEmail string) (*kubermaticv1.Cluster, error)
}

// SSHKeyListOptions allows to set filters that will be applied to filter the result.
type SSHKeyListOptions struct {
	// ClusterName gets the keys that are being used by the given cluster name
	ClusterName string
	// SSHKeyName gets the ssh keys with the given name in the spec
	SSHKeyName string
}

// SSHKeyProvider declares the set of methods for interacting with ssh keys
// This provider is Project and RBAC compliant
type SSHKeyProvider interface {
	// List gets a list of ssh keys, by default it will get all the keys that belong to the given project.
	// If you want to filter the result please take a look at SSHKeyListOptions
	//
	// Note:
	// After we get the list of the keys we could try to get each individually using unprivileged account to see if the user have read access,
	List(project *kubermaticv1.Project, options *SSHKeyListOptions) ([]*kubermaticv1.UserSSHKey, error)

	// Create creates a ssh key that belongs to the given project
	Create(userInfo *UserInfo, project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error)

	// Delete deletes the given ssh key
	Delete(userInfo *UserInfo, keyName string) error

	// Get returns a key with the given name
	Get(userInfo *UserInfo, keyName string) (*kubermaticv1.UserSSHKey, error)

	// Update simply updates the given key
	Update(userInfo *UserInfo, newKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error)
}

// SSHKeyProvider declares the set of methods for interacting with ssh keys and uses privileged account for it
type PrivilegedSSHKeyProvider interface {
	// GetUnsecured returns a key with the given name
	// This function is unsafe in a sense that it uses privileged account to get the ssh key
	GetUnsecured(keyName string) (*kubermaticv1.UserSSHKey, error)

	// UpdateUnsecured update a specific ssh key and returns the updated ssh key
	// This function is unsafe in a sense that it uses privileged account to update the ssh key
	UpdateUnsecured(sshKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error)

	// Create creates a ssh key that belongs to the given project
	// This function is unsafe in a sense that it uses privileged account to create the ssh key
	CreateUnsecured(project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error)

	// Delete deletes the given ssh key
	// This function is unsafe in a sense that it uses privileged account to delete the ssh key
	DeleteUnsecured(keyName string) error
}

// UserProvider declares the set of methods for interacting with kubermatic users
type UserProvider interface {
	UserByEmail(email string) (*kubermaticv1.User, error)
	CreateUser(id, name, email string) (*kubermaticv1.User, error)
	UpdateUser(user *kubermaticv1.User) (*kubermaticv1.User, error)
	UserByID(id string) (*kubermaticv1.User, error)
	AddUserTokenToBlacklist(user *kubermaticv1.User, token string, expiry apiv1.Time) error
	GetUserBlacklistTokens(user *kubermaticv1.User) ([]string, error)
	WatchUser() (watch.Interface, error)
}

// PrivilegedProjectProvider declares the set of method for interacting with kubermatic's project and uses privileged account for it
type PrivilegedProjectProvider interface {
	// GetUnsecured returns the project with the given name
	// This function is unsafe in a sense that it uses privileged account to get project with the given name
	GetUnsecured(projectInternalName string, options *ProjectGetOptions) (*kubermaticv1.Project, error)

	// DeleteUnsecured deletes any given project
	// This function is unsafe in a sense that it uses privileged account to delete project with the given name
	DeleteUnsecured(projectInternalName string) error

	// UpdateUnsecured update an existing project and returns it
	// This function is unsafe in a sense that it uses privileged account to update project
	UpdateUnsecured(project *kubermaticv1.Project) (*kubermaticv1.Project, error)
}

// ProjectProvider declares the set of method for interacting with kubermatic's project
type ProjectProvider interface {
	// New creates a brand new project in the system with the given name
	// Note that a user cannot own more than one project with the given name
	New(user *kubermaticv1.User, name string, labels map[string]string) (*kubermaticv1.Project, error)

	// Delete deletes the given project as the given user
	//
	// Note:
	// Before deletion project's status.phase is set to ProjectTerminating
	Delete(userInfo *UserInfo, projectInternalName string) error

	// Get returns the project with the given name
	Get(userInfo *UserInfo, projectInternalName string, options *ProjectGetOptions) (*kubermaticv1.Project, error)

	// Update update an existing project and returns it
	Update(userInfo *UserInfo, newProject *kubermaticv1.Project) (*kubermaticv1.Project, error)

	// List gets a list of projects, by default it returns all resources.
	// If you want to filter the result please set ProjectListOptions
	//
	// Note that the list is taken from the cache
	List(options *ProjectListOptions) ([]*kubermaticv1.Project, error)
}

// UserInfo represent authenticated user
type UserInfo struct {
	Email   string
	Group   string
	IsAdmin bool
}

// ProjectMemberListOptions allows to set filters that will be applied to filter the result.
type ProjectMemberListOptions struct {
	// MemberEmail set the email address of a member for the given project
	MemberEmail string

	// SkipPrivilegeVerification if set will not check if the user that wants to list members of the given project has sufficient privileges.
	SkipPrivilegeVerification bool
}

// ProjectMemberProvider binds users with projects
type ProjectMemberProvider interface {
	// Create creates a binding for the given member and the given project
	Create(userInfo *UserInfo, project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error)

	// List gets all members of the given project
	List(userInfo *UserInfo, project *kubermaticv1.Project, options *ProjectMemberListOptions) ([]*kubermaticv1.UserProjectBinding, error)

	// Delete deletes the given binding
	// Note:
	// Use List to get binding for the specific member of the given project
	Delete(userInfo *UserInfo, bindinName string) error

	// Update updates the given binding
	Update(userInfo *UserInfo, binding *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error)
}

// PrivilegedProjectMemberProvider binds users with projects and uses privileged account for it
type PrivilegedProjectMemberProvider interface {
	// CreateUnsecured creates a binding for the given member and the given project
	// This function is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error)

	// DeleteUnsecured deletes the given binding
	// Note:
	// Use List to get binding for the specific member of the given project
	// This function is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(bindingName string) error

	// UpdateUnsecured updates the given binding
	// This function is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(binding *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error)
}

// ProjectMemberMapper exposes method that knows how to map
// a user to a group for a project
type ProjectMemberMapper interface {
	// MapUserToGroup maps the given user to a specific group of the given project
	// This function is unsafe in a sense that it uses privileged account to list all members in the system
	MapUserToGroup(userEmail string, projectID string) (string, error)

	// MappingsFor returns the list of projects (bindings) for the given user
	// This function is unsafe in a sense that it uses privileged account to list all members in the system
	MappingsFor(userEmail string) ([]*kubermaticv1.UserProjectBinding, error)
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
	if spec.Packet != nil {
		clouds = append(clouds, PacketCloudProvider)
	}
	if spec.Hetzner != nil {
		clouds = append(clouds, HetznerCloudProvider)
	}
	if spec.VSphere != nil {
		clouds = append(clouds, VSphereCloudProvider)
	}
	if spec.GCP != nil {
		clouds = append(clouds, GCPCloudProvider)
	}
	if spec.Kubevirt != nil {
		clouds = append(clouds, KubevirtCloudProvider)
	}
	if spec.Alibaba != nil {
		clouds = append(clouds, AlibabaCloudProvider)
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
func DatacenterCloudProviderName(spec *kubermaticv1.DatacenterSpec) (string, error) {
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
	if spec.Packet != nil {
		clouds = append(clouds, PacketCloudProvider)
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
	if spec.GCP != nil {
		clouds = append(clouds, GCPCloudProvider)
	}
	if spec.Fake != nil {
		clouds = append(clouds, FakeCloudProvider)
	}
	if spec.Kubevirt != nil {
		clouds = append(clouds, KubevirtCloudProvider)
	}
	if spec.Alibaba != nil {
		clouds = append(clouds, AlibabaCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in DatacenterSpec: %+v", spec)
	}
	return clouds[0], nil
}

// ServiceAccountProvider declares the set of methods for interacting with kubermatic service account
type ServiceAccountProvider interface {
	Create(userInfo *UserInfo, project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error)
	List(userInfo *UserInfo, project *kubermaticv1.Project, options *ServiceAccountListOptions) ([]*kubermaticv1.User, error)
	Get(userInfo *UserInfo, name string, options *ServiceAccountGetOptions) (*kubermaticv1.User, error)
	Update(userInfo *UserInfo, serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error)
	Delete(userInfo *UserInfo, name string) error
}

// PrivilegedServiceAccountProvider declares the set of methods for interacting with kubermatic service account
type PrivilegedServiceAccountProvider interface {
	// CreateUnsecured creates a service accounts
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resources
	CreateUnsecured(project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error)

	// ListUnsecured gets all service accounts
	// If you want to filter the result please take a look at ServiceAccountListOptions
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(project *kubermaticv1.Project, options *ServiceAccountListOptions) ([]*kubermaticv1.User, error)

	// GetUnsecured gets all service accounts
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(name string, options *ServiceAccountGetOptions) (*kubermaticv1.User, error)

	// UpdateUnsecured gets all service accounts
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error)

	// DeleteUnsecured gets all service accounts
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(name string) error
}

// ServiceAccountGetOptions allows to set filters that will be applied to filter the get result.
type ServiceAccountGetOptions struct {
	// RemovePrefix when set to false will NOT remove "serviceaccount-" prefix from the ID
	//
	// Note:
	// By default the prefix IS removed, for example given "serviceaccount-7d4b5695vb" it returns "7d4b5695vb"
	RemovePrefix bool
}

// ServiceAccountListOptions allows to set filters that will be applied to filter the result.
type ServiceAccountListOptions struct {
	// ServiceAccountName list only service account with the given name
	ServiceAccountName string
}

// ServiceAccountTokenProvider declares the set of methods for interacting with kubermatic service account token
type ServiceAccountTokenProvider interface {
	Create(userInfo *UserInfo, sa *kubermaticv1.User, projectID, tokenName, tokenID, tokenData string) (*corev1.Secret, error)
	List(userInfo *UserInfo, project *kubermaticv1.Project, sa *kubermaticv1.User, options *ServiceAccountTokenListOptions) ([]*corev1.Secret, error)
	Get(userInfo *UserInfo, name string) (*corev1.Secret, error)
	Update(userInfo *UserInfo, secret *corev1.Secret) (*corev1.Secret, error)
	Delete(userInfo *UserInfo, name string) error
}

// ServiceAccountTokenListOptions allows to set filters that will be applied to filter the result.
type ServiceAccountTokenListOptions struct {
	// TokenID list only tokens with the specified name
	TokenID string

	// LabelSelector list only tokens with the specified label
	LabelSelector labels.Selector

	// TokenID list only tokens which belong to the SA
	ServiceAccountID string
}

// PrivilegedServiceAccountTokenProvider declares the set of method for interacting with kubermatic's sa's tokens and uses privileged account for it
type PrivilegedServiceAccountTokenProvider interface {
	// ListUnsecured returns all tokens in kubermatic namespace
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	// gets resources from the cache
	ListUnsecured(*ServiceAccountTokenListOptions) ([]*corev1.Secret, error)

	// CreateUnsecured creates a new token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(sa *kubermaticv1.User, projectID, tokenName, tokenID, tokenData string) (*corev1.Secret, error)

	// GetUnsecured gets the token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(name string) (*corev1.Secret, error)

	// UpdateUnsecured updates the token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	UpdateUnsecured(secret *corev1.Secret) (*corev1.Secret, error)

	// DeleteUnsecured deletes the token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(name string) error
}

// EventRecorderProvider allows to record events for objects that can be read using K8S API.
type EventRecorderProvider interface {
	// ClusterRecorderFor returns a event recorder that will be able to record event for objects in the cluster
	// referred by provided cluster config.
	ClusterRecorderFor(client kubernetes.Interface) record.EventRecorder
}

// AddonProvider declares the set of methods for interacting with addons
type AddonProvider interface {
	// New creates a new addon in the given cluster
	New(userInfo *UserInfo, cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) (*kubermaticv1.Addon, error)

	// List gets all addons that belong to the given cluster
	// If you want to filter the result please take a look at ClusterListOptions
	List(userInfo *UserInfo, cluster *kubermaticv1.Cluster) ([]*kubermaticv1.Addon, error)

	// Get returns the given addon
	Get(userInfo *UserInfo, cluster *kubermaticv1.Cluster, addonName string) (*kubermaticv1.Addon, error)

	// Update updates an addon
	Update(userInfo *UserInfo, cluster *kubermaticv1.Cluster, newAddon *kubermaticv1.Addon) (*kubermaticv1.Addon, error)

	// Delete deletes the given addon
	Delete(userInfo *UserInfo, cluster *kubermaticv1.Cluster, addonName string) error
}

type PrivilegedAddonProvider interface {
	// ListUnsecured gets all addons that belong to the given cluster
	// If you want to filter the result please take a look at ClusterListOptions
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(cluster *kubermaticv1.Cluster) ([]*kubermaticv1.Addon, error)

	// NewUnsecured creates a new addon in the given cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	NewUnsecured(cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) (*kubermaticv1.Addon, error)

	// GetUnsecured returns the given addon
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(cluster *kubermaticv1.Cluster, addonName string) (*kubermaticv1.Addon, error)

	// UpdateUnsecured updates an addon
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(cluster *kubermaticv1.Cluster, newAddon *kubermaticv1.Addon) (*kubermaticv1.Addon, error)

	// DeleteUnsecured deletes the given addon
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(cluster *kubermaticv1.Cluster, addonName string) error
}

type AddonConfigProvider interface {
	Get(addonName string) (*kubermaticv1.AddonConfig, error)
	List() (*kubermaticv1.AddonConfigList, error)
}

// SettingsProvider declares the set of methods for interacting global settings
type SettingsProvider interface {
	GetGlobalSettings() (*kubermaticv1.KubermaticSetting, error)
	UpdateGlobalSettings(userInfo *UserInfo, settings *kubermaticv1.KubermaticSetting) (*kubermaticv1.KubermaticSetting, error)
	WatchGlobalSettings() (watch.Interface, error)
}

// AdminProvider declares the set of methods for interacting with admin
type AdminProvider interface {
	SetAdmin(userInfo *UserInfo, email string, isAdmin bool) (*kubermaticv1.User, error)
	GetAdmins(userInfo *UserInfo) ([]kubermaticv1.User, error)
}

// PresetProvider declares the set of methods for interacting with presets
type PresetProvider interface {
	GetPresets(userInfo *UserInfo) ([]kubermaticv1.Preset, error)
	GetPreset(userInfo *UserInfo, name string) (*kubermaticv1.Preset, error)
	SetCloudCredentials(userInfo *UserInfo, presetName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error)
}

// AdmissionPluginsProvider declares the set of methods for interacting with admission plugins
type AdmissionPluginsProvider interface {
	List(userInfo *UserInfo) ([]kubermaticv1.AdmissionPlugin, error)
	Get(userInfo *UserInfo, name string) (*kubermaticv1.AdmissionPlugin, error)
	Delete(userInfo *UserInfo, name string) error
	Update(userInfo *UserInfo, admissionPlugin *kubermaticv1.AdmissionPlugin) (*kubermaticv1.AdmissionPlugin, error)
	ListPluginNamesFromVersion(fromVersion string) ([]string, error)
}

// ExternalClusterProvider declares the set of methods for interacting with external cluster
type ExternalClusterProvider interface {
	New(userInfo *UserInfo, project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)

	Get(userInfo *UserInfo, clusterName string) (*kubermaticv1.ExternalCluster, error)

	Delete(userInfo *UserInfo, cluster *kubermaticv1.ExternalCluster) error

	Update(userInfo *UserInfo, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)

	List(project *kubermaticv1.Project) (*kubermaticv1.ExternalClusterList, error)

	GenerateClient(cfg *clientcmdapi.Config) (ctrlruntimeclient.Client, error)

	GetClient(cluster *kubermaticv1.ExternalCluster) (ctrlruntimeclient.Client, error)

	CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticv1.ExternalCluster, kubeconfig string) error

	GetVersion(cluster *kubermaticv1.ExternalCluster) (*ksemver.Semver, error)

	ListNodes(cluster *kubermaticv1.ExternalCluster) (*corev1.NodeList, error)

	GetNode(cluster *kubermaticv1.ExternalCluster, nodeName string) (*corev1.Node, error)

	IsMetricServerAvailable(cluster *kubermaticv1.ExternalCluster) (bool, error)
}

// ExternalClusterProvider declares the set of methods for interacting with external cluster
type PrivilegedExternalClusterProvider interface {
	// NewUnsecured creates an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resources
	NewUnsecured(project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)

	// DeleteUnsecured deletes an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resources
	DeleteUnsecured(cluster *kubermaticv1.ExternalCluster) error

	// GetUnsecured gets an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	GetUnsecured(clusterName string) (*kubermaticv1.ExternalCluster, error)

	// UpdateUnsecured updates an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resources
	UpdateUnsecured(cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)
}

// ConstraintTemplateProvider declares the set of method for interacting with gatekeeper's constraint templates
type ConstraintTemplateProvider interface {
	// List gets a list of constraint templates, by default it returns all resources.
	//
	// Note that the list is taken from the cache
	List() (*kubermaticv1.ConstraintTemplateList, error)

	// Get gets the given constraint template
	Get(name string) (*kubermaticv1.ConstraintTemplate, error)

	// Create a Constraint Template
	Create(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error)

	// Update a Constraint Template
	Update(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error)

	// Delete a Constraint Template
	Delete(ct *kubermaticv1.ConstraintTemplate) error
}

// ConstraintProvider declares the set of method for interacting with constraints
type ConstraintProvider interface {
	// List gets a list of constraints
	//
	// Note that the list is taken from the cache
	List(userInfo *UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.ConstraintList, error)

	// Get gets the given constraints
	Get(userInfo *UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.Constraint, error)
}

// PrivilegedConstraintProvider declares the set of method for interacting with constraints using a privileged client
type PrivilegedConstraintProvider interface {
	// List gets a list of constraints
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(cluster *kubermaticv1.Cluster) (*kubermaticv1.ConstraintList, error)

	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	GetUnsecured(cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.Constraint, error)
}
