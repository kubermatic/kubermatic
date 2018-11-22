package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

	corev1 "k8s.io/api/core/v1"
)

// DexCACertificate returns a secret with the certificate for TLS verification against dex
func DexCACertificate(data resources.SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	return certificates.GetDexCACreator(
		resources.DexCASecretName,
		"apiserver",
		resources.DexCAFileName,
		data.GetDexCA)(data, existing)
}
