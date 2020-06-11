package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// ClientCertificate returns a function to create/update the secret with the client certificate
// for the openvpn client in the user cluster
func ClientCertificate(ca *resources.ECDSAKeyPair) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.OpenVPNClientCertificatesSecretName,
			certificates.GetECDSAClientCertificateCreator(
				resources.OpenVPNClientCertificatesSecretName,
				"user-cluster-client",
				[]string{},
				resources.OpenVPNInternalClientCertSecretKey,
				resources.OpenVPNInternalClientKeySecretKey,
				func() (*resources.ECDSAKeyPair, error) { return ca, nil })
	}
}
