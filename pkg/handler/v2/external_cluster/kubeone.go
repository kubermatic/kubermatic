/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package externalcluster

import (
	"context"
	"fmt"
	"strings"

	kubeonev1beta2 "k8c.io/kubeone/pkg/apis/kubeone/v1beta2"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	NodeControlPlaneLabel = "node-role.kubernetes.io/control-plane"
)

func importKubeOneCluster(ctx context.Context, name string, userInfoGetter func(ctx context.Context, projectID string) (*provider.UserInfo, error), project *kubermaticv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticv1.ExternalCluster, error) {
	kubeOneCluster, err := DecodeManifestFromKubeOneReq(cloud.KubeOne.Manifest)
	if err != nil {
		return nil, err
	}

	newCluster := genExternalCluster(kubeOneCluster.Name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticv1.ExternalClusterCloudSpec{
		KubeOne: &kubermaticv1.ExternalClusterKubeOneCloudSpec{},
	}

	err = clusterProvider.CreateKubeOneClusterNamespace(ctx, newCluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	err = clusterProvider.CreateOrUpdateKubeOneSSHSecret(ctx, cloud.KubeOne.SSHKey, newCluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.ExternalClusterKubeOneSSHSecretCleanupFinalizer)

	err = clusterProvider.CreateOrUpdateKubeOneManifestSecret(ctx, cloud.KubeOne.Manifest, newCluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.ExternalClusterKubeOneManifestSecretCleanupFinalizer)

	err = clusterProvider.CreateOrUpdateKubeOneCredentialSecret(ctx, *cloud.KubeOne.CloudSpec, newCluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.CredentialsSecretsCleanupFinalizer)

	newCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusProvisioning
	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchKubeOneCluster(ctx context.Context,
	cluster *kubermaticv1.ExternalCluster,
	oldCluster *apiv2.ExternalCluster,
	newCluster *apiv2.ExternalCluster,
	secretKeySelector provider.SecretKeySelectorValueFunc,
	clusterProvider provider.ExternalClusterProvider,
	masterClient ctrlruntimeclient.Client) (*apiv2.ExternalCluster, error) {
	kubeOneSpec := cluster.Spec.CloudSpec.KubeOne
	operation := kubeOneSpec.ClusterStatus.Status
	if operation == kubermaticv1.StatusReconciling {
		return nil, errors.NewBadRequest("Operation is not allowed: Another operation: (%s) is in progress, please wait for it to finish before starting a new operation.", operation)
	}

	if oldCluster.Spec.Version != newCluster.Spec.Version {
		return UpgradeKubeOneCluster(ctx, cluster, oldCluster, newCluster, clusterProvider, masterClient)
	}
	if oldCluster.Cloud.KubeOne.ContainerRuntime != newCluster.Cloud.KubeOne.ContainerRuntime {
		return MigrateKubeOneToContainerd(ctx, cluster, oldCluster, newCluster, clusterProvider, masterClient)
	}

	return newCluster, nil
}

func UpgradeKubeOneCluster(ctx context.Context,
	externalCluster *kubermaticv1.ExternalCluster,
	oldCluster *apiv2.ExternalCluster,
	newCluster *apiv2.ExternalCluster,
	externalClusterProvider provider.ExternalClusterProvider,
	masterClient ctrlruntimeclient.Client,
) (*apiv2.ExternalCluster, error) {
	manifest := externalCluster.Spec.CloudSpec.KubeOne.ManifestReference

	manifestSecret := &corev1.Secret{}
	if err := masterClient.Get(ctx, types.NamespacedName{Namespace: manifest.Namespace, Name: manifest.Name}, manifestSecret); err != nil {
		return nil, errors.NewBadRequest(fmt.Sprintf("can not retrieve kubeone manifest secret: %v", err))
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]

	cluster := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, cluster); err != nil {
		return nil, fmt.Errorf("failed to decode manifest secret data: %w", err)
	}
	upgradeVersion := newCluster.Spec.Version.Semver().String()
	cluster.Versions = kubeonev1beta2.VersionConfig{
		Kubernetes: upgradeVersion,
	}

	if oldCluster.Cloud.KubeOne.ContainerRuntime == resources.ContainerRuntimeDocker {
		cluster.ContainerRuntime.Containerd = nil
		if upgradeVersion >= "1.24" {
			return nil, errors.NewBadRequest("container runtime is \"docker\". Support for docker will be removed with Kubernetes 1.24 release.")
		} else if cluster.ContainerRuntime.Docker == nil {
			cluster.ContainerRuntime.Docker = &kubeonev1beta2.ContainerRuntimeDocker{}
		}
	}

	patchManifest, err := yaml.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to encode kubeone cluster manifest config as YAML: %w", err)
	}

	oldManifestSecret := manifestSecret.DeepCopy()
	manifestSecret.Data = map[string][]byte{
		resources.KubeOneManifest: patchManifest,
	}
	if err := masterClient.Patch(ctx, manifestSecret, ctrlruntimeclient.MergeFrom(oldManifestSecret)); err != nil {
		return nil, fmt.Errorf("failed to update kubeone manifest secret for upgrade version %s/%s: %w", manifest.Name, manifest.Namespace, err)
	}

	oldexternalCluster := externalCluster.DeepCopy()
	// update kubeone externalcluster status.
	externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusReconciling
	// update api externalcluster status.
	newCluster.Status.State = apiv2.RECONCILING
	if err := masterClient.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		return nil, fmt.Errorf("failed to update kubeone cluster status %s: %w", externalCluster.Name, err)
	}

	return newCluster, nil
}

