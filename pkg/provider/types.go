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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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

	// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to
	Get(ctx context.Context, userInfo *UserInfo, clusterName string, options *ClusterGetOptions) (*kubermaticv1.Cluster, error)

	// Update updates a cluster
	Update(ctx context.Context, project *kubermaticv1.Project, userInfo *UserInfo, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error)

	// Delete deletes the given cluster
	Delete(ctx context.Context, userInfo *UserInfo, cluster *kubermaticv1.Cluster) error

	// GetAdminClientForUserCluster returns a client to interact with all resources in the given cluster
	//
	// Note that the client you will get has admin privileges
	GetAdminClientForUserCluster(context.Context, *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error)

	// GetAdminClientConfigForUserCluster returns a client config
	//
	// Note that the client you will get has admin privileges.
	GetAdminClientConfigForUserCluster(ctx context.Context, c *kubermaticv1.Cluster) (*restclient.Config, error)
}

// PrivilegedClusterProvider declares the set of methods for interacting with the seed clusters
// as an admin.
type PrivilegedClusterProvider interface {
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

// SettingsProvider declares the set of methods for interacting global settings.
type SettingsProvider interface {
	GetGlobalSettings(ctx context.Context) (*kubermaticv1.KubermaticSetting, error)
	UpdateGlobalSettings(ctx context.Context, userInfo *UserInfo, settings *kubermaticv1.KubermaticSetting) (*kubermaticv1.KubermaticSetting, error)
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

	CreateOrUpdateKubeOneCredentialSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, externalCluster *kubermaticv1.ExternalCluster) error

	GetVersion(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*ksemver.Semver, error)

	MasterVersions(ctx context.Context, configGetter KubermaticConfigurationGetter, providerType kubermaticv1.ExternalClusterProviderType) ([]apiv1.MasterVersion, error)

	ListNodes(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*corev1.NodeList, error)

	GetNode(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodeName string) (*corev1.Node, error)

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
