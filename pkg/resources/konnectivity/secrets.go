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
	"context"
	"encoding/json"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd/api/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AgentTokenCreator steals the service account token from konnectivity agent in user-cluster to be used in seed-cluster
// by the konnectivity agent in the seed-cluster.
func AgentTokenCreator(userClusterClient ctrlruntimeclient.Client) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.KonnectivityStolenAgentTokenSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {

			sa := new(corev1.ServiceAccount)
			err := userClusterClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{
				Namespace: "kube-system",
				Name:      resources.KonnectivityServiceAccountName,
			}, sa)

			if err != nil {
				return nil, fmt.Errorf("failed to get konnectivity-service-account: %v", err)
			}

			if len(sa.Secrets) == 0 {
				return nil, fmt.Errorf("konnectivity-service-account has no secrets")
			}

			tokenName := sa.Secrets[0].Name
			s := new(corev1.Secret)
			err = userClusterClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{
				Namespace: "kube-system",
				Name:      tokenName,
			}, s)

			if err != nil {
				return nil, fmt.Errorf("failed to get konnectivity-service-account secret token: %v", err)
			}

			caCrt, ok := s.Data["ca.crt"]
			if !ok {
				return nil, fmt.Errorf("failed to get konnectivity-service-account secret token doesn't have ca.crt: %v", err)
			}

			token, ok := s.Data["token"]
			if !ok {
				return nil, fmt.Errorf("failed to get konnectivity-service-account secret token doesn't have token: %v", err)
			}

			cm.Data = map[string][]byte{
				resources.KonnectivityStolenAgentTokenNameCert:  caCrt,
				resources.KonnectivityStolenAgentTokenNameToken: token,
			}
			return cm, nil
		}
	}
}

// ProxyKubeconfig returns kubeconfig for konnectivity proxy server.
func ProxyKubeconfig(data *resources.TemplateData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.KonnectivityKubeconfigSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if _, exists := se.Data[resources.KonnectivityServerConf]; exists {
				return se, nil
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster CA: %v", err)
			}

			clientKeyPair, err := triple.NewClientKeyPair(ca, "system:konnectivity-server", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create client key pair: %v", err)
			}

			konnectivityServerConf := v1.Config{
				Kind:       "Config",
				APIVersion: "v1",
				Clusters: []v1.NamedCluster{
					{
						Name: "kubernetes",
						Cluster: v1.Cluster{
							CertificateAuthorityData: triple.EncodeCertPEM(ca.Cert),
							Server:                   data.Cluster().Address.URL,
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
				return nil, fmt.Errorf("failed to marshall konnectivity-server config: %v", err)
			}

			se.Data = map[string][]byte{
				resources.KonnectivityServerConf: data,
			}

			return se, nil
		}
	}
}
