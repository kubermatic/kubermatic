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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/log"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/resources"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// FinalizerNamespace will ensure the deletion of the dedicated namespace.
	FinalizerNamespace = "kubermatic.k8c.io/cleanup-kubevirt-namespace"
	// KubeVirtImagesNamespace namespace contains globally available custom images and cached standard images.
	KubeVirtImagesNamespace = "kubevirt-images"
	// FinalizerClonerRoleBinding will ensure the deletion of the DataVolume cloner role-binding.
	FinalizerClonerRoleBinding = "kubermatic.k8c.io/cleanup-kubevirt-cloner-rbac"
)

type kubevirt struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
	dc                *kubermaticv1.DatacenterSpecKubeVirt
	log               *zap.SugaredLogger
}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (provider.CloudProvider, error) {
	if dc.Spec.KubeVirt == nil {
		return nil, errors.New("datacenter is not an KubeVirt datacenter")
	}
	return &kubevirt{
		secretKeySelector: secretKeyGetter,
		dc:                dc.Spec.KubeVirt,
		log:               log.Logger,
	}, nil
}

var _ provider.ReconcilingCloudProvider = &kubevirt{}

func (k *kubevirt) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.KubeVirt == nil {
		return errors.New("KubeVirt cloud provider spec is empty")
	}

	client, err := k.GetClientForCluster(spec.Cloud)
	if err != nil {
		return err
	}

	return updateInfraStorageClassesInfo(ctx, client, spec.Cloud.KubeVirt, k.dc)
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

	// TODO: (mfranczy) this has to be changed
	// it is wrong to mutate the value of kubeconfig in the validation method
	spec.KubeVirt.Kubeconfig = string(config)

	return nil
}

func (k *kubevirt) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return k.reconcileCluster(ctx, cluster, update)
}

func (k *kubevirt) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return k.reconcileCluster(ctx, cluster, update)
}

func (k *kubevirt) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	client, err := k.GetClientForCluster(cluster.Spec.Cloud)
	if err != nil {
		return cluster, err
	}

	// If the cluster NamespaceName is not filled yet, return a conflict error:
	// will requeue but not send an error event
	if cluster.Status.NamespaceName == "" {
		return cluster, apierrors.NewConflict(kubermaticv1.Resource("cluster"), cluster.Name, fmt.Errorf("cluster.Status.NamespaceName for cluster %s", cluster.Name))
	}

	cluster, err = reconcileNamespace(ctx, cluster.Status.NamespaceName, cluster, update, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileCSIRoleRoleBinding(ctx, cluster.Status.NamespaceName, client)
	if err != nil {
		return cluster, err
	}

	err = ReconcileInfraTokenAccess(ctx, cluster.Status.NamespaceName, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileInstancetypes(ctx, cluster.Status.NamespaceName, client)
	if err != nil {
		return cluster, err
	}

	err = reconcilePreferences(ctx, cluster.Status.NamespaceName, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileClusterIsolationNetworkPolicy(ctx, cluster, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileCustomNetworkPolicies(ctx, cluster, k.dc, client)
	if err != nil {
		return cluster, err
	}

	return cluster, err
}

func (k *kubevirt) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasAnyFinalizer(cluster, FinalizerNamespace, FinalizerClonerRoleBinding) {
		return cluster, nil
	}

	client, err := k.GetClientForCluster(cluster.Spec.Cloud)
	if err != nil {
		return cluster, err
	}

	if err := deleteNamespace(ctx, cluster.Status.NamespaceName, client); err != nil && !apierrors.IsNotFound(err) {
		return cluster, fmt.Errorf("failed to delete namespace %s: %w", cluster.Status.NamespaceName, err)
	}
	cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerNamespace)
	})
	if err != nil {
		return cluster, err
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerClonerRoleBinding)
	})
}

func (k *kubevirt) ValidateCloudSpecUpdate(ctx context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetClientForCluster returns the kubernetes client the KubeVirt underlying cluster.
func (k *kubevirt) GetClientForCluster(spec kubermaticv1.CloudSpec) (*Client, error) {
	kubeconfig, err := GetCredentialsForCluster(spec, k.secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(kubeconfig, ClientOptions{})
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (kubeconfig string, err error) {
	kubeconfig = cloud.KubeVirt.Kubeconfig

	if kubeconfig == "" {
		if cloud.KubeVirt.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		kubeconfig, err = secretKeySelector(cloud.KubeVirt.CredentialsReference, resources.KubeVirtKubeconfig)
		if err != nil {
			return "", err
		}
	}

	return kubeconfig, nil
}
