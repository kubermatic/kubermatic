package apiserver

import (
	"crypto/x509"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// DexCACertificateCreator returns a function to create/update the secret with the certificate for TLS verification against dex
func DexCACertificateCreator(getDexCA func() ([]*x509.Certificate, error)) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.DexCASecretName, certificates.GetDexCACreator(
			resources.DexCAFileName,
			getDexCA,
		)
	}
}
