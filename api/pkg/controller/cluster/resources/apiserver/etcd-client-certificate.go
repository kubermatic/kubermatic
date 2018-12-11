package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

	corev1 "k8s.io/api/core/v1"
)

// EtcdClientCertificate returns a secret with the client certificate for authenticating against etcd
func EtcdClientCertificate(data resources.SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	return certificates.GetClientCertificateCreator(
		resources.ApiserverEtcdClientCertificateSecretName,
		"apiserver",
		nil,
		resources.ApiserverEtcdClientCertificateCertSecretKey,
		resources.ApiserverEtcdClientCertificateKeySecretKey,
		data.GetRootCA)(data, existing)
}
