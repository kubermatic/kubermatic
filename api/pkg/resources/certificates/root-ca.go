package certificates

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

// RootCA returns the secret containing the root ca of a cluster
func RootCA(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	var se *corev1.Secret
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Secret{}
	}

	se.Name = resources.CASecretName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	if _, exists := se.Data[resources.CACertSecretKey]; exists {
		return se, nil
	}

	caKp, err := triple.NewCA(fmt.Sprintf("root-ca.%s", data.Cluster.Address.ExternalName))
	if err != nil {
		return nil, fmt.Errorf("unable to create a new CA: %v", err)
	}

	se.Data = map[string][]byte{
		resources.CAKeySecretKey:  certutil.EncodePrivateKeyPEM(caKp.Key),
		resources.CACertSecretKey: certutil.EncodeCertPEM(caKp.Cert),
	}

	return se, nil
}
