package certificates

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

// GetCACreator returns a function to create a secret containing a CA with the specified name
func getCACreator(name, commonName string) func(*resources.TemplateData, *corev1.Secret) (*corev1.Secret, error) {
	return func(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
		var se *corev1.Secret
		if existing != nil {
			se = existing
		} else {
			se = &corev1.Secret{}
		}
		se.Name = name
		se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

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

// RootCA returns a function to create a secret with a ca
func RootCA(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	create := getCACreator(resources.CASecretName, fmt.Sprintf("root-ca.%s", data.Cluster.Address.ExternalName))

	return create(data, existing)
}

// FrontProxyCA returns a function to create a secret with a ca
func FrontProxyCA(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	create := getCACreator(resources.FrontProxyCASecretName, resources.FrontProxyCASecretName)

	return create(data, existing)
}
