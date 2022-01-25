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
	"errors"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiserverconfigv1 "k8s.io/apiserver/pkg/apis/config/v1"
	"sigs.k8s.io/yaml"
)

func EncryptionConfigurationSecretCreator(data *resources.TemplateData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.EncryptionConfigurationSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			secret.Name = resources.EncryptionConfigurationSecretName

			// return empty secret if no config and no condition is set
			if !data.IsEncryptionConfigurationEnabled() ||
				data.Cluster().Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionFalse) {
				return secret, nil
			}

			// if encryption was initialized but re-encryption of data hasn't finished, do not update the secret.
			// we are waiting for kubermatic_encryption_controller to finish the job.
			if data.Cluster().Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionTrue) &&
				data.Cluster().Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionFinished, corev1.ConditionFalse) {
				return secret, nil
			}

			if data.IsEncryptionConfigurationEnabled() && data.Cluster().Spec.EncryptionConfiguration.Secretbox == nil {
				return nil, errors.New("cannot configure encryption, encryption is enabled but no secretbox key is provided")
			}

			var config apiserverconfigv1.EncryptionConfiguration

			resourceList := data.Cluster().Spec.EncryptionConfiguration.Resources
			if len(resourceList) == 0 {
				resourceList = []string{"secrets"}
			}

			var providerList []apiserverconfigv1.ProviderConfiguration

			if data.Cluster().Spec.EncryptionConfiguration.Secretbox != nil {
				var secretboxKeys []apiserverconfigv1.Key

				for _, key := range data.Cluster().Spec.EncryptionConfiguration.Secretbox.Keys {
					secretboxKeys = append(secretboxKeys, apiserverconfigv1.Key{
						Name: key.Name,
						// TODO: read from Secret if secretRef exists
						Secret: key.Value,
					})
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

			return secret, nil
		}
	}
}
