package certificates

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
)

// Duration365d is a time.Duration that represents a year
const Duration365d = time.Hour * 24 * 365

type ecdsaCAGetter func() (*resources.ECDSAKeyPair, error)

// GetECDSAClientCertificateCreatorWithOwnerRef is a generic function to return a secret generator to create a client certificate
// signed by the cert returned by the passed getCA func. The resulting secret has an ownerRef
// pointing to the cluster object in data.GetClusterRef
func GetECDSAClientCertificateCreatorWithOwnerRef(name, commonName string, organizations []string, dataCertKey, dataKeyKey string, getCA ecdsaCAGetter) func(data templateDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {

	return func(data templateDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
		ca, err := getCA()
		if err != nil {
			return nil, err
		}
		secret, err := GetECDSAClientCertificateCreator(name, commonName, organizations, dataCertKey, dataKeyKey, ca)(existing)
		if err != nil {
			secret.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
		}
		return secret, err
	}
}

// GetECDSAClientCertificateCreator is a generic function to return a secret generator to create a client certificate
// signed by the cert returned by the passed getCA func. The resulting secret has no ownerRef
func GetECDSAClientCertificateCreator(name, commonName string, organizations []string, dataCertKey, dataKeyKey string, ca *resources.ECDSAKeyPair) func(existing *corev1.Secret) (*corev1.Secret, error) {
	return func(existing *corev1.Secret) (*corev1.Secret, error) {
		var se *corev1.Secret
		if existing != nil {
			se = existing
		} else {
			se = &corev1.Secret{}
		}

		se.Name = name

		if se.Data == nil {
			se.Data = map[string][]byte{}
		}

		if b, exists := se.Data[dataCertKey]; exists {
			certs, err := certutil.ParseCertsPEM(b)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret %s: %v", name, dataCertKey, err)
			}

			if resources.IsClientCertificateValidForAllOf(certs[0], commonName, organizations, ca.Cert) {
				return se, nil
			}
		}

		config := certutil.Config{
			CommonName:   commonName,
			Organization: organizations,
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		cert, key, err := GetSignedECDSACertAndKey(Duration365d, config, ca.Cert, ca.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to get a signed ECDSA cert and key: %v", err)
		}

		se.Data[dataCertKey] = cert
		se.Data[dataKeyKey] = key
		// Include the CA for simplicity
		se.Data[resources.CACertSecretKey] = certutil.EncodeCertPEM(ca.Cert)

		return se, nil
	}
}

// GetSignedECDSACertAndKey creates and returns a signed ECDSA x509 certificate and key
func GetSignedECDSACertAndKey(notAfter time.Duration, cfg certutil.Config, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) (cert []byte, key []byte, err error) {
	if len(cfg.Usages) == 0 {
		return nil, nil, errors.New("must specify at least one ExtKeyUsage")
	}

	return generateECDSACertAndKey(notAfter, false, cfg, caCert, caKey)
}

// GetECDSACACertAndKey returns a pem-encoded ECDSA certificate and key
func GetECDSACACertAndKey() (cert []byte, key []byte, err error) {
	return generateECDSACertAndKey(Duration365d*10, true, certutil.Config{CommonName: "CA"}, nil, nil)
}

// generateECDSACertAndKey generates an ECDSA x509 certificate and key
// if both caCert and caKey are non-nil it will be signed by that CA
// Most of the code is copied over from https://golang.org/src/crypto/tls/generate_cert.go
func generateECDSACertAndKey(notAfter time.Duration, isCA bool, cfg certutil.Config, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) ([]byte, []byte, error) {
	if len(cfg.CommonName) == 0 {
		return nil, nil, errors.New("must specify a CommonName")
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:    cfg.AltNames.DNSNames,
		IPAddresses: cfg.AltNames.IPs,
		NotBefore:   time.Now().UTC(),
		NotAfter:    time.Now().Add(notAfter).UTC(),

		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: cfg.Usages,
	}

	if isCA {
		template.BasicConstraintsValid = true
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	var derBytes []byte
	if caCert != nil && caKey != nil {
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, caCert, &privateKey.PublicKey, caKey)
	} else {
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate certificate: %v", err)
	}

	pemCertReader := bytes.NewBuffer([]byte{})
	if err := pem.Encode(pemCertReader, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, nil, fmt.Errorf("failed to pem-encode cert: %v", err)
	}

	privateKeyPemBlock, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshalECPrivateKey into pem.Block: %v", err)
	}

	pemKeyReader := bytes.NewBuffer([]byte{})
	if err := pem.Encode(pemKeyReader, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyPemBlock}); err != nil {
		return nil, nil, fmt.Errorf("failed to pem-encode private key: %v", err)
	}

	return pemCertReader.Bytes(), pemKeyReader.Bytes(), nil
}
