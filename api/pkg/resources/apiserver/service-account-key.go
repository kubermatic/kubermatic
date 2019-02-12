package apiserver

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountKeyCreator returns a function to create/update a secret with the ServiceAccount key
func ServiceAccountKeyCreator() resources.SecretCreator {
	return func(se *corev1.Secret) (*corev1.Secret, error) {
		if _, exists := se.Data[resources.ServiceAccountKeySecretKey]; exists {
			return se, nil
		}
		priv, err := rsa.GenerateKey(cryptorand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		saKey := x509.MarshalPKCS1PrivateKey(priv)
		block := pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: saKey,
		}
		if se.Data == nil {
			se.Data = map[string][]byte{}
		}
		se.Data[resources.ServiceAccountKeySecretKey] = pem.EncodeToMemory(&block)
		return se, nil

	}
}
