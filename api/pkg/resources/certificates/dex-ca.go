package certificates

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type dexCAGetter func() ([]*x509.Certificate, error)

// GetDexCACreator returns a function to create a secret containing a CA bundle with the specified name
func GetDexCACreator(dataCAKey string, getCA dexCAGetter) reconciling.SecretCreator {
	return func(se *corev1.Secret) (*corev1.Secret, error) {
		ca, err := getCA()
		if err != nil {
			return nil, fmt.Errorf("failed to get dex public keys: %v", err)
		}

		if se.Data == nil {
			se.Data = map[string][]byte{}
		}

		if _, exists := se.Data[dataCAKey]; exists {
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

		se.Data[dataCAKey] = cert
		return se, nil
	}
}
