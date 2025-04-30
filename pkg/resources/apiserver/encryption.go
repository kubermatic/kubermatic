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
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	encryptionresources "k8c.io/kubermatic/v2/pkg/resources/encryption"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiserverconfigv1 "k8s.io/apiserver/pkg/apis/apiserver/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type encryptionData interface {
	Cluster() *kubermaticv1.Cluster
	GetSecretKeyValue(ref *corev1.SecretKeySelector) ([]byte, error)
}

func EncryptionResourcesForDeletion(namespace string) []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.EncryptionConfigurationKeyName,
				Namespace: namespace,
			},
		},
	}
}

func EncryptionConfigurationSecretReconciler(data encryptionData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.EncryptionConfigurationSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			secret.Name = resources.EncryptionConfigurationSecretName

			// return empty secret if no config and no condition is set.
			if !data.Cluster().IsEncryptionEnabled() && !data.Cluster().IsEncryptionActive() {
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

			// Unmarshal existing configuration, if there was any.
			if val, ok := secret.Data[resources.EncryptionConfigurationKeyName]; ok {
				if err := yaml.Unmarshal(val, &existingConfig); err != nil {
					return secret, err
				}
			}

			if data.Cluster().IsEncryptionEnabled() {
				// handle active encryption configuration.

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

				// always append the "unencrypted" provider.
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
			} else {
				// encryptionConfiguration is not set; this means it was disabled and we need to rotate keys
				// to go back to 'identity', the "unencrypted" provider.

				// if the first provider is identity already, no further changes to the config are needed.
				if existingConfig.Resources[0].Providers[0].Identity != nil {
					return secret, nil
				}

				config = *existingConfig.DeepCopy()
				if len(config.Resources) != 1 {
					return nil, fmt.Errorf("malfored existing configuration, expected one entry for 'resources', got %d", len(config.Resources))
				}

				// rotate identity provider to the start of the list.
				providers := config.Resources[0].Providers[:len(config.Resources[0].Providers)-1]
				config.Resources[0].Providers = append([]apiserverconfigv1.ProviderConfiguration{
					{Identity: &apiserverconfigv1.IdentityConfiguration{}},
				}, providers...)
			}

			var err error

			secretData := map[string][]byte{}
			secretData[resources.EncryptionConfigurationKeyName], err = yaml.Marshal(config)
			if err != nil {
				return nil, err
			}

			secret.Data = secretData

			if secret.Labels == nil {
				secret.Labels = map[string]string{}
			}

			spec, err := json.Marshal(data.Cluster().Spec.EncryptionConfiguration)
			if err != nil {
				return nil, err
			}

			hash := sha1.New()
			hash.Write(spec)

			secret.Labels[encryptionresources.ApiserverEncryptionHashLabelKey] = hex.EncodeToString(hash.Sum(nil))

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
