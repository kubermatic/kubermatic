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
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrNotFound tells that the requests resource was not found.
	ErrNotFound = errors.New("the given resource was not found")
	// ErrAlreadyExists tells that the given resource already exists.
	ErrAlreadyExists = errors.New("the given resource already exists")

	ErrNoKubermaticConfigurationFound      = errors.New("no KubermaticConfiguration resource found")
	ErrTooManyKubermaticConfigurationFound = errors.New("more than one KubermaticConfiguration resource found")
)

const (
	DefaultSSHPort     = 22
	DefaultKubeletPort = 10250

	DefaultKubeconfigFieldPath = "kubeconfig"
)

// KubermaticConfigurationGetter is a function to retrieve the currently relevant
// KubermaticConfiguration. That is the one in the same namespace as the
// running application (e.g. the seed-controller-manager). It's an error
// if there are none or more than one KubermaticConfiguration objects in
// a single namespace.
type KubermaticConfigurationGetter = func(ctx context.Context) (*kubermaticv1.KubermaticConfiguration, error)

// SeedGetter is a function to retrieve a single seed.
type SeedGetter = func() (*kubermaticv1.Seed, error)

// SeedsGetter is a function to retrieve a list of seeds.
type SeedsGetter = func() (map[string]*kubermaticv1.Seed, error)

// SeedKubeconfigGetter is used to fetch the kubeconfig for a given seed.
type SeedKubeconfigGetter = func(seed *kubermaticv1.Seed) (*rest.Config, error)

// SeedClientGetter is used to get a ctrlruntimeclient for a given seed.
type SeedClientGetter = func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error)

// ClusterProviderGetter is used to get a clusterProvider.
type ClusterProviderGetter = func(seed *kubermaticv1.Seed) (ClusterProvider, error)

// AddonProviderGetter is used to get an AddonProvider.
type AddonProviderGetter = func(seed *kubermaticv1.Seed) (AddonProvider, error)

// ConstraintProviderGetter is used to get a ConstraintProvider.
type ConstraintProviderGetter = func(seed *kubermaticv1.Seed) (ConstraintProvider, error)

// AlertmanagerProviderGetter is used to get an AlertmanagerProvider.
type AlertmanagerProviderGetter = func(seed *kubermaticv1.Seed) (AlertmanagerProvider, error)

// RuleGroupProviderGetter is used to get an RuleGroupProvider.
type RuleGroupProviderGetter = func(seed *kubermaticv1.Seed) (RuleGroupProvider, error)

// PrivilegedMLAAdminSettingProviderGetter is used to get a PrivilegedMLAAdminSettingProvider.
type PrivilegedMLAAdminSettingProviderGetter = func(seed *kubermaticv1.Seed) (PrivilegedMLAAdminSettingProvider, error)

// ClusterTemplateInstanceProviderGetter is used to get a ClusterTemplateInstanceProvider.
type ClusterTemplateInstanceProviderGetter = func(seed *kubermaticv1.Seed) (ClusterTemplateInstanceProvider, error)

// EtcdBackupConfigProviderGetter is used to get a EtcdBackupConfigProvider.
type EtcdBackupConfigProviderGetter = func(seed *kubermaticv1.Seed) (EtcdBackupConfigProvider, error)

// EtcdBackupConfigProjectProviderGetter is used to get a EtcdBackupConfigProjectProvider.
type EtcdBackupConfigProjectProviderGetter = func(seeds map[string]*kubermaticv1.Seed) (EtcdBackupConfigProjectProvider, error)

// EtcdRestoreProviderGetter is used to get a EtcdRestoreProvider.
type EtcdRestoreProviderGetter = func(seed *kubermaticv1.Seed) (EtcdRestoreProvider, error)

// EtcdRestoreProjectProviderGetter is used to get a EtcdRestoreProjectProvider.
type EtcdRestoreProjectProviderGetter = func(seeds map[string]*kubermaticv1.Seed) (EtcdRestoreProjectProvider, error)

// BackupCredentialsProviderGetter is used to get a BackupCredentialsProvider.
type BackupCredentialsProviderGetter = func(seed *kubermaticv1.Seed) (BackupCredentialsProvider, error)

// PrivilegedIPAMPoolProviderGetter is used to get a PrivilegedIPAMPoolProvider.
type PrivilegedIPAMPoolProviderGetter = func(seed *kubermaticv1.Seed) (PrivilegedIPAMPoolProvider, error)

// PrivilegedOperatingSystemProfileProviderGetter is used to get a PrivilegedOperatingSystemProfileProvider.
type PrivilegedOperatingSystemProfileProviderGetter = func(seed *kubermaticv1.Seed) (PrivilegedOperatingSystemProfileProvider, error)

