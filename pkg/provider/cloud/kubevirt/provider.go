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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/image"
	"k8c.io/kubermatic/v2/pkg/resources"

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
	dc                *kubermaticv1.DatacenterSpecKubevirt
	log               *zap.SugaredLogger
}

func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (provider.CloudProvider, error) {
	if dc.Spec.Kubevirt == nil {
		return nil, errors.New("datacenter is not an KubeVirt datacenter")
	}
	return &kubevirt{
		secretKeySelector: secretKeyGetter,
		dc:                dc.Spec.Kubevirt,
		log:               log.Logger,
	}, nil
}

var _ provider.ReconcilingCloudProvider = &kubevirt{}

func (k *kubevirt) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.Kubevirt == nil {
		return errors.New("KubeVirt cloud provider spec is empty")
	}

	client, err := k.GetClientForCluster(spec.Cloud)
	if err != nil {
		return err
	}

	// defaults imageCloningEnabled flag to datacenter value
	spec.Cloud.Kubevirt.ImageCloningEnabled = k.dc.Images.HTTP != nil && k.dc.Images.HTTP.ImageCloning.Enabled

	return updateInfraStorageClassesInfo(ctx, client, &spec.Cloud)
}

// isImageCloningEnabled returns if the image cloning feature is enable or not:
// The value is defaulted from the seed datacenter in DefaultCloudSpec() at cluster creation time.
func (k *kubevirt) isImageCloningEnabled(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Kubevirt.ImageCloningEnabled
}

// isCustomImagesEnabled value always comes from dc.
func (k *kubevirt) isCustomImagesEnabled() bool {
	return k.dc.Images.EnableCustomImages
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

	// TODO: refactor, it is wrong to mutate the value of kubeconfig in the validation method
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
	logger := k.log.With("cluster", cluster.Name)
	client, err := k.GetClientForCluster(cluster.Spec.Cloud)
	if err != nil {
		return cluster, err
	}

	// If the cluster NamespaceName is not filled yet, return a conflict error:
	// will requeue but not send an error event
	if cluster.Status.NamespaceName == "" {
		return cluster, apierrors.NewConflict(kubermaticv1.Resource("cluster"), cluster.Name, fmt.Errorf("cluster.Status.NamespaceName for cluster %s", cluster.Name))
	}

	cluster, err = reconcileClusterNamespace(ctx, cluster.Status.NamespaceName, cluster, update, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileCSIRoleRoleBinding(ctx, cluster.Status.NamespaceName, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileInfraTokenAccess(ctx, cluster.Status.NamespaceName, client)
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

	if k.isImageCloningEnabled(cluster) || k.isCustomImagesEnabled() {
		err = reconcileImagesNamespace(ctx, KubeVirtImagesNamespace, client)
		if err != nil {
			return cluster, err
		}

		err = reconcileClusterImagesNetworkPolicy(ctx, cluster, client)
		if err != nil {
			return cluster, err
		}

		cluster, err = image.ReconcileImagesRoleBinding(ctx, KubeVirtImagesNamespace, cluster.Status.NamespaceName, cluster,
			update, client, []string{FinalizerNamespace})
		if err != nil {
			return cluster, err
		}

		if k.isImageCloningEnabled(cluster) {
			// Note: no cleanup of data volume in the KubeVirt image namespace is done.
			// If imageCloning is disabled in the cluster then machine-controller is restarted with env var that disables data volume clone.
			// TODO: move reconciliation of standard images cache to external KubeVirt operator.
			err = image.ReconcileStandardImages(ctx, k.dc, client, logger)
		}

		if k.isCustomImagesEnabled() {
			err = image.ReconcileCustomImages(ctx, cluster, client, logger)
			if err != nil {
				return cluster, err
			}
		}
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

	if err = deleteNamespace(ctx, cluster.Status.NamespaceName, client); err != nil && !apierrors.IsNotFound(err) {
		return cluster, fmt.Errorf("failed to delete namespace %s: %w", cluster.Status.NamespaceName, err)
	}
	cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(updatedCluster, FinalizerNamespace)
	})
	if err != nil {
		return cluster, err
	}

	rbacName := fmt.Sprintf("%s-%s", image.RBACName, cluster.Status.NamespaceName)
	if err = image.DeleteImagesRoleBinding(ctx, rbacName, client); err != nil && !apierrors.IsNotFound(err) {
		return cluster, fmt.Errorf("failed to delete rolebinding %s: %w", rbacName, err)
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
	kubeconfig = cloud.Kubevirt.Kubeconfig

	if kubeconfig == "" {
		if cloud.Kubevirt.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		kubeconfig, err = secretKeySelector(cloud.Kubevirt.CredentialsReference, resources.KubeVirtKubeconfig)
		if err != nil {
			return "", err
		}
	}

	return kubeconfig, nil
}
