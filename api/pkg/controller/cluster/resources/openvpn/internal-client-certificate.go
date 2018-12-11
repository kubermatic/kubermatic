package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

	corev1 "k8s.io/api/core/v1"
)

// InternalClientCertificate returns a secret with a client certificate for the openvpn clients in the seed cluster.
func InternalClientCertificate(data resources.SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	return certificates.GetECDSAClientCertificateCreatorWithOwnerRef(
		resources.OpenVPNClientCertificatesSecretName,
		"internal-client",
		[]string{},
		resources.OpenVPNInternalClientCertSecretKey,
		resources.OpenVPNInternalClientKeySecretKey,
		data.GetOpenVPNCA)(data, existing)
}

// UserClusterClientCertificate returns a secret with the client certificate for the openvpn client in the user
// cluster
func UserClusterClientCertificate(existing *corev1.Secret, ca *resources.ECDSAKeyPair) (*corev1.Secret, error) {
	return certificates.GetECDSAClientCertificateCreator(
		resources.OpenVPNClientCertificatesSecretName,
		"user-cluster-client",
		[]string{},
		resources.OpenVPNInternalClientCertSecretKey,
		resources.OpenVPNInternalClientKeySecretKey,
		ca)(existing)
}
