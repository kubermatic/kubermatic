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

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
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
type KubermaticConfigurationGetter = func(context.Context) (*kubermaticv1.KubermaticConfiguration, error)

// DatacenterGetter is a function to retrieve a single Datacenter.
type DatacenterGetter = func(context.Context, string) (*kubermaticv1.Datacenter, error)

// DatacentersGetter is a function to retrieve a list of all available Datacenters.
type DatacentersGetter = func(context.Context) (map[string]*kubermaticv1.Datacenter, error)

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
}

// ClusterUpdater defines a function to persist an update to a cluster.
type ClusterUpdater func(context.Context, string, func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error)

// SecretKeySelectorValueFunc is used to fetch the value of a config var. Do not build your own
// implementation, use SecretKeySelectorValueFuncFactory.
type SecretKeySelectorValueFunc func(configVar *kubermaticv1.GlobalSecretKeySelector, key string) (string, error)

func SecretKeySelectorValueFuncFactory(ctx context.Context, client ctrlruntimeclient.Reader) SecretKeySelectorValueFunc {
	return func(configVar *kubermaticv1.GlobalSecretKeySelector, key string) (string, error) {
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
