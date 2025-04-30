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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// FinalizerNamespace will ensure the deletion of the dedicated namespace.
	FinalizerNamespace = "kubermatic.k8c.io/cleanup-kubevirt-namespace"
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
		return nil, errors.New("datacenter is not a KubeVirt datacenter")
	}
	return &kubevirt{
		secretKeySelector: secretKeyGetter,
		dc:                dc.Spec.Kubevirt,
		log:               log.Logger,
	}, nil
}

func isNamespaceModeEnabled(dc *kubermaticv1.DatacenterSpecKubevirt) bool {
	if dc.NamespacedMode != nil {
		return dc.NamespacedMode.Enabled
	}
	return false
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

	if k.dc.CSIDriverOperator != nil {
		spec.Cloud.Kubevirt.CSIDriverOperator = k.dc.CSIDriverOperator
	}

	return updateInfraStorageClassesInfo(ctx, client, spec.Cloud.Kubevirt, k.dc)
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
	spec.Kubevirt.Kubeconfig = string(config)

	return nil
}

func (k *kubevirt) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return k.reconcileCluster(ctx, cluster, update)
}

func (*kubevirt) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

func (k *kubevirt) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return k.reconcileCluster(ctx, cluster, update)
}

func (k *kubevirt) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	client, err := k.GetClientForCluster(cluster.Spec.Cloud)
	if err != nil {
		return cluster, err
	}
	kubevirtNamespace := cluster.Status.NamespaceName
	if k.dc.NamespacedMode != nil && k.dc.NamespacedMode.Enabled {
		kubevirtNamespace = k.dc.NamespacedMode.Namespace
	}
	// If the cluster NamespaceName is not filled yet, return a conflict error:
	// will requeue but not send an error event
	if cluster.Status.NamespaceName == "" {
		return cluster, apierrors.NewConflict(kubermaticv1.Resource("cluster"), cluster.Name, fmt.Errorf("cluster.Status.NamespaceName for cluster %s", cluster.Name))
	}

	cluster, err = reconcileNamespace(ctx, kubevirtNamespace, cluster, update, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileCSIRoleRoleBinding(ctx, kubevirtNamespace, client)
	if err != nil {
		return cluster, err
	}

	err = ReconcileInfraTokenAccess(ctx, kubevirtNamespace, client)
	if err != nil {
		return cluster, err
	}

	err = reconcileInstancetypes(ctx, kubevirtNamespace, client)
	if err != nil {
		return cluster, err
	}

	err = reconcilePreferences(ctx, kubevirtNamespace, client)
	if err != nil {
		return cluster, err
	}

	enableDefaultNetworkPolices := true
	if k.dc.EnableDefaultNetworkPolicies != nil {
		enableDefaultNetworkPolices = *k.dc.EnableDefaultNetworkPolicies
	}
	if enableDefaultNetworkPolices && !isNamespaceModeEnabled(k.dc) {
		err = reconcileClusterIsolationNetworkPolicy(ctx, cluster, k.dc, client, kubevirtNamespace)
		if err != nil {
			return cluster, err
		}
	}

	err = reconcileCustomNetworkPolicies(ctx, cluster, k.dc, client, kubevirtNamespace)
	if err != nil {
		return cluster, err
	}

	return cluster, err
}

func (k *kubevirt) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasAnyFinalizer(cluster, FinalizerNamespace, FinalizerClonerRoleBinding) {
		return cluster, nil
	}
	// When the seed and kubevirt provider cluster are the same, the user cluster's namespace (cluster-<clusterID>) created on the kubevirt-provider
	// is the same as the namespace on the seed that has all the control plane pods of the user cluster. When a user cluster on such setup is deleted
	// the namespace is cleaned up by the cloud provider finalizer (FinalizerNamespace) & doesn't wait for the etcd backups to be removed, the namespace
	// deletion removes the secrets required for cleaning up etcd backups & blocks the cluster deletion, to prevent this we wait for the backups to get
	// cleaned by before deleting the namespace.
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticv1.EtcdBackupConfigCleanupFinalizer) {
		client, err := k.GetClientForCluster(cluster.Spec.Cloud)
		if err != nil {
			return cluster, err
		}
		if !isNamespaceModeEnabled(k.dc) {
			if err := deleteNamespace(ctx, cluster.Status.NamespaceName, client); err != nil && !apierrors.IsNotFound(err) {
				return cluster, fmt.Errorf("failed to delete namespace %s: %w", cluster.Status.NamespaceName, err)
			}
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
	return cluster, nil
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
