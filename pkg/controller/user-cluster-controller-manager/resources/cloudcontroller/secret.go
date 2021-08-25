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

package cloudcontroller

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// CloudConfig generates the cloud-config secret to be injected in the user cluster.
func CloudConfig(cloudConfig []byte, secretName string) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return secretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Name = secretName
			if existing.Data == nil {
				existing.Data = map[string][]byte{}
			}
			existing.Data[resources.CloudConfigSecretKey] = cloudConfig
			return existing, nil
		}
	}
}