// CloudProvider declares a set of methods for interacting with a cloud provider.
type CloudProvider interface {
	InitializeCloudProvider(context.Context, *kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	CleanUpCloudProvider(context.Context, *kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	DefaultCloudSpec(context.Context, *kubermaticv1.CloudSpec) error
	ValidateCloudSpec(context.Context, kubermaticv1.CloudSpec) error
	ValidateCloudSpecUpdate(ctx context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error
}

// ReconcilingCloudProvider is a cloud provider that can not just created resources
// once, but is capable of continuously reconciling and fixing any problems with them.
type ReconcilingCloudProvider interface {
	CloudProvider

	ReconcileCluster(context.Context, *kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
}

// ClusterUpdater defines a function to persist an update to a cluster.
type ClusterUpdater func(context.Context, string, func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error)

// ClusterListOptions allows to set filters that will be applied to filter the result.
type ClusterListOptions struct {
	// ClusterSpecName gets the clusters with the given name in the spec
	ClusterSpecName string
}

// ClusterGetOptions allows to check the status of the cluster.
type ClusterGetOptions struct {
	// CheckInitStatus if set to true will check if cluster is initialized. The call will return error if
	// not all cluster components are running
	CheckInitStatus bool
}

// SecretKeySelectorValueFunc is used to fetch the value of a config var. Do not build your own
// implementation, use SecretKeySelectorValueFuncFactory.
type SecretKeySelectorValueFunc func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)

func SecretKeySelectorValueFuncFactory(ctx context.Context, client ctrlruntimeclient.Reader) SecretKeySelectorValueFunc {
	return func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
		if configVar == nil {
			return "", errors.New("configVar is nil")
		}
		if configVar.Name == "" {
			return "", errors.New("configVar.Name is empty")
		}
		if configVar.Namespace == "" {
			return "", errors.New("configVar.Namespace is empty")
		}
		if key == "" {
			return "", errors.New("key is empty")
		}

		secret := &corev1.Secret{}
		namespacedName := types.NamespacedName{Namespace: configVar.Namespace, Name: configVar.Name}
		if err := client.Get(ctx, namespacedName, secret); err != nil {
			return "", fmt.Errorf("failed to get secret %q: %w", namespacedName.String(), err)
		}

		if _, ok := secret.Data[key]; !ok {
			return "", fmt.Errorf("secret %q has no key %q", namespacedName.String(), key)
		}

		return string(secret.Data[key]), nil
	}
}

// ProjectGetOptions allows to check the status of the Project.
type ProjectGetOptions struct {
	// IncludeUninitialized if set to true will skip the check if project is initialized. By default the call will return
	// an  error if not all project components are active
	IncludeUninitialized bool
}

// ProjectListOptions allows to set filters that will be applied to the result returned form List method.
type ProjectListOptions struct {
	// ProjectName list only projects with the given name
	ProjectName string
}

// ClusterProvider declares the set of methods for interacting with clusters
// This provider is Project and RBAC compliant.
type ClusterProvider interface {
	// New creates a brand new cluster that is bound to the given project
	New(ctx context.Context, project *kubermaticv1.Project, userInfo *UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// List gets all clusters that belong to the given project
	// If you want to filter the result please take a look at ClusterListOptions
	//
	// Note:
	// After we get the list of clusters we could try to get each cluster individually using unprivileged account to see if the user have read access,
	// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
	List(ctx context.Context, project *kubermaticv1.Project, options *ClusterListOptions) (*kubermaticv1.ClusterList, error)

	// ListAll gets all clusters for the seed
	ListAll(ctx context.Context, labelSelector labels.Selector) (*kubermaticv1.ClusterList, error)

	// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to
	Get(ctx context.Context, userInfo *UserInfo, clusterName string, options *ClusterGetOptions) (*kubermaticv1.Cluster, error)

	// Update updates a cluster
	Update(ctx context.Context, project *kubermaticv1.Project, userInfo *UserInfo, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// Delete deletes the given cluster
	Delete(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster) error

	// GetAdminKubeconfigForUserCluster returns the admin kubeconfig for the given cluster
	GetAdminKubeconfigForUserCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*clientcmdapi.Config, error)

	// GetViewerKubeconfigForUserCluster returns the viewer kubeconfig for the given cluster
	GetViewerKubeconfigForUserCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*clientcmdapi.Config, error)

	// RevokeViewerKubeconfig revokes viewer token and kubeconfig
	RevokeViewerKubeconfig(ctx context.Context, c *kubermaticv1.Cluster) error

	// RevokeAdminKubeconfig revokes the viewer token and kubeconfig
	RevokeAdminKubeconfig(ctx context.Context, c *kubermaticv1.Cluster) error

	// GetAdminClientForUserCluster returns a client to interact with all resources in the given cluster
	//
	// Note that the client you will get has admin privileges
	GetAdminClientForUserCluster(context.Context, *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error)

	// GetAdminK8sClientForUserCluster returns a k8s go client to interact with all resources in the given cluster
	//
	// Note that the client you will get has admin privileges
	GetAdminK8sClientForUserCluster(context.Context, *kubermaticv1.Cluster) (kubernetes.Interface, error)

	// GetAdminClientConfigForUserCluster returns a client config
	//
	// Note that the client you will get has admin privileges.
	GetAdminClientConfigForUserCluster(ctx context.Context, c *kubermaticv1.Cluster) (*restclient.Config, error)

	// GetClientForUserCluster returns a client to interact with all resources in the given cluster
	//
	// Note that the client doesn't use admin account instead it authn/authz as userInfo(email, group)
	GetClientForUserCluster(context.Context, *UserInfo, *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error)

	// GetTokenForUserCluster returns a token for the given cluster with permissions granted to group that
	// user belongs to.
	GetTokenForUserCluster(context.Context, *UserInfo, *kubermaticv1.Cluster) (string, error)

	// IsCluster checks if cluster exist with the given name
	IsCluster(ctx context.Context, clusterName string) bool

	// GetSeedName gets the seed name of the cluster
	GetSeedName() string
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
	GetUnsecured(ctx context.Context, project *kubermaticv1.Project, clusterName string, options *ClusterGetOptions) (*kubermaticv1.Cluster, error)

	// UpdateUnsecured updates a cluster.
	//
	// Note that the admin privileges are used to update cluster
	UpdateUnsecured(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// DeleteUnsecured deletes a cluster.
	//
	// Note that the admin privileges are used to delete cluster
	DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) error

	// NewUnsecured creates a brand new cluster that is bound to the given project.
	//
	// Note that the admin privileges are used to create cluster
	NewUnsecured(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, userEmail string) (*kubermaticv1.Cluster, error)
}

// SSHKeyListOptions allows to set filters that will be applied to filter the result.
type SSHKeyListOptions struct {
	// ClusterName gets the keys that are being used by the given cluster name
	ClusterName string
	// SSHKeyName gets the ssh keys with the given name in the spec
	SSHKeyName string
}

// SSHKeyProvider declares the set of methods for interacting with ssh keys
// This provider is Project and RBAC compliant.
type SSHKeyProvider interface {
	// List gets a list of ssh keys, by default it will get all the keys that belong to the given project.
	// If you want to filter the result please take a look at SSHKeyListOptions
	//
	// Note:
	// After we get the list of the keys we could try to get each individually using unprivileged account to see if the user have read access,
	List(ctx context.Context, project *kubermaticv1.Project, options *SSHKeyListOptions) ([]*kubermaticv1.UserSSHKey, error)

	// Create creates a ssh key that belongs to the given project
	Create(ctx context.Context, userInfo *UserInfo, project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error)

	// Delete deletes the given ssh key
	Delete(ctx context.Context, userInfo *UserInfo, keyName string) error

	// Get returns a key with the given name
	Get(ctx context.Context, userInfo *UserInfo, keyName string) (*kubermaticv1.UserSSHKey, error)

	// Update simply updates the given key
	Update(ctx context.Context, userInfo *UserInfo, newKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error)
}

// SSHKeyProvider declares the set of methods for interacting with ssh keys and uses privileged account for it.
type PrivilegedSSHKeyProvider interface {
	// GetUnsecured returns a key with the given name
	// This function is unsafe in a sense that it uses privileged account to get the ssh key
	GetUnsecured(ctx context.Context, keyName string) (*kubermaticv1.UserSSHKey, error)

	// UpdateUnsecured update a specific ssh key and returns the updated ssh key
	// This function is unsafe in a sense that it uses privileged account to update the ssh key
	UpdateUnsecured(ctx context.Context, sshKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error)

	// Create creates a ssh key that belongs to the given project
	// This function is unsafe in a sense that it uses privileged account to create the ssh key
	CreateUnsecured(ctx context.Context, project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error)

	// Delete deletes the given ssh key
	// This function is unsafe in a sense that it uses privileged account to delete the ssh key
	DeleteUnsecured(ctx context.Context, keyName string) error
}

// UserProvider declares the set of methods for interacting with kubermatic users.
type UserProvider interface {
	UserByEmail(ctx context.Context, email string) (*kubermaticv1.User, error)
	CreateUser(ctx context.Context, name, email string, groups []string) (*kubermaticv1.User, error)
	UpdateUser(ctx context.Context, user *kubermaticv1.User) (*kubermaticv1.User, error)
	UserByID(ctx context.Context, id string) (*kubermaticv1.User, error)
	InvalidateToken(ctx context.Context, user *kubermaticv1.User, token string, expiry apiv1.Time) error
	GetInvalidatedTokens(ctx context.Context, user *kubermaticv1.User) ([]string, error)
	List(ctx context.Context) ([]kubermaticv1.User, error)
}

// PrivilegedProjectProvider declares the set of method for interacting with kubermatic's project and uses privileged account for it.
type PrivilegedProjectProvider interface {
	// GetUnsecured returns the project with the given name
	// This function is unsafe in a sense that it uses privileged account to get project with the given name
	GetUnsecured(ctx context.Context, projectInternalName string, options *ProjectGetOptions) (*kubermaticv1.Project, error)

	// DeleteUnsecured deletes any given project
	// This function is unsafe in a sense that it uses privileged account to delete project with the given name
	DeleteUnsecured(ctx context.Context, projectInternalName string) error

	// UpdateUnsecured update an existing project and returns it
	// This function is unsafe in a sense that it uses privileged account to update project
	UpdateUnsecured(ctx context.Context, project *kubermaticv1.Project) (*kubermaticv1.Project, error)
}

// ProjectProvider declares the set of method for interacting with kubermatic's project.
type ProjectProvider interface {
	// New creates a brand new project in the system with the given name
	// Note that a user cannot own more than one project with the given name
	New(ctx context.Context, name string, labels map[string]string) (*kubermaticv1.Project, error)

	// Delete deletes the given project as the given user
	//
	// Note:
	// Before deletion project's status.phase is set to ProjectTerminating
	Delete(ctx context.Context, userInfo *UserInfo, projectInternalName string) error

	// Get returns the project with the given name
	Get(ctx context.Context, userInfo *UserInfo, projectInternalName string, options *ProjectGetOptions) (*kubermaticv1.Project, error)

	// Update update an existing project and returns it
	Update(ctx context.Context, userInfo *UserInfo, newProject *kubermaticv1.Project) (*kubermaticv1.Project, error)

	// List gets a list of projects, by default it returns all resources.
	// If you want to filter the result please set ProjectListOptions
	//
	// Note that the list is taken from the cache
	List(ctx context.Context, options *ProjectListOptions) ([]*kubermaticv1.Project, error)
}

// UserInfo represent authenticated user.
type UserInfo struct {
	Email   string
	Groups  []string
	Roles   sets.String
	IsAdmin bool
}

// ProjectMemberListOptions allows to set filters that will be applied to filter the result.
type ProjectMemberListOptions struct {
	// MemberEmail set the email address of a member for the given project
	MemberEmail string

	// SkipPrivilegeVerification if set will not check if the user that wants to list members of the given project has sufficient privileges.
	SkipPrivilegeVerification bool
}

// ProjectMemberProvider binds users with projects.
type ProjectMemberProvider interface {
	// Create creates a binding for the given member and the given project
	Create(ctx context.Context, userInfo *UserInfo, project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error)

	// List gets all members of the given project
	List(ctx context.Context, userInfo *UserInfo, project *kubermaticv1.Project, options *ProjectMemberListOptions) ([]*kubermaticv1.UserProjectBinding, error)

	// Delete deletes the given binding
	// Note:
	// Use List to get binding for the specific member of the given project
	Delete(ctx context.Context, userInfo *UserInfo, bindinName string) error

	// Update updates the given binding
	Update(ctx context.Context, userInfo *UserInfo, binding *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error)
}

// PrivilegedProjectMemberProvider binds users with projects and uses privileged account for it.
type PrivilegedProjectMemberProvider interface {
	// CreateUnsecured creates a binding for the given member and the given project
	// This function is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error)

	// CreateUnsecuredForServiceAccount creates a binding for the given service account and the given project
	// This function is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecuredForServiceAccount(ctx context.Context, project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error)

	// DeleteUnsecured deletes the given binding
	// Note:
	// Use List to get binding for the specific member of the given project
	// This function is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, bindingName string) error

	// UpdateUnsecured updates the given binding
	// This function is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, binding *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error)
}

// ProjectMemberMapper exposes method that knows how to map
// a user to a group for a project.
type ProjectMemberMapper interface {
	// MapUserToGroup maps the given user to a specific group of the given project
	// This function is unsafe in a sense that it uses privileged account to list all members in the system
	MapUserToGroup(ctx context.Context, userEmail string, projectID string) (string, error)

	// MappingsFor returns the list of projects (bindings) for the given user
	// This function is unsafe in a sense that it uses privileged account to list all members in the system
	MappingsFor(ctx context.Context, userEmail string) ([]*kubermaticv1.UserProjectBinding, error)

	// GroupMappingsFor returns the list of projects (bindings) for the given set of groups
	// This function is unsafe in a sense that it uses privileged account to list all members in the system.
	GroupMappingsFor(ctx context.Context, userGroups []string) ([]*kubermaticv1.GroupProjectBinding, error)

	// MapUserToRoles returns the roles of the user in the project. It searches across the user project bindings and the group
	// project bindings for the user and returns the roles.
	// This function is unsafe in a sense that it uses privileged account to list all userProjectBindings and groupProjectBindings in the system.
	MapUserToRoles(ctx context.Context, user *kubermaticv1.User, projectID string) (sets.String, error)

	// MapUserToGroups returns the groups of the user in the project. It combines identity provider groups with
	// group from UserProjectBinding (if exists). Groups returned by this function are suffixed with project's ID to
	// avoid leaking permissions among projects having binding with the same group but different roles.
	// This function is unsafe in a sense that it uses privileged account to list all userProjectBindings in the system.
	MapUserToGroups(ctx context.Context, user *kubermaticv1.User, projectID string) (sets.String, error)
}

// ClusterCloudProvider returns the provider for the given cluster where
// one of Cluster.Spec.Cloud.* is set.
func ClusterCloudProvider(cps map[string]CloudProvider, c *kubermaticv1.Cluster) (string, CloudProvider, error) {
	name, err := kubermaticv1helper.ClusterCloudProviderName(c.Spec.Cloud)
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

// ServiceAccountProvider declares the set of methods for interacting with kubermatic service account.
type ServiceAccountProvider interface {
	CreateProjectServiceAccount(ctx context.Context, userInfo *UserInfo, project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error)
	ListProjectServiceAccount(ctx context.Context, userInfo *UserInfo, project *kubermaticv1.Project, options *ServiceAccountListOptions) ([]*kubermaticv1.User, error)
	GetProjectServiceAccount(ctx context.Context, userInfo *UserInfo, name string, options *ServiceAccountGetOptions) (*kubermaticv1.User, error)
	UpdateProjectServiceAccount(ctx context.Context, userInfo *UserInfo, serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error)
	DeleteProjectServiceAccount(ctx context.Context, userInfo *UserInfo, name string) error
}

// PrivilegedServiceAccountProvider declares the set of methods for interacting with kubermatic service account.
type PrivilegedServiceAccountProvider interface {
	// CreateUnsecuredProjectServiceAccount creates a project service account
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resources
	CreateUnsecuredProjectServiceAccount(ctx context.Context, project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error)

	// ListUnsecuredProjectServiceAccount gets all project service accounts
	// If you want to filter the result please take a look at ServiceAccountListOptions
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecuredProjectServiceAccount(ctx context.Context, project *kubermaticv1.Project, options *ServiceAccountListOptions) ([]*kubermaticv1.User, error)

	// GetUnsecuredProjectServiceAccount get the project service account
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecuredProjectServiceAccount(ctx context.Context, name string, options *ServiceAccountGetOptions) (*kubermaticv1.User, error)

	// UpdateUnsecuredProjectServiceAccount updates the project service account
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecuredProjectServiceAccount(ctx context.Context, serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error)

	// DeleteUnsecuredProjectServiceAccount deletes the project service account
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecuredProjectServiceAccount(ctx context.Context, name string) error
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

// ServiceAccountTokenProvider declares the set of methods for interacting with kubermatic service account token.
type ServiceAccountTokenProvider interface {
	Create(ctx context.Context, userInfo *UserInfo, sa *kubermaticv1.User, projectID, tokenName, tokenID, tokenData string) (*corev1.Secret, error)
	List(ctx context.Context, userInfo *UserInfo, project *kubermaticv1.Project, sa *kubermaticv1.User, options *ServiceAccountTokenListOptions) ([]*corev1.Secret, error)
	Get(ctx context.Context, userInfo *UserInfo, name string) (*corev1.Secret, error)
	Update(ctx context.Context, userInfo *UserInfo, secret *corev1.Secret) (*corev1.Secret, error)
	Delete(ctx context.Context, userInfo *UserInfo, name string) error
}

// ServiceAccountTokenListOptions allows to set filters that will be applied to filter the result.
type ServiceAccountTokenListOptions struct {
	// TokenID list only tokens with the specified ID
	TokenID string

	// TokenName list only tokens with the specified name
	TokenName string

	// LabelSelector list only tokens with the specified label
	LabelSelector labels.Selector

	// TokenID list only tokens which belong to the SA
	ServiceAccountID string
}

// PrivilegedServiceAccountTokenProvider declares the set of method for interacting with kubermatic's sa's tokens and uses privileged account for it.
type PrivilegedServiceAccountTokenProvider interface {
	// ListUnsecured returns all tokens in kubermatic namespace
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	// gets resources from the cache
	ListUnsecured(context.Context, *ServiceAccountTokenListOptions) ([]*corev1.Secret, error)

	// CreateUnsecured creates a new token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, sa *kubermaticv1.User, projectID, tokenName, tokenID, tokenData string) (*corev1.Secret, error)

	// GetUnsecured gets the token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, name string) (*corev1.Secret, error)

	// UpdateUnsecured updates the token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	UpdateUnsecured(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error)

	// DeleteUnsecured deletes the token
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, name string) error
}

// EventRecorderProvider allows to record events for objects that can be read using K8S API.
type EventRecorderProvider interface {
	// ClusterRecorderFor returns a event recorder that will be able to record event for objects in the cluster
	// referred by provided cluster config.
	ClusterRecorderFor(client kubernetes.Interface) record.EventRecorder
}

// AddonProvider declares the set of methods for interacting with addons.
type AddonProvider interface {
	// New creates a new addon in the given cluster
	New(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) (*kubermaticv1.Addon, error)

	// List gets all addons that belong to the given cluster
	// If you want to filter the result please take a look at ClusterListOptions
	List(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster) ([]*kubermaticv1.Addon, error)

	// Get returns the given addon
	Get(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, addonName string) (*kubermaticv1.Addon, error)

	// Update updates an addon
	Update(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, newAddon *kubermaticv1.Addon) (*kubermaticv1.Addon, error)

	// Delete deletes the given addon
	Delete(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, addonName string) error
}

type PrivilegedAddonProvider interface {
	// ListUnsecured gets all addons that belong to the given cluster
	// If you want to filter the result please take a look at ClusterListOptions
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) ([]*kubermaticv1.Addon, error)

	// NewUnsecured creates a new addon in the given cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	NewUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) (*kubermaticv1.Addon, error)

	// GetUnsecured returns the given addon
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, addonName string) (*kubermaticv1.Addon, error)

	// UpdateUnsecured updates an addon
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, newAddon *kubermaticv1.Addon) (*kubermaticv1.Addon, error)

	// DeleteUnsecured deletes the given addon
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, addonName string) error
}

