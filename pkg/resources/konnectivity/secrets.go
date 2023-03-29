/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package konnectivity

import (
	"encoding/json"
	"fmt"

	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

// ProxyKubeconfig returns kubeconfig for konnectivity proxy server.
func ProxyKubeconfig(data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.KonnectivityKubeconfigSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster CA: %w", err)
			}

			encodedCACert := triple.EncodeCertPEM(ca.Cert)
			address := data.Cluster().Status.Address

			oldConfData, exists := se.Data[resources.KonnectivityServerConf]

			if exists {
				oldConf := new(v1.Config)
				err := json.Unmarshal(oldConfData, &oldConf)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal old config: %w", err)
				}

				if len(oldConf.Clusters) == 0 {
					return nil, fmt.Errorf("old config has no clusters")
				}

				if oldConf.Clusters[0].Cluster.Server == address.URL &&
					string(oldConf.Clusters[0].Cluster.CertificateAuthorityData) == string(encodedCACert) {
					return se, nil
				}
			}

			clientKeyPair, err := triple.NewClientKeyPair(ca, "system:konnectivity-server", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create client key pair: %w", err)
			}

			konnectivityServerConf := v1.Config{
				Kind:       "Config",
				APIVersion: "v1",
				Clusters: []v1.NamedCluster{
					{
						Name: "kubernetes",
						Cluster: v1.Cluster{
							CertificateAuthorityData: encodedCACert,
							Server:                   address.URL,
						},
					},
				},
				AuthInfos: []v1.NamedAuthInfo{
					{
						Name: "system:konnectivity-server",
						AuthInfo: v1.AuthInfo{
							ClientCertificateData: triple.EncodeCertPEM(clientKeyPair.Cert),
							ClientKeyData:         triple.EncodePrivateKeyPEM(clientKeyPair.Key),
						},
					},
				},
				Contexts: []v1.NamedContext{
					{
						Name: "system:konnectivity-server@kubernetes",
						Context: v1.Context{
							Cluster:  "kubernetes",
							AuthInfo: "system:konnectivity-server",
						},
					},
				},
				CurrentContext: "system:konnectivity-server@kubernetes",
			}

			data, err := json.Marshal(konnectivityServerConf)
			if err != nil {
				return nil, fmt.Errorf("failed to marshall konnectivity-server config: %w", err)
			}

			se.Data = map[string][]byte{
				resources.KonnectivityServerConf: data,
			}

			return se, nil
		}
	}
}
