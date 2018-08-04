package certificates

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

type templateDataProvider interface {
	GetClusterRef() metav1.OwnerReference
	GetClusterCA() (*triple.KeyPair, error)
}

// GetClientCertificateCreator is a generic function to return a secret generator to create a client certificate signed by the cluster CA
func GetClientCertificateCreator(name, commonName string, organizations []string, dataCertKey, dataKeyKey string) func(data templateDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	return func(data templateDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
		var se *corev1.Secret
		if existing != nil {
			se = existing
		} else {
			se = &corev1.Secret{}
		}

		se.Name = name
		se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

		ca, err := data.GetClusterCA()
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster ca: %v", err)
		}

		if b, exists := se.Data[dataCertKey]; exists {
			certs, err := certutil.ParseCertsPEM(b)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret %s: %v", name, dataCertKey, err)
			}

			if resources.IsClientCertificateValidForAllOf(certs[0], commonName, organizations) {
				return se, nil
			}
		}

		newKP, err := triple.NewClientKeyPair(ca, commonName, organizations)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s key pair: %v", name, err)
		}

		se.Data = map[string][]byte{
			dataKeyKey:  certutil.EncodePrivateKeyPEM(newKP.Key),
			dataCertKey: certutil.EncodeCertPEM(newKP.Cert),
			// Include the CA for simplicity
			resources.CACertSecretKey: certutil.EncodeCertPEM(ca.Cert),
		}

		return se, nil
	}
}