type AddonConfigProvider interface {
	Get(ctx context.Context, addonName string) (*kubermaticv1.AddonConfig, error)
	List(ctx context.Context) (*kubermaticv1.AddonConfigList, error)
}

// SettingsProvider declares the set of methods for interacting global settings.
type SettingsProvider interface {
	GetGlobalSettings(ctx context.Context) (*kubermaticv1.KubermaticSetting, error)
	UpdateGlobalSettings(ctx context.Context, userInfo *UserInfo, settings *kubermaticv1.KubermaticSetting) (*kubermaticv1.KubermaticSetting, error)
}

// AdminProvider declares the set of methods for interacting with admin.
type AdminProvider interface {
	SetAdmin(ctx context.Context, userInfo *UserInfo, email string, isAdmin bool) (*kubermaticv1.User, error)
	GetAdmins(ctx context.Context, userInfo *UserInfo) ([]kubermaticv1.User, error)
}

// PresetProvider declares the set of methods for interacting with presets.
type PresetProvider interface {
	CreatePreset(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error)
	UpdatePreset(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error)
	GetPresets(ctx context.Context, userInfo *UserInfo, projectID *string) ([]kubermaticv1.Preset, error)
	GetPreset(ctx context.Context, userInfo *UserInfo, projectID *string, name string) (*kubermaticv1.Preset, error)
	DeletePreset(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error)
	SetCloudCredentials(ctx context.Context, userInfo *UserInfo, projectID string, presetName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error)
}

