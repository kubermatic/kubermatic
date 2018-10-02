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

	certutil "k8s.io/client-go/util/cert"
)

const Duration365d = time.Hour * 24 * 365

func GetSignedECDSACertAndKey(notAfter time.Duration, cfg certutil.Config, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) (cert []byte, key []byte, err error) {
	if len(cfg.CommonName) == 0 {
		return nil, nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, nil, errors.New("must specify at least one ExtKeyUsage")
	}

	return generateECDSACertAndKey(notAfter, false, cfg, caCert, caKey)
}

// GetECDSACertAndKey returns a pem-encoded ECDSA certificate and key
func GetECDSACACertAndKey() (cert []byte, key []byte, err error) {
	return generateECDSACertAndKey(Duration365d*10, true, certutil.Config{}, nil, nil)
}

// generateECDSACertAndKey generates an ECDSA x509 certificate and key
// if both caCert and caKey are non-nil it will be signed by that CA
// Most of the code is copied over from https://golang.org/src/crypto/tls/generate_cert.go
func generateECDSACertAndKey(notAfter time.Duration, isCA bool, cfg certutil.Config, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) ([]byte, []byte, error) {
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
