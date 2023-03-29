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

package csisnapshotter

import (
	"fmt"

	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
)

func TLSServingCertificateReconciler(webhookName string, ca *triple.KeyPair) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.CSISnapshotWebhookSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			commonName := fmt.Sprintf("%s.%s.svc.cluster.local.", webhookName, metav1.NamespaceSystem)
			altNames := certutil.AltNames{
				DNSNames: []string{
					webhookName,
					fmt.Sprintf("%s.%s", webhookName, metav1.NamespaceSystem),
					commonName,
					fmt.Sprintf("%s.%s.svc", webhookName, metav1.NamespaceSystem),
					fmt.Sprintf("%s.%s.svc.", webhookName, metav1.NamespaceSystem),
				},
			}
			if b, exists := se.Data[resources.CSIWebhookServingCertCertKeyName]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.CSISnapshotWebhookSecretName, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}

			newKP, err := triple.NewServerKeyPair(ca,
				commonName,
				webhookName,
				metav1.NamespaceSystem,
				"",
				nil,
				[]string{commonName})
			if err != nil {
				return nil, fmt.Errorf("failed to generate serving cert: %w", err)
			}
			se.Data[resources.CSIWebhookServingCertCertKeyName] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.CSIWebhookServingCertKeyKeyName] = triple.EncodePrivateKeyPEM(newKP.Key)
			return se, nil
		}
	}
}