// AdmissionPluginsProvider declares the set of methods for interacting with admission plugins.
type AdmissionPluginsProvider interface {
	List(ctx context.Context, userInfo *UserInfo) ([]kubermaticv1.AdmissionPlugin, error)
	Get(ctx context.Context, userInfo *UserInfo, name string) (*kubermaticv1.AdmissionPlugin, error)
	Delete(ctx context.Context, userInfo *UserInfo, name string) error
	Update(ctx context.Context, userInfo *UserInfo, admissionPlugin *kubermaticv1.AdmissionPlugin) (*kubermaticv1.AdmissionPlugin, error)
	ListPluginNamesFromVersion(ctx context.Context, fromVersion string) ([]string, error)
}

// ExternalClusterProvider declares the set of methods for interacting with external cluster.
type ExternalClusterProvider interface {
	New(ctx context.Context, userInfo *UserInfo, project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)

	Get(ctx context.Context, userInfo *UserInfo, clusterName string) (*kubermaticv1.ExternalCluster, error)

	Delete(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.ExternalCluster) error

	Update(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)

	List(ctx context.Context, project *kubermaticv1.Project) (*kubermaticv1.ExternalClusterList, error)

	GenerateClient(cfg *clientcmdapi.Config) (ctrlruntimeclient.Client, error)

	GetClient(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (ctrlruntimeclient.Client, error)

	ValidateKubeconfig(ctx context.Context, kubeconfig []byte) error

	CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticv1.ExternalCluster, kubeconfig []byte) error

	CreateOrUpdateCredentialSecretForCluster(ctx context.Context, cloud *apiv2.ExternalClusterCloudSpec, projectID, clusterID string) (*providerconfig.GlobalSecretKeySelector, error)

	CreateKubeOneClusterNamespace(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster) error

	CreateOrUpdateKubeOneSSHSecret(ctx context.Context, sshKey apiv2.KubeOneSSHKey, externalCluster *kubermaticv1.ExternalCluster) error

	CreateOrUpdateKubeOneManifestSecret(ctx context.Context, manifest string, externalCluster *kubermaticv1.ExternalCluster) error

	CreateOrUpdateKubeOneCredentialSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, externalCluster *kubermaticv1.ExternalCluster) error

	GetVersion(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*ksemver.Semver, error)

	VersionsEndpoint(ctx context.Context, configGetter KubermaticConfigurationGetter, providerType kubermaticv1.ExternalClusterProviderType) ([]apiv1.MasterVersion, error)

	ListNodes(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*corev1.NodeList, error)

	GetNode(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodeName string) (*corev1.Node, error)

	GetProviderPoolNodes(ctx context.Context, cluster *kubermaticv1.ExternalCluster, providerNodeLabel, providerNodePoolName string) ([]corev1.Node, error)

	IsMetricServerAvailable(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (bool, error)
}

