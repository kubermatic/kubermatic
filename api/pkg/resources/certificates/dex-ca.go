package certificates

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dexCAGetter func() ([]*x509.Certificate, error)

// GetDexCACreator returns a function to create a secret containing a CA bundle with the specified name
func GetDexCACreator(name, commonName, dataCAKey string, getCA dexCAGetter) func(resources.SecretDataProvider, *corev1.Secret) (*corev1.Secret, error) {
	return func(data resources.SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
		var se *corev1.Secret
		if existing != nil {
			se = existing
		} else {
			se = &corev1.Secret{}
		}
		se.Name = name
		se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

		ca, err := getCA()
		if err != nil {
			return nil, fmt.Errorf("failed to get dex public keys: %v", err)
		}

		if se.Data == nil {
			se.Data = map[string][]byte{}
		}

		if _, exists := se.Data[resources.DexCAFileName]; exists {
			return se, nil
		}

		var cert []byte
		for _, certRaw := range ca {
			block := pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certRaw.Raw,
			}
			cert = append(cert, pem.EncodeToMemory(&block)...)
		}

		se.Data[resources.DexCAFileName] = cert
		return se, nil
	}
}
