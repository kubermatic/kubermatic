package openvpn

import (
	"crypto/x509"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
)

// TLSServingCertificate returns a secret with the openvpn server tls certificate
func TLSServingCertificate(data resources.SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	var se *corev1.Secret
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Secret{}
	}
	se.Name = resources.OpenVPNServerCertificatesSecretName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	if se.Data == nil {
		se.Data = map[string][]byte{}
	}

	ca, err := data.GetRootCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster ca: %v", err)
	}
	altNames := certutil.AltNames{}
	if b, exists := se.Data[resources.OpenVPNServerCertSecretKey]; exists {
		certs, err := certutil.ParseCertsPEM(b)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.OpenVPNServerCertSecretKey, err)
		}
		if resources.IsServerCertificateValidForAllOf(certs[0], "openvpn-server", altNames) {
			return se, nil
		}
	}
	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to create a server private key: %v", err)
	}
	config := certutil.Config{
		CommonName: "openvpn-server",
		AltNames:   altNames,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	cert, err := certutil.NewSignedCert(config, key, ca.Cert, ca.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
	}

	se.Data[resources.OpenVPNServerKeySecretKey] = certutil.EncodePrivateKeyPEM(key)
	se.Data[resources.OpenVPNServerCertSecretKey] = certutil.EncodeCertPEM(cert)

	return se, nil
}
