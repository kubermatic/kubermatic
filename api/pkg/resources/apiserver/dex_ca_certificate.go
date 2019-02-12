package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

// DexCACertificateCreator returns a function to create/update the secret with the certificate for TLS verification against dex
func DexCACertificateCreator(data resources.SecretDataProvider) resources.SecretCreator {
	return certificates.GetDexCACreator(
		resources.DexCAFileName,
		data.GetDexCA)
}
