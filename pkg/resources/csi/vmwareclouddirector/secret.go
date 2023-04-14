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

package vmwareclouddirector

import (
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// Secretsreators returns the CSI secrets for KubeVirt.
func SecretsReconcilers(data *resources.TemplateData) []reconciling.NamedSecretReconcilerFactory {
	creators := []reconciling.NamedSecretReconcilerFactory{
		CloudConfigSecretNameReconciler(data),
	}
	return creators
}

// CloudConfigSecretNameReconciler returns the CSI secrets for vmwareclouddirector.
func CloudConfigSecretNameReconciler(data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.CSICloudConfigSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			if cm.Data == nil {
				cm.Data = map[string][]byte{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			vcdCloudConfig, err := GetVMwareCloudDirectorCSIConfig(data.Cluster(), data.Datacenter(), credentials)
			if err != nil {
				return nil, err
			}

			cm.Labels = resources.BaseAppLabels(resources.CSICloudConfigSecretName, nil)
			cm.Data[resources.CloudConfigKey] = []byte(vcdCloudConfig)

			return cm, nil
		}
	}
}
