package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

// InternalClientCertificateCreator returns a function to create/update the secret with a client certificate for the openvpn clients in the seed cluster.
func InternalClientCertificateCreator(data resources.SecretDataProvider) resources.SecretCreator {
	return certificates.GetECDSAClientCertificateCreator(
		resources.OpenVPNClientCertificatesSecretName,
		"internal-client",
		[]string{},
		resources.OpenVPNInternalClientCertSecretKey,
		resources.OpenVPNInternalClientKeySecretKey,
		data.GetOpenVPNCA,
	)
}

// UserClusterClientCertificateCreator returns a function to create/update the secret with the client certificate for the openvpn client in the user
// cluster
func UserClusterClientCertificateCreator(ca *resources.ECDSAKeyPair) resources.SecretCreator {
	return certificates.GetECDSAClientCertificateCreator(
		resources.OpenVPNClientCertificatesSecretName,
		"user-cluster-client",
		[]string{},
		resources.OpenVPNInternalClientCertSecretKey,
		resources.OpenVPNInternalClientKeySecretKey,
		func() (*resources.ECDSAKeyPair, error) { return ca, nil },
	)
}