// ExternalClusterProvider declares the set of methods for interacting with external cluster.
type PrivilegedExternalClusterProvider interface {
	// NewUnsecured creates an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resources
	NewUnsecured(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)

	// DeleteUnsecured deletes an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resources
	DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error

	// GetUnsecured gets an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	GetUnsecured(ctx context.Context, clusterName string) (*kubermaticv1.ExternalCluster, error)

	// UpdateUnsecured updates an external cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resources
	UpdateUnsecured(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error)

	// GetMasterClient returns master client
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resources
	GetMasterClient() ctrlruntimeclient.Client
}

// ConstraintTemplateProvider declares the set of method for interacting with gatekeeper's constraint templates.
type ConstraintTemplateProvider interface {
	// List gets a list of constraint templates, by default it returns all resources.
	//
	// Note that the list is taken from the cache
	List(ctx context.Context) (*kubermaticv1.ConstraintTemplateList, error)

	// Get gets the given constraint template
	Get(ctx context.Context, name string) (*kubermaticv1.ConstraintTemplate, error)

	// Create a Constraint Template
	Create(ctx context.Context, ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error)

	// Update a Constraint Template
	Update(ctx context.Context, ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error)

	// Delete a Constraint Template
	Delete(ctx context.Context, ct *kubermaticv1.ConstraintTemplate) error
}

