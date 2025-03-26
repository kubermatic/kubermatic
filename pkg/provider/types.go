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
	"strconv"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	apiv2 "k8c.io/kubermatic/sdk/v2/api/v2"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ksemver "k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
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

// CloudProvider declares a set of methods for interacting with a cloud provider.
type CloudProvider interface {
	InitializeCloudProvider(context.Context, *kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	CleanUpCloudProvider(context.Context, *kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)
	DefaultCloudSpec(context.Context, *kubermaticv1.ClusterSpec) error
	ValidateCloudSpec(context.Context, kubermaticv1.CloudSpec) error
	ValidateCloudSpecUpdate(ctx context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error
}

// ReconcilingCloudProvider is a cloud provider that can not just created resources
// once, but is capable of continuously reconciling and fixing any problems with them.
type ReconcilingCloudProvider interface {
	CloudProvider

	ReconcileCluster(context.Context, *kubermaticv1.Cluster, ClusterUpdater) (*kubermaticv1.Cluster, error)

	// Normally reconciling happens on a regular basis, but the interval between reconciliations can
	// be quite long if migrations/upgrades need to happen. A cloud provider can implement this to
	// decide based on the Cluster whether an immediate reconciliation is required. Cloud providers
	// should be careful and not blindly just "return true" here, as that would defeat the whole
	// "wait N minutes before doing expensive reconciliation" logic.
	ClusterNeedsReconciling(*kubermaticv1.Cluster) bool
}

// ClusterUpdater defines a function to persist an update to a cluster.
type ClusterUpdater func(context.Context, string, func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error)

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

// UserInfo represent authenticated user.
type UserInfo struct {
	Email   string
	Groups  []string
	Roles   sets.Set[string]
	IsAdmin bool
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

	CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticv1.ExternalCluster, kubeconfig []byte, namespace string) error

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

// NodeCapacity represents the size of a cluster node in a Kubernetes cluster.
type NodeCapacity struct {
	CPUCores *resource.Quantity
	GPUs     *resource.Quantity
	Memory   *resource.Quantity
	Storage  *resource.Quantity
}

func NewNodeCapacity() *NodeCapacity {
	return &NodeCapacity{}
}

func (c *NodeCapacity) WithCPUCount(cpus int) {
	quantity, _ := resource.ParseQuantity(strconv.Itoa(cpus))
	c.CPUCores = &quantity
}

func (c *NodeCapacity) WithGPUCount(gpus int) {
	quantity, _ := resource.ParseQuantity(strconv.Itoa(gpus))
	c.GPUs = &quantity
}

func (c *NodeCapacity) WithMemory(value int, unit string) error {
	quantity, err := resource.ParseQuantity(fmt.Sprintf("%d%s", value, unit))
	if err != nil {
		return err
	}

	c.Memory = &quantity

	return nil
}

func (c *NodeCapacity) WithStorage(value int, unit string) error {
	quantity, err := resource.ParseQuantity(fmt.Sprintf("%d%s", value, unit))
	if err != nil {
		return err
	}

	c.Storage = &quantity

	return nil
}
