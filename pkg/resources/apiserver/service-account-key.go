package apiserver

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountKeyCreator returns a function to create/update a secret with the ServiceAccount key
func ServiceAccountKeyCreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.ServiceAccountKeySecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if _, exists := se.Data[resources.ServiceAccountKeySecretKey]; exists {
				return se, nil
			}
			priv, err := rsa.GenerateKey(cryptorand.Reader, 2048)
			if err != nil {
				return nil, err
			}
			saKey := x509.MarshalPKCS1PrivateKey(priv)
			privKeyBlock := pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: saKey,
			}
			publicKeyDer, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
			if err != nil {
				return nil, err
			}
			publicKeyBlock := pem.Block{
				Type:    "PUBLIC KEY",
				Headers: nil,
				Bytes:   publicKeyDer,
			}
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			se.Data[resources.ServiceAccountKeySecretKey] = pem.EncodeToMemory(&privKeyBlock)
			se.Data[resources.ServiceAccountKeyPublicKey] = pem.EncodeToMemory(&publicKeyBlock)
			return se, nil

		}
	}

}