// ConstraintProvider declares the set of method for interacting with constraints.
type ConstraintProvider interface {
	// List gets a list of constraints
	//
	// Note that the list is taken from the cache
	List(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.ConstraintList, error)

	// Get gets the given constraints
	Get(ctx context.Context, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.Constraint, error)

	// Create creates the given constraint
	Create(ctx context.Context, userInfo *UserInfo, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error)

	// Delete deletes the given constraint
	Delete(ctx context.Context, cluster *kubermaticv1.Cluster, userInfo *UserInfo, name string) error

	// Update updates the given constraint
	Update(ctx context.Context, userInfo *UserInfo, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error)
}

// PrivilegedConstraintProvider declares a set of methods for interacting with constraints using a privileged client.
type PrivilegedConstraintProvider interface {
	// CreateUnsecured creates the given constraint using a privileged client
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error)

	// DeleteUnsecured deletes a constraint using a privileged client
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) error

	// UpdateUnsecured updates the given constraint using a privileged client
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error)
}

// DefaultConstraintProvider declares the set of method for interacting with default constraints.
type DefaultConstraintProvider interface {
	// List gets a list of default constraints
	//
	// Note that the list is taken from the cache
	List(ctx context.Context) (*kubermaticv1.ConstraintList, error)

	// Get gets the given default constraints
	Get(ctx context.Context, name string) (*kubermaticv1.Constraint, error)

	// Create creates the given default constraint
	Create(ctx context.Context, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error)

	// Delete deletes the given default constraint
	Delete(ctx context.Context, name string) error

	// Update a default constraint
	Update(ctx context.Context, ct *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error)
}

// AlertmanagerProvider declares the set of method for interacting with alertmanagers.
type AlertmanagerProvider interface {
	// Get gets the given alertmanager and the config secret
	Get(ctx context.Context, cluster *kubermaticv1.Cluster, userInfo *UserInfo) (*kubermaticv1.Alertmanager, *corev1.Secret, error)

	// Update updates the given alertmanager and the config secret
	Update(ctx context.Context, alertmanager *kubermaticv1.Alertmanager, configSecret *corev1.Secret, userInfo *UserInfo) (*kubermaticv1.Alertmanager, *corev1.Secret, error)

	// Reset resets the given alertmanager to default
	Reset(ctx context.Context, cluster *kubermaticv1.Cluster, userInfo *UserInfo) error
}

// PrivilegedAlertmanagerProvider declares the set of method for interacting with alertmanagers using a privileged client.
type PrivilegedAlertmanagerProvider interface {
	// GetUnsecured gets the given alertmanager and the config secret using a privileged client
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.Alertmanager, *corev1.Secret, error)

	// UpdateUnsecured updates the given alertmanager and the config secret using a privileged client
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, alertmanager *kubermaticv1.Alertmanager, configSecret *corev1.Secret) (*kubermaticv1.Alertmanager, *corev1.Secret, error)

	// ResetUnsecured resets the given alertmanager to default using a privileged client
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to reset the resource
	ResetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) error
}

// ClusterTemplateProvider declares the set of method for interacting with cluster templates.
type ClusterTemplateProvider interface {
	New(ctx context.Context, userInfo *UserInfo, newClusterTemplate *kubermaticv1.ClusterTemplate, scope, projectID string) (*kubermaticv1.ClusterTemplate, error)
	List(ctx context.Context, userInfo *UserInfo, projectID string) ([]kubermaticv1.ClusterTemplate, error)
	ListALL(ctx context.Context, labelSelector labels.Selector) ([]kubermaticv1.ClusterTemplate, error)
	Get(ctx context.Context, userInfo *UserInfo, projectID, templateID string) (*kubermaticv1.ClusterTemplate, error)
	Delete(ctx context.Context, userInfo *UserInfo, projectID, templateID string) error
}

// ClusterTemplateInstanceProvider declares the set of method for interacting with cluster templates.
type ClusterTemplateInstanceProvider interface {
	Create(ctx context.Context, userInfo *UserInfo, template *kubermaticv1.ClusterTemplate, project *kubermaticv1.Project, replicas int64) (*kubermaticv1.ClusterTemplateInstance, error)
	Get(ctx context.Context, userInfo *UserInfo, name string) (*kubermaticv1.ClusterTemplateInstance, error)
	List(ctx context.Context, userInfo *UserInfo, options ClusterTemplateInstanceListOptions) (*kubermaticv1.ClusterTemplateInstanceList, error)
	Patch(ctx context.Context, userInfo *UserInfo, instance *kubermaticv1.ClusterTemplateInstance) (*kubermaticv1.ClusterTemplateInstance, error)
}

// PrivilegedClusterTemplateInstanceProvider declares the set of methods for interacting with the cluster template instances
// as an admin.
type PrivilegedClusterTemplateInstanceProvider interface {
	// CreateUnsecured create cluster template instance
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	CreateUnsecured(ctx context.Context, userInfo *UserInfo, template *kubermaticv1.ClusterTemplate, project *kubermaticv1.Project, replicas int64) (*kubermaticv1.ClusterTemplateInstance, error)

	// GetUnsecured gets cluster template instance
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, name string) (*kubermaticv1.ClusterTemplateInstance, error)

	// ListUnsecured lists cluster template instances
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	ListUnsecured(ctx context.Context, options ClusterTemplateInstanceListOptions) (*kubermaticv1.ClusterTemplateInstanceList, error)

	// PatchUnsecured patches cluster template instances
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	PatchUnsecured(ctx context.Context, instance *kubermaticv1.ClusterTemplateInstance) (*kubermaticv1.ClusterTemplateInstance, error)
}

