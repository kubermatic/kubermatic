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

package encryption

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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
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
	if err := client.List(ctx, &podList, []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName),
		ctrlruntimeclient.MatchingLabels{resources.AppLabelKey: "apiserver"},
	}...); err != nil {
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

	if len(config.Resources) != 1 || len(config.Resources[0].Providers) != 1 {
		return "", errors.New("unexpected apiserverconfigv1.EncryptionConfiguration: too many items in .resources or .resources[0].providers")
	}

	providerConfig := &config.Resources[0].Providers[0]

	switch {
	case providerConfig.Secretbox != nil:
		keyName = fmt.Sprintf("%s/%s", encryptionresources.SecretboxPrefix, providerConfig.Secretbox.Keys[0].Name)
	}

	return keyName, nil
}
