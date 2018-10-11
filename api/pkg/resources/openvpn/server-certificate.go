package openvpn

import (
	"crypto/x509"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

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

	ca, err := data.GetOpenVPNCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get openvpn ca: %v", err)
	}
	altNames := certutil.AltNames{}
	if b, exists := se.Data[resources.OpenVPNServerCertSecretKey]; exists {
		certs, err := certutil.ParseCertsPEM(b)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.OpenVPNServerCertSecretKey, err)
		}
		if resources.IsServerCertificateValidForAllOf(certs[0], "openvpn-server", altNames, ca.Cert) {
			return se, nil
		}
	}
	config := certutil.Config{
		CommonName: "openvpn-server",
		AltNames:   altNames,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	cert, key, err := certificates.GetSignedECDSACertAndKey(certificates.Duration365d, config, ca.Cert, ca.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
	}

	se.Data[resources.OpenVPNServerCertSecretKey] = cert
	se.Data[resources.OpenVPNServerKeySecretKey] = key

	return se, nil
}

// CertificateAuthority returns a secret that holds the ECDSA-based CA to be used for OpenVPN
func CertificateAuthority(data resources.SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	var se *corev1.Secret
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Secret{}
	}
	se.Name = resources.OpenVPNCASecretName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	if se.Data == nil {
		se.Data = map[string][]byte{}
	}

	if data, exists := se.Data[resources.OpenVPNCACertKey]; exists {
		certs, err := certutil.ParseCertsPEM(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate %s from existing secret %s: %v",
				resources.OpenVPNCACertKey, resources.OpenVPNCASecretName, err)
		}
		if !resources.CertWillExpireSoon(certs[0]) {
			return se, nil
		}
	}

	cert, key, err := certificates.GetECDSACACertAndKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OpenVPN CA: %v", err)
	}
	se.Data[resources.OpenVPNCACertKey] = cert
	se.Data[resources.OpenVPNCAKeyKey] = key

	return se, nil
}
