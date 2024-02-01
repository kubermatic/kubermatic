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
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// Secretsreators returns the CSI secrets for KubeVirt.
func SecretsReconcilers(data *resources.TemplateData) []reconciling.NamedSecretReconcilerFactory {
	creators := []reconciling.NamedSecretReconcilerFactory{
		cloudConfigSecretNameReconciler(data),
		basicAuthSecretNameReconciler(data),
	}
	return creators
}

// CloudConfigSecretNameReconciler returns the CSI secrets for VMware Cloud Director cloud config.
func cloudConfigSecretNameReconciler(data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.CSICloudConfigSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			if s.Data == nil {
				s.Data = map[string][]byte{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			vcdCloudConfig, err := GetVMwareCloudDirectorCSIConfig(data.Cluster(), data.DC(), credentials)
			if err != nil {
				return nil, err
			}

			s.Labels = resources.BaseAppLabels(resources.CSICloudConfigSecretName, nil)
			s.Data[resources.CloudConfigKey] = []byte(vcdCloudConfig)

			return s, nil
		}
	}
}

// BasicAuthSecretNameReconciler returns the CSI secrets for VMware Cloud Director.
func basicAuthSecretNameReconciler(data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.VMwareCloudDirectorCSISecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			if s.Data == nil {
				s.Data = map[string][]byte{}
			}
			s.Labels = resources.BaseAppLabels(resources.VMwareCloudDirectorCSISecretName, nil)

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			if credentials.VMwareCloudDirector.APIToken != "" {
				s.Data["refreshToken"] = []byte(credentials.VMwareCloudDirector.APIToken)
			} else {
				s.Data["username"] = []byte(credentials.VMwareCloudDirector.Username)
				s.Data["password"] = []byte(credentials.VMwareCloudDirector.Password)
			}
			return s, nil
		}
	}
}
