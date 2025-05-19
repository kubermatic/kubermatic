//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package admissioncontrollerresources

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

// tlsServingCertReconcilerData contains the minimal subset of the controller-runtime manager that
// is required to reconcile TLS secrets for the Kyverno admission controller.
// It intentionally mirrors the interface used by other reconcilers (e.g. operating-system-manager)
// to keep the implementation consistent across components.
type tlsServingCertReconcilerData interface {
	// GetRootCA returns the cluster root CA which is used to sign the server certificate.
	GetRootCA() (*triple.KeyPair, error)
	// Cluster returns the Cluster object that is being reconciled.
	Cluster() *kubermaticv1.Cluster
}

// TLSServingCAReconciler returns a function to create or update the secret that contains the CA
// certificate used by the Kyverno admission controller. The secret is named
//
//	kyverno-svc.<cluster-namespace>.svc.kyverno-tls-ca
//
// and only contains the `ca.crt` key.
func TLSServingCAReconciler(data tlsServingCertReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		secretName := fmt.Sprintf("%s.%s.svc.kyverno-tls-ca", serviceName, data.Cluster().Status.NamespaceName)
		return secretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get root ca: %w", err)
			}

			// Keep the existing secret if it already contains the expected certificate.
			if b, exists := se.Data[resources.CACertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.CACertSecretKey, err)
				}
				if len(certs) > 0 && certs[0].Equal(ca.Cert) {
					return se, nil
				}
			}

			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)
			return se, nil
		}
	}
}

// TLSServingCertificateReconciler returns a function to create or update the secret that contains
// the TLS key-pair (server certificate and private key) for the Kyverno admission controller.
// The secret is named
//
//	kyverno-svc.<cluster-namespace>.svc.kyverno-tls-pair
//
// It contains the following keys:
//
//	tls.crt – the PEM-encoded server certificate
//	tls.key – the PEM-encoded private key
//	ca.crt  – the PEM-encoded CA certificate (for convenience)
func TLSServingCertificateReconciler(data tlsServingCertReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		secretName := fmt.Sprintf("%s.%s.svc.kyverno-tls-pair", serviceName, data.Cluster().Status.NamespaceName)
		return secretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get root ca: %w", err)
			}

			commonName := fmt.Sprintf("%s.%s.svc.cluster.local.", serviceName, data.Cluster().Status.NamespaceName)
			altNames := certutil.AltNames{
				DNSNames: []string{
					serviceName,
					fmt.Sprintf("%s.%s", serviceName, data.Cluster().Status.NamespaceName),
					commonName,
					fmt.Sprintf("%s.%s.svc", serviceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc.", serviceName, data.Cluster().Status.NamespaceName),
				},
			}

			// Re-use existing certificate if it is still valid for all required SANs.
			if b, exists := se.Data[resources.OperatingSystemManagerWebhookServingCertCertKeyName]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.OperatingSystemManagerWebhookServingCertCertKeyName, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}

			newKP, err := triple.NewServerKeyPair(ca,
				commonName,
				serviceName,
				data.Cluster().Status.NamespaceName,
				"",
				nil,
				// The APIServer requires the CommonName to also be present in the SAN extension.
				[]string{commonName})
			if err != nil {
				return nil, fmt.Errorf("failed to generate serving cert: %w", err)
			}

			se.Data[resources.OperatingSystemManagerWebhookServingCertCertKeyName] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.OperatingSystemManagerWebhookServingCertKeyKeyName] = triple.EncodePrivateKeyPEM(newKP.Key)
			// Include the CA certificate for convenience.
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)

			return se, nil
		}
	}
}
