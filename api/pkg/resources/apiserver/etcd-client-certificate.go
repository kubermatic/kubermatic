package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

	corev1 "k8s.io/api/core/v1"
)

// EtcdClientCertificate returns a secret with the client certificate for authenticating against etcd
func EtcdClientCertificate(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	return certificates.GetClientCertificateCreator(
		resources.CASecretName,
		resources.ApiserverEtcdClientCertificateSecretName,
		"apiserver",
		nil,
		resources.ApiserverEtcdClientCertificateCertSecretKey,
		resources.ApiserverEtcdClientCertificateKeySecretKey)(data, existing)
}

// FrontProxyClientCertificate returns a secret with the client certificate for authenticating against extension apiserver
func FrontProxyClientCertificate(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	return certificates.GetClientCertificateCreator(
		resources.FrontProxyCASecretName,
		resources.ApiserverFrontProxyClientCertificateSecretName,
		"apiserver-aggregator",
		nil,
		resources.ApiserverProxyClientCertificateCertSecretKey,
		resources.ApiserverProxyClientCertificateKeySecretKey)(data, existing)
}