func MigrateKubeOneToContainerd(ctx context.Context,
	externalCluster *kubermaticv1.ExternalCluster,
	oldCluster *apiv2.ExternalCluster,
	newCluster *apiv2.ExternalCluster,
	externalClusterProvider provider.ExternalClusterProvider,
	masterClient ctrlruntimeclient.Client,
) (*apiv2.ExternalCluster, error) {
	kubeOneSpec := externalCluster.Spec.CloudSpec.KubeOne
	manifest := kubeOneSpec.ManifestReference
	wantedContainerRuntime := newCluster.Cloud.KubeOne.ContainerRuntime

	if kubeOneSpec.ClusterStatus.Status == kubermaticv1.StatusReconciling {
		return nil, errors.NewBadRequest("Operation is not allowed: Another operation: (Upgrading) is in progress, please wait for it to finish before starting a new operation.")
	}

	// currently only migration to containerd is supported
	supportedMigrationContainerRuntimes := map[string]struct{}{
		"containerd": {},
	}
	if _, isSupported := supportedMigrationContainerRuntimes[wantedContainerRuntime]; !isSupported {
		return nil, fmt.Errorf("container runtime not supported: %s", wantedContainerRuntime)
	}
	manifestSecret := &corev1.Secret{}
	if err := masterClient.Get(ctx, types.NamespacedName{Namespace: manifest.Namespace, Name: manifest.Name}, manifestSecret); err != nil {
		return nil, errors.NewBadRequest(fmt.Sprintf("can not retrieve kubeone manifest secret: %v", err))
	}
	currentManifest := manifestSecret.Data[resources.KubeOneManifest]
	cluster := &kubeonev1beta2.KubeOneCluster{}
	if err := yaml.UnmarshalStrict(currentManifest, cluster); err != nil {
		return nil, fmt.Errorf("failed to decode manifest secret data: %w", err)
	}
	cluster.ContainerRuntime.Docker = nil
	cluster.ContainerRuntime.Containerd = &kubeonev1beta2.ContainerRuntimeContainerd{}

	patchManifest, err := yaml.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to encode kubeone cluster manifest config as YAML: %w", err)
	}

	oldManifestSecret := manifestSecret.DeepCopy()
	manifestSecret.Data = map[string][]byte{
		resources.KubeOneManifest: patchManifest,
	}
	if err := masterClient.Patch(ctx, manifestSecret, ctrlruntimeclient.MergeFrom(oldManifestSecret)); err != nil {
		return nil, fmt.Errorf("failed to update kubeone manifest secret for container-runtime containerd %s/%s: %w", manifest.Name, manifest.Namespace, err)
	}

	oldexternalCluster := externalCluster.DeepCopy()
	// update kubeone externalcluster status.
	externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusReconciling
	// update api externalcluster status.
	newCluster.Status = apiv2.ExternalClusterStatus{State: apiv2.RECONCILING}

	if err := masterClient.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		return nil, fmt.Errorf("failed to update kubeone cluster status %s: %w", externalCluster.Name, err)
	}

	return newCluster, nil
}

func checkContainerRuntime(ctx context.Context,
	externalCluster *kubermaticv1.ExternalCluster,
	externalClusterProvider provider.ExternalClusterProvider,
) (string, error) {
	var containerRuntime string
	nodes, err := externalClusterProvider.ListNodes(ctx, externalCluster)
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}
	for _, node := range nodes.Items {
		if _, ok := node.Labels[NodeControlPlaneLabel]; ok {
			containerRuntimeVersion := node.Status.NodeInfo.ContainerRuntimeVersion
			strSlice := strings.Split(containerRuntimeVersion, ":")
			for _, v := range strSlice {
				containerRuntime = v
				break
			}
		}
	}
	return containerRuntime, nil
}
