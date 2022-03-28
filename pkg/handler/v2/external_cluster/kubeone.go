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

func patchKubeOneCluster(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster, newCluster *apiv2.ExternalCluster, upgradeMD string, secretKeySelector provider.SecretKeySelectorValueFunc, masterClient ctrlruntimeclient.Client) (*apiv2.ExternalCluster, error) {
	kubeOneSpec := externalCluster.Spec.CloudSpec.KubeOne
	if kubeOneSpec.ClusterStatus.Status == kubermaticv1.StatusReconciling {
		return nil, errors.NewBadRequest("Operation is not allowed: Another operation: (Upgrading) is in progress, please wait for it to finish before starting a new operation.")
	}

	manifest := kubeOneSpec.ManifestReference

	manifestValue, err := secretKeySelector(&manifest, resources.KubeOneManifest)

	kubeOneClusterManifest := &kubeonev1beta2.KubeOneCluster{}
	if err != nil {
		return nil, err
	}
	if err := yaml.UnmarshalStrict([]byte(manifestValue), kubeOneClusterManifest); err != nil {
		return nil, err
	}
	kubeOneClusterManifest.Versions = kubeonev1beta2.VersionConfig{
		Kubernetes: newCluster.Spec.Version.Semver().String(),
	}
	yamlManifest, err := yaml.Marshal(kubeOneClusterManifest)
	if err != nil {
		return nil, err
	}

	manifestSecret := &corev1.Secret{}
	if err := masterClient.Get(ctx, types.NamespacedName{Namespace: manifest.Namespace, Name: manifest.Name}, manifestSecret); err != nil {
		return nil, errors.NewBadRequest(fmt.Sprintf("can not retrieve kubeone manifest secret: %v", err))
	}
	oldManifestSecret := manifestSecret.DeepCopy()

	manifestSecret.Data = map[string][]byte{
		resources.KubeOneManifest: yamlManifest,
	}
	if err := masterClient.Patch(ctx, manifestSecret, ctrlruntimeclient.MergeFrom(oldManifestSecret)); err != nil {
		return nil, fmt.Errorf("failed to update kubeone manifest secret for upgrade version %s/%s: %w", manifest.Name, manifest.Namespace, err)
	}

	oldexternalCluster := externalCluster.DeepCopy()
	// update kubeone externalcluster status.
	externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.Status = kubermaticv1.StatusReconciling
	if upgradeMD == "true" {
		externalCluster.Spec.CloudSpec.KubeOne.ClusterStatus.StatusMessage = resources.KubeOneUpgradeMDMsg
	}
	// update api externalcluster status
	newCluster.Status.State = apiv2.RECONCILING
	if err := masterClient.Patch(ctx, externalCluster, ctrlruntimeclient.MergeFrom(oldexternalCluster)); err != nil {
		return nil, fmt.Errorf("failed to update kubeone cluster status %s: %w", externalCluster.Name, err)
	}

	return newCluster, nil
}
