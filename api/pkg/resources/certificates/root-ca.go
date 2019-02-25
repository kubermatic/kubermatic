package certificates

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

// GetCACreator returns a function to create a secret containing a CA with the specified name
func getCACreator(commonName string) resources.SecretCreator {
	return func(se *corev1.Secret) (*corev1.Secret, error) {
		if se.Data == nil {
			se.Data = map[string][]byte{}
		}

		if _, exists := se.Data[resources.CACertSecretKey]; exists {
			return se, nil
		}

		caKp, err := triple.NewCA(commonName)
		if err != nil {
			return nil, fmt.Errorf("unable to create a new CA: %v", err)
		}

		se.Data[resources.CAKeySecretKey] = certutil.EncodePrivateKeyPEM(caKp.Key)
		se.Data[resources.CACertSecretKey] = certutil.EncodeCertPEM(caKp.Cert)

		return se, nil
	}
}

// RootCACreator returns a function to create a secret with the root ca
func RootCACreator(data resources.SecretDataProvider) resources.NamedSecretCreatorGetter {
	return func() (string, resources.SecretCreator) {
		return resources.CASecretName, getCACreator(fmt.Sprintf("root-ca.%s", data.Cluster().Address.ExternalName))
	}
}

// FrontProxyCACreator returns a function to create a secret with front proxy ca
func FrontProxyCACreator(data resources.SecretDataProvider) resources.NamedSecretCreatorGetter {
	return func() (string, resources.SecretCreator) {
		return resources.FrontProxyCASecretName, getCACreator("front-proxy-ca")
	}
}
