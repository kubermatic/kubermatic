package certificates

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// GetCACreator returns a function to create a secret containing a CA with the specified name
func GetCACreator(commonName string) reconciling.SecretCreator {
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

		se.Data[resources.CAKeySecretKey] = triple.EncodePrivateKeyPEM(caKp.Key)
		se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(caKp.Cert)

		return se, nil
	}
}

type caCreatorData interface {
	Cluster() *kubermaticv1.Cluster
}

// RootCACreator returns a function to create a secret with the root ca
func RootCACreator(data caCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CASecretName, GetCACreator(fmt.Sprintf("root-ca.%s", data.Cluster().Address.ExternalName))
	}
}

// FrontProxyCACreator returns a function to create a secret with front proxy ca
func FrontProxyCACreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.FrontProxyCASecretName, GetCACreator("front-proxy-ca")
	}
}
