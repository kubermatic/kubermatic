package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

type frontProxyClientCertificateCreatorData interface {
	GetFrontProxyCA() (*triple.KeyPair, error)
}

// FrontProxyClientCertificateCreator returns a function to create/update the secret with the client certificate for authenticating against extension apiserver
func FrontProxyClientCertificateCreator(data frontProxyClientCertificateCreatorData) reconciling.NamedSecretCreatorGetter {
	return certificates.GetClientCertificateCreator(
		resources.ApiserverFrontProxyClientCertificateSecretName,
		"apiserver-aggregator",
		nil,
		resources.ApiserverProxyClientCertificateCertSecretKey,
		resources.ApiserverProxyClientCertificateKeySecretKey,
		data.GetFrontProxyCA)

}