// ClusterTemplateInstanceListOptions allows to set filters that will be applied to filter the result.
type ClusterTemplateInstanceListOptions struct {
	// ProjectID list only instances with the specified ID
	ProjectID string

	// TemplateID list only instances with the specified ID
	TemplateID string
}

type RuleGroupListOptions struct {
	RuleGroupType kubermaticv1.RuleGroupType
}

// RuleGroupProvider declares the set of methods for interacting with ruleGroups.
type RuleGroupProvider interface {
	// Get gets the given ruleGroup
	Get(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, ruleGroupName string) (*kubermaticv1.RuleGroup, error)

	// List gets a list of ruleGroups, by default it returns all ruleGroup objects.
	// If you would like to filer the result, please set RuleGroupListOptions
	List(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, options *RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error)

	// Create creates the given ruleGroup
	Create(ctx context.Context, userInfo *UserInfo, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error)

	// Update updates an ruleGroup
	Update(ctx context.Context, userInfo *UserInfo, newRuleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error)

	// Delete deletes the ruleGroup with the given name
	Delete(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, ruleGroupName string) error
}

type PrivilegedRuleGroupProvider interface {
	// GetUnsecured gets the given ruleGroup
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, ruleGroupName, namespace string) (*kubermaticv1.RuleGroup, error)

	// ListUnsecured gets a list of ruleGroups, by default it returns all ruleGroup objects.
	// If you would like to filer the result, please set RuleGroupListOptions
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(ctx context.Context, namespace string, options *RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error)

	// CreateUnsecured creates the given ruleGroup
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error)

	// UpdateUnsecured updates an ruleGroup
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, newRuleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error)

	// DeleteUnsecured deletes the ruleGroup with the given name
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, ruleGroupName, namespace string) error
}

// PrivilegedAllowedRegistryProvider declares the set of method for interacting with allowed registries.
type PrivilegedAllowedRegistryProvider interface {
	// CreateUnsecured creates the given allowed registry
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, ar *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error)

	// GetUnsecured gets the given allowed registry
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, name string) (*kubermaticv1.AllowedRegistry, error)

	// ListUnsecured gets a list of all allowed registries
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(ctx context.Context) (*kubermaticv1.AllowedRegistryList, error)

	// UpdateUnsecured updates the allowed registry
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, ar *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error)

	// DeleteUnsecured deletes the allowed registry with the given name
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, name string) error
}

// EtcdBackupConfigProvider declares the set of method for interacting with etcd backup configs.
type EtcdBackupConfigProvider interface {
	// Create creates the given etcdBackupConfig
	Create(ctx context.Context, userInfo *UserInfo, etcdBackupConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error)

	// Get gets the given etcdBackupConfig
	Get(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error)

	// List gets a list of etcdBackupConfig for a given cluster
	List(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error)

	// Delete deletes the given etcdBackupConfig
	Delete(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, name string) error

	// Patch updates the given etcdBackupConfig
	Patch(ctx context.Context, userInfo *UserInfo, oldConfig, newConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error)
}

// PrivilegedEtcdBackupConfigProvider declares the set of method for interacting with etcd backup configs using a privileged client.
type PrivilegedEtcdBackupConfigProvider interface {
	// CreateUnsecured creates the given etcdBackupConfig
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, etcdBackupConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error)

	// GetUnsecured gets the given etcdBackupConfig
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error)

	// ListUnsecured gets a list of all etcdBackupConfigs for a given cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to list the resources
	ListUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error)

	// DeleteUnsecured deletes the given etcdBackupConfig
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) error

	// PatchUnsecured patches the given etcdBackupConfig
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to patch the resource
	PatchUnsecured(ctx context.Context, oldConfig, newConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error)
}

// EtcdRestoreProvider declares the set of method for interacting with etcd backup restores.
type EtcdRestoreProvider interface {
	// Create creates the given etcdRestore
	Create(ctx context.Context, userInfo *UserInfo, etcdRestore *kubermaticv1.EtcdRestore) (*kubermaticv1.EtcdRestore, error)

	// Get gets the given etcdRestore
	Get(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdRestore, error)

	// List gets a list of etcdRestore for a given cluster
	List(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdRestoreList, error)

	// Delete deletes the given etcdRestore
	Delete(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster, name string) error
}

// PrivilegedEtcdRestoreProvider declares the set of method for interacting with etcd backup configs using a privileged client.
type PrivilegedEtcdRestoreProvider interface {
	// CreateUnsecured creates the given etcdRestore
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, etcdRestore *kubermaticv1.EtcdRestore) (*kubermaticv1.EtcdRestore, error)

	// GetUnsecured gets the given etcdRestore
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdRestore, error)

	// ListUnsecured gets a list of all etcdRestores for a given cluster
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to list the resources
	ListUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdRestoreList, error)

	// DeleteUnsecured deletes the given etcdRestore
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) error
}

// EtcdBackupConfigProjectProvider declares the set of method for interacting with etcd backup configs across projects and its seeds.
type EtcdBackupConfigProjectProvider interface {
	// List gets a list of etcdBackupConfig for a given project
	List(ctx context.Context, userInfo *UserInfo, projectID string) ([]*kubermaticv1.EtcdBackupConfigList, error)
}

// PrivilegedEtcdBackupConfigProjectProvider declares the set of method for interacting with etcd backup configs using a privileged client across projects and its seeds.
type PrivilegedEtcdBackupConfigProjectProvider interface {
	// ListUnsecured gets a list of all etcdBackupConfigs for a given project
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to list the resources
	ListUnsecured(ctx context.Context, projectID string) ([]*kubermaticv1.EtcdBackupConfigList, error)
}

// EtcdRestoreProjectProvider declares the set of method for interacting with etcd backup restores across projects and its seeds.
type EtcdRestoreProjectProvider interface {
	// List gets a list of etcdRestore for a given project
	List(ctx context.Context, userInfo *UserInfo, projectID string) ([]*kubermaticv1.EtcdRestoreList, error)
}

