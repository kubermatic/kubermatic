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

package apiserver

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	encryptionresources "k8c.io/kubermatic/v2/pkg/resources/encryption"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiserverconfigv1 "k8s.io/apiserver/pkg/apis/config/v1"
	"sigs.k8s.io/yaml"
)

type encryptionData interface {
	Cluster() *kubermaticv1.Cluster
	GetSecretKeyValue(ref *corev1.SecretKeySelector) ([]byte, error)
}

func EncryptionConfigurationSecretCreator(data encryptionData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.EncryptionConfigurationSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			secret.Name = resources.EncryptionConfigurationSecretName

			// return empty secret if no config and no condition is set
			if !(data.Cluster().IsEncryptionEnabled() || data.Cluster().IsEncryptionActive()) {
				return secret, nil
			}

			// if encryption was initialized but re-encryption of data hasn't finished, do not update the secret.
			// we are waiting for kubermatic_encryption_controller to finish the re-encryption job. You do not want
			// to mess with the EncryptionConfiguration at that moment as introducing different keys might make the
			// encrypted data unreadable for kube-apiserver.
			if data.Cluster().IsEncryptionActive() &&
				data.Cluster().Status.Encryption != nil && data.Cluster().Status.Encryption.Phase == kubermaticv1.ClusterEncryptionPhaseEncryptionNeeded {
				return secret, nil
			}

			var existingConfig, config apiserverconfigv1.EncryptionConfiguration

			// Unmarshal existing configuration, if there was any
			if val, ok := secret.Data[resources.EncryptionConfigurationKeyName]; ok {
				if err := yaml.Unmarshal(val, &existingConfig); err != nil {
					return secret, err
				}
			}

			resourceList := data.Cluster().Spec.EncryptionConfiguration.Resources
			if len(resourceList) == 0 {
				resourceList = []string{"secrets"}
			}

			var providerList []apiserverconfigv1.ProviderConfiguration

			if data.Cluster().Spec.EncryptionConfiguration.Secretbox != nil {
				var existingKeys, secretboxKeys []apiserverconfigv1.Key

				if len(existingConfig.Resources) == 1 && len(existingConfig.Resources[0].Providers) == 2 &&
					existingConfig.Resources[0].Providers[0].Secretbox != nil {
					existingKeys = existingConfig.Resources[0].Providers[0].Secretbox.Keys
				}

				for _, key := range data.Cluster().Spec.EncryptionConfiguration.Secretbox.Keys {
					secretboxKey := apiserverconfigv1.Key{
						Name: key.Name,
					}

					// If a key with the given name exists, we will prefer the existing key value
					// over the cluster spec. This is done to prevent changing encryption keys in
					// place, as that is not supported and might result in unreadable resources.
					// This is especially aimed at keys read from Secret references, as those are
					// hard to keep track of.
					if existingKey := getKeyByName(existingKeys, key.Name); existingKey != nil {
						secretboxKey.Secret = existingKey.Secret
					} else {
						if key.SecretRef != nil {
							val, err := data.GetSecretKeyValue(key.SecretRef)
							if err != nil {
								return secret, err
							}
							secretboxKey.Secret = string(val)
						} else {
							secretboxKey.Secret = key.Value
						}
					}

					secretboxKeys = append(secretboxKeys, secretboxKey)
				}

				providerList = append(providerList, apiserverconfigv1.ProviderConfiguration{
					Secretbox: &apiserverconfigv1.SecretboxConfiguration{
						Keys: secretboxKeys,
					},
				})
			}

			// always append the "unencrypted" provider
			providerList = append(providerList, apiserverconfigv1.ProviderConfiguration{
				Identity: &apiserverconfigv1.IdentityConfiguration{},
			})

			config = apiserverconfigv1.EncryptionConfiguration{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiserver.config.k8s.io/v1",
					Kind:       "EncryptionConfiguration",
				},
				Resources: []apiserverconfigv1.ResourceConfiguration{
					{
						Resources: resourceList,
						Providers: providerList,
					},
				},
			}

			var err error

			secretData := map[string][]byte{}
			secretData[resources.EncryptionConfigurationKeyName], err = yaml.Marshal(config)
			if err != nil {
				return nil, err
			}

			secret.Data = secretData

			if secret.ObjectMeta.Labels == nil {
				secret.ObjectMeta.Labels = map[string]string{}
			}

			spec, err := json.Marshal(data.Cluster().Spec.EncryptionConfiguration)
			if err != nil {
				return nil, err
			}

			hash := sha1.New()
			hash.Write(spec)

			secret.ObjectMeta.Labels[encryptionresources.ApiserverEncryptionHashLabelKey] = hex.EncodeToString(hash.Sum(nil))

			return secret, nil
		}
	}
}

func getKeyByName(keys []apiserverconfigv1.Key, name string) *apiserverconfigv1.Key {
	for _, key := range keys {
		if key.Name == name {
			return &key
		}
	}

	return nil
}
