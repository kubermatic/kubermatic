package certificates

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

type caGetter func() (*triple.KeyPair, error)

// GetClientCertificateCreator is a generic function to return a secret generator to create a client certificate signed by the cluster CA
func GetClientCertificateCreator(name, commonName string, organizations []string, dataCertKey, dataKeyKey string, getCA caGetter) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return name, func(se *corev1.Secret) (*corev1.Secret, error) {
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

			se.Data[dataKeyKey] = triple.EncodePrivateKeyPEM(newKP.Key)
			se.Data[dataCertKey] = triple.EncodeCertPEM(newKP.Cert)
			// Include the CA for simplicity
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)

			return se, nil
		}
	}
}
