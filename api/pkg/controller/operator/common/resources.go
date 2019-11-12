package common

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

const (
	DockercfgSecretName                 = "dockercfg"
	DexCASecretName                     = "dex-ca"
	MasterFilesSecretName               = "extra-files"
	SeedAdmissionWebhookName            = "kubermatic.io-seeds"
	SeedWebhookServingCertSecretName    = "seed-webhook-serving-cert"
	seedWebhookCommonName               = "seed-webhook"
	seedWebhookCACertFile               = "caCert.pem"
	seedWebhookCertFile                 = "serverCert.pem"
	seedWebhookKeyFile                  = "serverKey.pem"
	IngressName                         = "kubermatic"
	SeedControllerManagerDeploymentName = "kubermatic-seed-controller-manager"
	OpenIDAuthFeatureFlag               = "OpenIDAuthPlugin"
)

func DockercfgSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return DockercfgSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson

			return createSecretData(s, map[string]string{
				corev1.DockerConfigJsonKey: cfg.Spec.ImagePullSecret,
			}), nil
		}
	}
}

func DexCASecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return DexCASecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, map[string]string{
				"caBundle.pem": cfg.Spec.Auth.CABundle,
			}), nil
		}
	}
}

func MasterFilesSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return MasterFilesSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			return createSecretData(s, cfg.Spec.MasterFiles), nil
		}
	}
}

func SeedWebhookServingCertSecretCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedSecretCreatorGetter {
	certConfig := certutil.Config{
		CommonName: seedWebhookCommonName,
		AltNames: certutil.AltNames{
			DNSNames: []string{
				fmt.Sprintf("%s.%s", seedWebhookCommonName, cfg.Namespace),
				fmt.Sprintf("%s.%s.svc", seedWebhookCommonName, cfg.Namespace),
			},
		},
		Usages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	return func() (string, reconciling.SecretCreator) {
		return SeedWebhookServingCertSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			// do not needlessly recreate new certificates on every reconcile
			if caCertPairValid(s, &certConfig) {
				return s, nil
			}

			// create new temporary CA keypair
			caKey, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("unable to create a private key for the seed serving cert CA: %v", err)
			}

			// create CA cert
			caCert, err := certutil.NewSelfSignedCACert(certConfig, caKey)
			if err != nil {
				return nil, fmt.Errorf("unable to create a self-signed certificate for a new seed serving cert CA: %v", err)
			}

			// create certificate keypair
			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("unable to create a serving cert key: %v", err)
			}

			// create certificate
			cert, err := triple.NewSignedCert(certConfig, key, caCert, caKey)
			if err != nil {
				return nil, fmt.Errorf("unable to sign serving certificate: %v", err)
			}

			return createSecretData(s, map[string]string{
				seedWebhookCACertFile: string(triple.EncodeCertPEM(caCert)),
				seedWebhookCertFile:   string(triple.EncodeCertPEM(cert)),
				seedWebhookKeyFile:    string(triple.EncodePrivateKeyPEM(key)),
			}), nil
		}
	}
}

func caCertPairValid(secret *corev1.Secret, config *certutil.Config) bool {
	// check if we have a certificate at all
	servingCertPEM := secret.Data[seedWebhookCertFile]
	servingKeyPEM := secret.Data[seedWebhookKeyFile]

	// this checks if cert and key are valid PEM encoded byte strings and
	// that the key matches the certificate;
	// sadly, the parsed cert PEM is not retained, so .Leaf is nil and we
	// have to parse the cert later on again
	if _, err := tls.X509KeyPair(servingCertPEM, servingKeyPEM); err != nil {
		return false
	}

	// check if we have a valid CA cert
	servingCerts, _ := certutil.ParseCertsPEM(servingCertPEM)
	caCertPEM := secret.Data[seedWebhookCACertFile]

	caCerts, err := certutil.ParseCertsPEM(caCertPEM)
	if err != nil {
		return false
	}

	// check if the cert is valid for the CA
	return resources.IsServerCertificateValidForAllOf(servingCerts[0], config.CommonName, config.AltNames, caCerts[0])
}
