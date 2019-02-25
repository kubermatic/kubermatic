package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

// DexCACertificateCreator returns a function to create/update the secret with the certificate for TLS verification against dex
func DexCACertificateCreator(data resources.SecretDataProvider) resources.NamedSecretCreatorGetter {
	return func() (string, resources.SecretCreator) {
		return resources.DexCASecretName, certificates.GetDexCACreator(
			resources.DexCAFileName,
			data.GetDexCA,
		)
	}
}
