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

package encryptionatrestcontroller

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	encryptionresources "k8c.io/kubermatic/v2/pkg/resources/encryption"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	apiserverconfigv1 "k8s.io/apiserver/pkg/apis/config/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func isApiserverUpdated(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (bool, error) {
	var secret corev1.Secret
	if err := client.Get(ctx, types.NamespacedName{
		Name:      resources.EncryptionConfigurationSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, &secret); err != nil {
		return false, ctrlruntimeclient.IgnoreNotFound(err)
	}

	spec, err := json.Marshal(cluster.Spec.EncryptionConfiguration)
	if err != nil {
		return false, err
	}

	hash := sha1.New()
	hash.Write(spec)

	if val, ok := secret.ObjectMeta.Labels[encryptionresources.ApiserverEncryptionHashLabelKey]; !ok || val != hex.EncodeToString(hash.Sum(nil)) {
		// the secret on the cluster (or in the cache) doesn't seem updated yet
		return false, nil
	}

	var podList corev1.PodList
	if err := client.List(ctx, &podList,
		ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName),
		ctrlruntimeclient.MatchingLabels{resources.AppLabelKey: "apiserver"},
	); err != nil {
		return false, err
	}

	if len(podList.Items) == 0 {
		return false, nil
	}

	for _, pod := range podList.Items {
		if val, ok := pod.Labels[encryptionresources.ApiserverEncryptionRevisionLabelKey]; !ok || val != secret.ResourceVersion {
			return false, nil
		}
	}

	return true, nil
}

// getActiveKey returns a key "hint" that is comprised of a provider-specific prefix and the key name. It does not return secret data.
func getActiveKey(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (string, error) {
	var (
		keyName string
		secret  corev1.Secret
		config  apiserverconfigv1.EncryptionConfiguration
	)

	if err := client.Get(ctx, types.NamespacedName{
		Name:      resources.EncryptionConfigurationSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, &secret); err != nil {
		return "", err
	}

	if data, ok := secret.Data[resources.EncryptionConfigurationKeyName]; ok {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return "", err
		}
	}

	// we expect two providers, (1) the configured encryption provider as per the ClusterSpec (secretbox or KMS plugins)
	// and (2) the "identity" provider, which is there for reading (and if at the top of the list, writing) resources as
	// unencrypted.
	if len(config.Resources) != 1 || len(config.Resources[0].Providers) != 1 || len(config.Resources[0].Providers) != 2 {
		return "", errors.New("unexpected apiserverconfigv1.EncryptionConfiguration: too many items in .resources or .resources[0].providers")
	}

	providerConfig := &config.Resources[0].Providers[0]

	switch {
	case providerConfig.Secretbox != nil:
		keyName = fmt.Sprintf("%s/%s", encryptionresources.SecretboxPrefix, providerConfig.Secretbox.Keys[0].Name)
	case providerConfig.Identity != nil:
		keyName = "identity"
	}

	return keyName, nil
}

// getConfiguredKey returns a key "hint" for the primary key as configured in a ClusterSpec. This can return a different result
// than `getActiveKey(ctx, client, cluster)`, because we are checking the specification (i.e. the target state), not the status
// (i.e. the current state). It does not return secret data.
func getConfiguredKey(cluster *kubermaticv1.Cluster) (string, error) {
	if cluster.Spec.EncryptionConfiguration == nil || !cluster.Spec.EncryptionConfiguration.Enabled {
		return "identity", nil
	}

	switch {
	case cluster.Spec.EncryptionConfiguration.Secretbox != nil:
		return fmt.Sprintf("%s/%s", encryptionresources.SecretboxPrefix, cluster.Spec.EncryptionConfiguration.Secretbox.Keys[0].Name), nil
	}

	return "", errors.New("no supported encryption provider found")
}
