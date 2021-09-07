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

package csi_migration

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
)

var webhookConfig = fmt.Sprintf(`
[WebHookConfig]
port = "%d"
cert-file = "/etc/webhook/cert.pem"
key-file = "/etc/webhook/key.pem"
`, resources.CSIMigrationWebhookPort)

func TLSServingCertificateCreator(ca *triple.KeyPair) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CSIMigrationWebhookSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			commonName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.CSIMigrationWebhookName, metav1.NamespaceSystem)
			altNames := certutil.AltNames{
				DNSNames: []string{
					resources.CSIMigrationWebhookName,
					fmt.Sprintf("%s.%s", resources.CSIMigrationWebhookName, metav1.NamespaceSystem),
					commonName,
					fmt.Sprintf("%s.%s.svc", resources.CSIMigrationWebhookName, metav1.NamespaceSystem),
					fmt.Sprintf("%s.%s.svc.", resources.CSIMigrationWebhookName, metav1.NamespaceSystem),
				},
			}
			if b, exists := se.Data[resources.CSIMigrationWebhookServingCertCertKeyName]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.CSIMigrationWebhookSecretName, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}

			newKP, err := triple.NewServerKeyPair(ca,
				commonName,
				resources.CSIMigrationWebhookName,
				metav1.NamespaceSystem,
				"",
				nil,
				// For some reason the name the APIServer validates against must be in the SANs, having it as CN is not enough
				[]string{commonName})
			if err != nil {
				return nil, fmt.Errorf("failed to generate serving cert: %v", err)
			}
			se.Data[resources.CSIMigrationWebhookServingCertCertKeyName] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.CSIMigrationWebhookServingCertKeyKeyName] = triple.EncodePrivateKeyPEM(newKP.Key)
			// Include the CA for simplicity
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)
			se.Data[resources.CSIMigrationWebhookConfig] = []byte(webhookConfig)
			return se, nil
		}
	}
}
