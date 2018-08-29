package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

	corev1 "k8s.io/api/core/v1"
)

// InternalClientCertificate returns a secret with a client certificate for the openvpn clients in the seed-cluster.
func InternalClientCertificate(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	return certificates.GetClientCertificateCreator(
		resources.OpenVPNClientCertificatesSecretName,
		"internal-client",
		[]string{},
		resources.OpenVPNInternalClientCertSecretKey,
		resources.OpenVPNInternalClientKeySecretKey,
		data.GetRootCA)(data, existing)
}
