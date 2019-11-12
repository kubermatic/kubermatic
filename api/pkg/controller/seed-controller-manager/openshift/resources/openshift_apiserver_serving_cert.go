package resources

import (
	"crypto/x509"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

const (
	openshiftAPIServerTLSServingCertSecretName = "openshift-apiserver-serving-cert"
	openshiftAPIServerCN                       = "api.openshift-apiserver.svc"
)

type tlsServingCertCreatorData interface {
	GetRootCA() (*triple.KeyPair, error)
}

func OpenShiftTLSServingCertificateCreator(data tlsServingCertCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return openshiftAPIServerTLSServingCertSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}

			altNames := certutil.AltNames{
				DNSNames: []string{
					"api.openshift-apiserver.svc",
					"api.openshift-apiserver.svc.cluster.local",
				},
			}
			if b, exists := se.Data[resources.ApiserverTLSCertSecretKey]; exists {
				certs, err := triple.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.ApiserverTLSCertSecretKey, err)
				}

				if resources.IsServerCertificateValidForAllOf(certs[0], openshiftAPIServerCN, altNames, ca.Cert) {
					return se, nil
				}
			}

			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("unable to create a server private key: %v", err)
			}

			config := certutil.Config{
				CommonName: openshiftAPIServerCN,
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}

			cert, err := triple.NewSignedCert(config, key, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			se.Data[resources.ApiserverTLSKeySecretKey] = triple.EncodePrivateKeyPEM(key)
			se.Data[resources.ApiserverTLSCertSecretKey] = triple.EncodeCertPEM(cert)

			return se, nil
		}
	}
}
