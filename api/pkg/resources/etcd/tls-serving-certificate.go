package etcd

import (
	"crypto/x509"
	"fmt"
	"net"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

type tlsCertificateCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetRootCA() (*triple.KeyPair, error)
}

// TLSCertificateCreator returns a function to create/update the secret with the etcd tls certificate
func TLSCertificateCreator(data tlsCertificateCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.EtcdTLSCertificateSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}

			altNames := certutil.AltNames{
				DNSNames: []string{
					"localhost",
				},
				IPs: []net.IP{
					net.ParseIP("127.0.0.1"),
				},
			}

			for i := 0; i < 3; i++ {
				// Member name
				podName := fmt.Sprintf("etcd-%d", i)
				altNames.DNSNames = append(altNames.DNSNames, podName)

				// Pod DNS name
				absolutePodDNSName := fmt.Sprintf("etcd-%d.%s.%s.svc.cluster.local", i, resources.EtcdServiceName, data.Cluster().Status.NamespaceName)
				altNames.DNSNames = append(altNames.DNSNames, absolutePodDNSName)
			}

			if b, exists := se.Data[resources.EtcdTLSCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret %s: %v", resources.EtcdTLSCertSecretKey, resources.EtcdTLSCertificateSecretName, err)
				}

				if resources.IsServerCertificateValidForAllOf(certs[0], "etcd", altNames, ca.Cert) {
					return se, nil
				}
			}

			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to create private key for etcd server tls certificate: %v", err)
			}

			config := certutil.Config{
				CommonName: "etcd",
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}

			cert, err := triple.NewSignedCert(config, key, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			se.Data[resources.EtcdTLSKeySecretKey] = triple.EncodePrivateKeyPEM(key)
			se.Data[resources.EtcdTLSCertSecretKey] = triple.EncodeCertPEM(cert)

			return se, nil
		}
	}
}
