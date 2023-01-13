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

package kubevirt

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	kubevirtv1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// FinalizerNamespace will ensure the deletion of the dedicated namespace.
	FinalizerNamespace = "kubermatic.k8c.io/cleanup-kubevirt-namespace"
)

type kubevirt struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
	dc                *kubermaticv1.DatacenterSpecKubevirt
}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (provider.CloudProvider, error) {
	if dc.Spec.Kubevirt == nil {
		return nil, errors.New("datacenter is not an KubeVirt datacenter")
	}
	return &kubevirt{
		secretKeySelector: secretKeyGetter,
		dc:                dc.Spec.Kubevirt,
	}, nil
}

var _ provider.ReconcilingCloudProvider = &kubevirt{}

func (k *kubevirt) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.CloudSpec) error {
	if spec.Kubevirt != nil {
		return updateInfraStorageClassesInfo(ctx, spec, k.secretKeySelector)
	}
	return nil
}

func (k *kubevirt) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	kubeconfig, err := GetCredentialsForCluster(spec, k.secretKeySelector)
	if err != nil {
		return err
	}

	config, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		// if the decoding failed, the kubeconfig is sent already decoded without the need of decoding it,
		// for example the value has been read from Vault during the ci tests, which is saved as json format.
		config = []byte(kubeconfig)
	}

	_, err = clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return err
	}

	spec.Kubevirt.Kubeconfig = string(config)

	return nil
}

func (k *kubevirt) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return k.reconcileCluster(ctx, cluster, update)
}

func (k *kubevirt) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return k.reconcileCluster(ctx, cluster, update)
}

func (k *kubevirt) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// Reconcile CSI access: Role and Rolebinding
	client, restConfig, err := k.GetClientWithRestConfigForCluster(cluster)
	if err != nil {
		return cluster, err
	}

	cluster, err = reconcileNamespace(ctx, cluster.Status.NamespaceName, cluster, update, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileCSIRoleRoleBinding(ctx, cluster.Status.NamespaceName, client, restConfig)
	if err != nil {
		return cluster, err
	}

	err = reconcilePresets(ctx, cluster.Status.NamespaceName, client)
	if err != nil {
		return cluster, err
	}

	err = reconcilePreAllocatedDataVolumes(ctx, cluster, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileNetworkPolicy(ctx, cluster, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileCustomNetworkPolicies(ctx, cluster, k.dc, client)

	return cluster, err
}

func (k *kubevirt) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(cluster, FinalizerNamespace) {
		return cluster, nil
	}

	client, _, err := k.GetClientWithRestConfigForCluster(cluster)
	if err != nil {
		return cluster, err
	}

	if err := deleteNamespace(ctx, cluster.Status.NamespaceName, client); err != nil && !apierrors.IsNotFound(err) {
		return cluster, fmt.Errorf("failed to delete namespace %s: %w", cluster.Status.NamespaceName, err)
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerNamespace)
	})
}

func (k *kubevirt) ValidateCloudSpecUpdate(ctx context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetClientWithRestConfigForCluster returns the kubernetes client and the rest config for the KubeVirt underlying cluster.
func (k *kubevirt) GetClientWithRestConfigForCluster(cluster *kubermaticv1.Cluster) (ctrlruntimeclient.Client, *restclient.Config, error) {
	if cluster.Spec.Cloud.Kubevirt == nil {
		return nil, nil, errors.New("No KubeVirt provider spec")
	}
	kubeconfig, err := GetCredentialsForCluster(cluster.Spec.Cloud, k.secretKeySelector)
	if err != nil {
		return nil, nil, err
	}

	client, restConfig, err := NewClientWithRestConfig(kubeconfig)
	if err != nil {
		return nil, nil, err
	}

	if err := kubevirtv1.AddToScheme(client.Scheme()); err != nil {
		return nil, nil, err
	}
	if err = cdiv1beta1.AddToScheme(client.Scheme()); err != nil {
		return nil, nil, err
	}

	return client, restConfig, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (kubeconfig string, err error) {
	kubeconfig = cloud.Kubevirt.Kubeconfig

	if kubeconfig == "" {
		if cloud.Kubevirt.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		kubeconfig, err = secretKeySelector(cloud.Kubevirt.CredentialsReference, resources.KubevirtKubeConfig)
		if err != nil {
			return "", err
		}
	}

	return kubeconfig, nil
}
