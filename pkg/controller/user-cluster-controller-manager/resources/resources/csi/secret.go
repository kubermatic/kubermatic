package csi

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

func TLSServingCertificateCreator(webhookName string, ca *triple.KeyPair) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CSIWebhookSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
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
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.CSIWebhookSecretName, err)
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
