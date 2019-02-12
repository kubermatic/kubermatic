package certificates

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

type caGetter func() (*triple.KeyPair, error)

// GetClientCertificateCreator is a generic function to return a secret generator to create a client certificate signed by the cluster CA
func GetClientCertificateCreator(commonName string, organizations []string, dataCertKey, dataKeyKey string, getCA caGetter) resources.SecretCreator {
	return func(se *corev1.Secret) (*corev1.Secret, error) {
		// TODO: Remove this after the backup controller has been adapter to the new reconciling behaviour
		if se == nil {
			se = &corev1.Secret{}
		}

		ca, err := getCA()
		if err != nil {
			return nil, fmt.Errorf("failed to get CA: %v", err)
		}

		if se.Data == nil {
			se.Data = map[string][]byte{}
		}

		if b, exists := se.Data[dataCertKey]; exists {
			certs, err := certutil.ParseCertsPEM(b)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", dataCertKey, err)
			}

			if resources.IsClientCertificateValidForAllOf(certs[0], commonName, organizations, ca.Cert) {
				return se, nil
			}
		}

		newKP, err := triple.NewClientKeyPair(ca, commonName, organizations)
		if err != nil {
			return nil, fmt.Errorf("failed to create key pair: %v", err)
		}

		se.Data[dataKeyKey] = certutil.EncodePrivateKeyPEM(newKP.Key)
		se.Data[dataCertKey] = certutil.EncodeCertPEM(newKP.Cert)
		// Include the CA for simplicity
		se.Data[resources.CACertSecretKey] = certutil.EncodeCertPEM(ca.Cert)

		return se, nil
	}
}
