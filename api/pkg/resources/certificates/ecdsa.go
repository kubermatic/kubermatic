package certificates

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// GetECDSACertAndKey returns a pem-encoded ECDSA certificate and key
// Most of the code is copied over from https://golang.org/src/crypto/tls/generate_cert.go
func GetECDSACertAndKey(validUntil time.Time, isCA bool, hosts []string) (cert []byte, key []byte, err error) {
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
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  validUntil,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
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
	cert = pemCertReader.Bytes()
	key = pemKeyReader.Bytes()

	return cert, key, nil
}
