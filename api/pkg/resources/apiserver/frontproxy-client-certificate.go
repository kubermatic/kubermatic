package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

// FrontProxyClientCertificateCreator returns a function to create/update the secret with the client certificate for authenticating against extension apiserver
func FrontProxyClientCertificateCreator(data resources.SecretDataProvider) resources.SecretCreator {
	return certificates.GetClientCertificateCreator(
		"apiserver-aggregator",
		nil,
		resources.ApiserverProxyClientCertificateCertSecretKey,
		resources.ApiserverProxyClientCertificateKeySecretKey,
		data.GetFrontProxyCA)
}