// PrivilegedEtcdRestoreProjectProvider declares the set of method for interacting with etcd backup configs using a privileged client across projects and its seeds.
type PrivilegedEtcdRestoreProjectProvider interface {
	// ListUnsecured gets a list of all etcdRestores for a given project
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to list the resources
	ListUnsecured(ctx context.Context, projectID string) ([]*kubermaticv1.EtcdRestoreList, error)
}

// FeatureGatesProvider declares the set of method for getting currently subset of provided feature gates.
type FeatureGatesProvider interface {
	GetFeatureGates() (apiv2.FeatureGates, error)
}

// BackupCredentialsProvider declares the set of method for interacting with etcd backup credentials using a privileged client.
type BackupCredentialsProvider interface {
	// CreateUnsecured creates the backup credentials
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, credentials *corev1.Secret) (*corev1.Secret, error)

	// GetUnsecured gets the backup credentials
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, credentialName string) (*corev1.Secret, error)

	// UpdateUnsecured updates the backup credentials
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, newSecret *corev1.Secret) (*corev1.Secret, error)
}

type PrivilegedMLAAdminSettingProvider interface {
	// GetUnsecured gets the given MLAAdminSetting
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.MLAAdminSetting, error)

	// CreateUnsecured creates the given MLAAdminSetting
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, mlaAdminSetting *kubermaticv1.MLAAdminSetting) (*kubermaticv1.MLAAdminSetting, error)

	// UpdateUnsecured updates an MLAAdminSetting
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, newMLAAdminSetting *kubermaticv1.MLAAdminSetting) (*kubermaticv1.MLAAdminSetting, error)

	// DeleteUnsecured deletes the MLAAdminSetting with the given name
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) error
}

type SeedProvider interface {
	// UpdateUnsecured updates a Seed
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	UpdateUnsecured(ctx context.Context, seed *kubermaticv1.Seed) (*kubermaticv1.Seed, error)
	// CreateUnsecured creates a new Seed
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	CreateUnsecured(ctx context.Context, seed *kubermaticv1.Seed) (*kubermaticv1.Seed, error)

	// CreateOrUpdateKubeconfigSecretForSeed creates or update seed kubeconfig
	CreateOrUpdateKubeconfigSecretForSeed(ctx context.Context, seed *kubermaticv1.Seed, kubeconfig []byte) error
}

type ResourceQuotaProvider interface {
	// GetUnsecured returns a resource quota based on object's name.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	GetUnsecured(ctx context.Context, name string) (*kubermaticv1.ResourceQuota, error)

	// Get returns a resource quota object based on name.
	Get(ctx context.Context, userInfo *UserInfo, name, kind string) (*kubermaticv1.ResourceQuota, error)

	// ListUnsecured returns a resource quota list.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	ListUnsecured(ctx context.Context, labelSet map[string]string) (*kubermaticv1.ResourceQuotaList, error)

	// CreateUnsecured creates a new resource quota.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	CreateUnsecured(ctx context.Context, subject kubermaticv1.Subject, quota kubermaticv1.ResourceDetails) error

	// PatchUnsecured patches given resource quota.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	PatchUnsecured(ctx context.Context, oldResourceQuota, newResourceQuota *kubermaticv1.ResourceQuota) error

	// DeleteUnsecured removes an existing resource quota.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	DeleteUnsecured(ctx context.Context, name string) error
}

type GroupProjectBindingProvider interface {
	// List returns a list of GroupProjectBindings for a given project.
	List(ctx context.Context, userInfo *UserInfo, projectID string) ([]kubermaticv1.GroupProjectBinding, error)

	// Get returns a GroupProjectBinding of a given name.
	Get(ctx context.Context, userInfo *UserInfo, name string) (*kubermaticv1.GroupProjectBinding, error)

	// Create creates a new GroupProjectBinding.
	Create(ctx context.Context, userInfo *UserInfo, binding *kubermaticv1.GroupProjectBinding) error

	// Patch patches an existing GroupProjectBinding.
	Patch(ctx context.Context, userInfo *UserInfo, oldBinding, newBinding *kubermaticv1.GroupProjectBinding) error

	// Delete removes an existing GroupProjectBinding.
	Delete(ctx context.Context, userInfo *UserInfo, name string) error
}

type PrivilegedIPAMPoolProvider interface {
	// ListUnsecured gets the IPAM pool list.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(ctx context.Context) (*kubermaticv1.IPAMPoolList, error)

	// Get returns a IPAM pool based on name.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resource
	GetUnsecured(ctx context.Context, ipamPoolName string) (*kubermaticv1.IPAMPool, error)

	// DeleteUnsecured deletes a IPAM pool based on name.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to delete the resource
	DeleteUnsecured(ctx context.Context, ipamPoolName string) error

	// CreateUnsecured creates a IPAM pool.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to create the resource
	CreateUnsecured(ctx context.Context, ipamPool *kubermaticv1.IPAMPool) error

	// PatchUnsecured patches a IPAM pool.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to update the resource
	PatchUnsecured(ctx context.Context, oldIPAMPool *kubermaticv1.IPAMPool, newIPAMPool *kubermaticv1.IPAMPool) error
}

type ApplicationDefinitionProvider interface {
	// List returns a list of ApplicationDefinitions for the KKP installation.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	ListUnsecured(ctx context.Context) (*appskubermaticv1.ApplicationDefinitionList, error)

	// Get returns a  ApplicationDefinition based on name.
	//
	// Note that this function:
	// is unsafe in a sense that it uses privileged account to get the resources
	GetUnsecured(ctx context.Context, appDefName string) (*appskubermaticv1.ApplicationDefinition, error)
}

type PrivilegedOperatingSystemProfileProvider interface {
	// List returns a list of OperatingSystemProfiles for the KKP installation.
	ListUnsecured(context.Context) (*osmv1alpha1.OperatingSystemProfileList, error)

	ListUnsecuredForUserClusterNamespace(context.Context, string) (*osmv1alpha1.OperatingSystemProfileList, error)
}
