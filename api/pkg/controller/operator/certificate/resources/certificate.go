package resources

import (
	"errors"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
)

const (
	CertificateName       = "kubermatic"
	CertificateSecretName = "kubermatic-tls"
)

func CertificateCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedCertificateCreatorGetter {
	return func() (string, reconciling.CertificateCreator) {
		return CertificateName, func(c *certmanagerv1alpha2.Certificate) (*certmanagerv1alpha2.Certificate, error) {
			name := cfg.Spec.CertificateIssuer.Name
			if name == "" {
				return nil, errors.New("no certificateIssuer configured in KubermaticConfiguration")
			}

			// cert-manager's default is Issuer, but since we do not create an Issuer,
			// it does not make sense to force to change the configuration for the
			// default case
			kind := cfg.Spec.CertificateIssuer.Kind
			if kind == "" {
				kind = "ClusterIssuer"
			}

			c.Spec.IssuerRef.Name = name
			c.Spec.IssuerRef.Kind = kind

			if group := cfg.Spec.CertificateIssuer.APIGroup; group != nil {
				c.Spec.IssuerRef.Group = *group
			}

			c.Spec.SecretName = CertificateSecretName
			c.Spec.DNSNames = []string{cfg.Spec.Domain}

			return c, nil
		}
	}
}
