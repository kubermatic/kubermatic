package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

// EtcdClientCertificateCreator returns a function to create/update the secret with the client certificate for authenticating against etcd
func EtcdClientCertificateCreator(data resources.SecretDataProvider) resources.SecretCreator {
	return certificates.GetClientCertificateCreator(
		"apiserver",
		nil,
		resources.ApiserverEtcdClientCertificateCertSecretKey,
		resources.ApiserverEtcdClientCertificateKeySecretKey,
		data.GetRootCA)
}
