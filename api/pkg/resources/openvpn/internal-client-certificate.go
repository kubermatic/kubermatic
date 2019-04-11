package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

type internalClientCertificateCreatorData interface {
	GetOpenVPNCA() (*resources.ECDSAKeyPair, error)
}

// InternalClientCertificateCreator returns a function to create/update the secret with a client certificate for the openvpn clients in the seed cluster.
func InternalClientCertificateCreator(data internalClientCertificateCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.OpenVPNClientCertificatesSecretName, certificates.GetECDSAClientCertificateCreator(
			resources.OpenVPNClientCertificatesSecretName,
			"internal-client",
			[]string{},
			resources.OpenVPNInternalClientCertSecretKey,
			resources.OpenVPNInternalClientKeySecretKey,
			data.GetOpenVPNCA,
		)
	}
}
