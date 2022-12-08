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

package cloudinitsettings

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	cloudInitGetterToken      = "cloud-init-getter-token"
	cloudInitGetterAnnotation = "cloud-init-getter"
)

// SecretReconciler returns a function to create a secret in the usercluster, for generating an API server token against
// the cloud-init-getter service account.
func SecretReconciler() reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return cloudInitGetterToken, func(sec *corev1.Secret) (*corev1.Secret, error) {
			sec.Type = resources.ServiceAccountTokenType
			sec.Namespace = resources.CloudInitSettingsNamespace
			sec.Annotations = map[string]string{
				resources.ServiceAccountTokenAnnotation: cloudInitGetterAnnotation,
			}

			return sec, nil
		}
	}
}
