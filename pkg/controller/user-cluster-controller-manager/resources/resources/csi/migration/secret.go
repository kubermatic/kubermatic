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

package csimigration

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

const webhookConfig = `
[WebHookConfig]
port = "8443"
cert-file = "/run/secrets/tls/cert.pem"
key-file = "/run/secrets/tls/key.pem"
`

func VsphereTLSServingCertificateReconciler(ca *triple.KeyPair) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.VSphereCSIWebhookSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			commonName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.VSphereCSIValidatingWebhookServiceName, resources.VSphereCSINamespace)
			altNames := certutil.AltNames{
				DNSNames: []string{
					resources.VSphereCSIValidatingWebhookServiceName,
					fmt.Sprintf("%s.%s", resources.VSphereCSIValidatingWebhookServiceName, resources.VSphereCSINamespace),
					commonName,
					fmt.Sprintf("%s.%s.svc", resources.VSphereCSIValidatingWebhookServiceName, resources.VSphereCSINamespace),
					fmt.Sprintf("%s.%s.svc.", resources.VSphereCSIValidatingWebhookServiceName, resources.VSphereCSINamespace),
				},
			}
			if b, exists := se.Data[resources.CSIWebhookServingCertCertKeyName]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.VSphereCSIWebhookSecretName, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}

			newKP, err := triple.NewServerKeyPair(ca,
				commonName,
				resources.VSphereCSIValidatingWebhookServiceName,
				resources.VSphereCSINamespace,
				"",
				nil,
				[]string{commonName})
			if err != nil {
				return nil, fmt.Errorf("failed to generate serving cert: %w", err)
			}
			se.Data[resources.CSIWebhookServingCertCertKeyName] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.CSIWebhookServingCertKeyKeyName] = triple.EncodePrivateKeyPEM(newKP.Key)
			// Include the CA for simplicity
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)
			se.Data[resources.VSphereCSIValidatingWebhookConfigKey] = []byte(webhookConfig)
			return se, nil
		}
	}
}
