/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package triple generates key-certificate pairs for the
// triple (CA, Server, Client).
package triple

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"time"

	certutil "k8s.io/client-go/util/cert"
)

const (
	rsaKeySize             = 2048
	duration365d           = time.Hour * 24 * 365
	certificateBlockType   = "CERTIFICATE"
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
	// ECPrivateKeyBlockType is a possible value for pem.Block.Type.
	ECPrivateKeyBlockType = "EC PRIVATE KEY"
	PrivateKeyBlockType   = "PRIVATE KEY"
	CertificateBlockType  = "CERTIFICATE"
	// PublicKeyBlockType is a possible value for pem.Block.Type.
	PublicKeyBlockType = "PUBLIC KEY"
)

// KeyPair is a certificate and the private key it was issued for. The key is
// kept as a crypto.Signer so that RSA and ECDSA material can be handled without
// distinguishing between them anywhere but at generation and encoding time.
type KeyPair struct {
	Key  crypto.Signer
	Cert *x509.Certificate
}

// NewCA creates a CA with the legacy RSA-2048 key.
func NewCA(name string) (*KeyPair, error) {
	return NewCAWithConfig(name, KeyConfig{})
}

// NewCAWithConfig creates a CA whose key follows the given configuration.
func NewCAWithConfig(name string, keyConfig KeyConfig) (*KeyPair, error) {
	key, err := keyConfig.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to create a private key for a new CA: %w", err)
	}

	config := certutil.Config{
		CommonName: name,
	}

	cert, err := certutil.NewSelfSignedCACert(config, key)
	if err != nil {
		return nil, fmt.Errorf("unable to create a self-signed certificate for a new CA: %w", err)
	}

	return &KeyPair{
		Key:  key,
		Cert: cert,
	}, nil
}

// NewServerKeyPair creates a serving certificate with the legacy RSA-2048 key.
func NewServerKeyPair(ca *KeyPair, commonName, svcName, svcNamespace, dnsDomain string, ips, hostnames []string) (*KeyPair, error) {
	return NewServerKeyPairWithConfig(ca, commonName, svcName, svcNamespace, dnsDomain, ips, hostnames, KeyConfig{})
}

// NewServerKeyPairWithConfig creates a serving certificate whose key follows the
// given configuration.
func NewServerKeyPairWithConfig(ca *KeyPair, commonName, svcName, svcNamespace, dnsDomain string, ips, hostnames []string, keyConfig KeyConfig) (*KeyPair, error) {
	key, err := keyConfig.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to create a server private key: %w", err)
	}

	namespacedName := fmt.Sprintf("%s.%s", svcName, svcNamespace)
	internalAPIServerFQDN := []string{
		svcName,
		namespacedName,
		fmt.Sprintf("%s.svc", namespacedName),
		fmt.Sprintf("%s.svc.%s", namespacedName, dnsDomain),
	}

	altNames := certutil.AltNames{}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			altNames.IPs = append(altNames.IPs, ip)
		}
	}
	altNames.DNSNames = append(altNames.DNSNames, hostnames...)
	altNames.DNSNames = append(altNames.DNSNames, internalAPIServerFQDN...)

	config := certutil.Config{
		CommonName: commonName,
		AltNames:   altNames,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	cert, err := newSignedCert(config, key, ca.Cert, ca.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to sign the server certificate: %w", err)
	}

	return &KeyPair{
		Key:  key,
		Cert: cert,
	}, nil
}

// NewClientKeyPair creates a client certificate with the legacy RSA-2048 key.
func NewClientKeyPair(ca *KeyPair, commonName string, organizations []string) (*KeyPair, error) {
	return NewClientKeyPairWithConfig(ca, commonName, organizations, KeyConfig{})
}

// NewClientKeyPairWithConfig creates a client certificate whose key follows the
// given configuration.
func NewClientKeyPairWithConfig(ca *KeyPair, commonName string, organizations []string, keyConfig KeyConfig) (*KeyPair, error) {
	key, err := keyConfig.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to create a client private key: %w", err)
	}

	config := certutil.Config{
		CommonName:   commonName,
		Organization: organizations,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	cert, err := newSignedCert(config, key, ca.Cert, ca.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to sign the client certificate: %w", err)
	}

	return &KeyPair{
		Key:  key,
		Cert: cert,
	}, nil
}

// newSignedCert creates a signed certificate using the given CA certificate and key.
func newSignedCert(cfg certutil.Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d).UTC(),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}

	// keyEncipherment only applies to keys that can be used for encryption, which
	// rules out ECDSA. RSA certificates keep the usage they have always had.
	if _, isRSA := key.Public().(*rsa.PublicKey); isRSA {
		certTmpl.KeyUsage |= x509.KeyUsageKeyEncipherment
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// ParseKeyPair parses a PEM-encoded certificate and private key of any supported
// algorithm.
func ParseKeyPair(certPEM, keyPEM []byte) (*KeyPair, error) {
	certs, err := certutil.ParseCertsPEM(certPEM)
	if err != nil {
		return nil, fmt.Errorf("certificate is not valid PEM: %w", err)
	}

	if len(certs) != 1 {
		return nil, fmt.Errorf("did not find exactly one but %v certificates", len(certs))
	}

	key, err := ParsePrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("private key is not valid PEM: %w", err)
	}

	signer, isSigner := key.(crypto.Signer)
	if !isSigner {
		return nil, fmt.Errorf("private key of type %T cannot be used to sign", key)
	}

	return &KeyPair{Cert: certs[0], Key: signer}, nil
}

// ParseRSAKeyPair is ParseKeyPair for callers that cannot handle anything but an
// RSA key.
func ParseRSAKeyPair(certPEM, keyPEM []byte) (*KeyPair, error) {
	keyPair, err := ParseKeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	if _, isRSAKey := keyPair.Key.(*rsa.PrivateKey); !isRSAKey {
		return nil, errors.New("private key is not a RSA key")
	}

	return keyPair, nil
}

// EncodeCertPEM returns PEM-encoded certificate data.
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// ParsePrivateKeyPEM returns a private key parsed from a PEM block in the supplied data.
// Recognizes PEM blocks for "EC PRIVATE KEY", "RSA PRIVATE KEY", or "PRIVATE KEY".
func ParsePrivateKeyPEM(keyData []byte) (interface{}, error) {
	var privateKeyPemBlock *pem.Block
	for {
		privateKeyPemBlock, keyData = pem.Decode(keyData)
		if privateKeyPemBlock == nil {
			break
		}

		switch privateKeyPemBlock.Type {
		case ECPrivateKeyBlockType:
			// ECDSA Private Key in ASN.1 format
			if key, err := x509.ParseECPrivateKey(privateKeyPemBlock.Bytes); err == nil {
				return key, nil
			}
		case RSAPrivateKeyBlockType:
			// RSA Private Key in PKCS#1 format
			if key, err := x509.ParsePKCS1PrivateKey(privateKeyPemBlock.Bytes); err == nil {
				return key, nil
			}
		case PrivateKeyBlockType:
			// RSA or ECDSA Private Key in unencrypted PKCS#8 format
			if key, err := x509.ParsePKCS8PrivateKey(privateKeyPemBlock.Bytes); err == nil {
				return key, nil
			}
		}

		// tolerate non-key PEM blocks for compatibility with things like "EC PARAMETERS" blocks
		// originally, only the first PEM block was parsed and expected to be a key block
	}

	// we read all the PEM blocks and didn't recognize one
	return nil, fmt.Errorf("data does not contain a valid RSA or ECDSA private key")
}

// ParseCertsPEM returns the x509.Certificates contained in the given PEM-encoded byte array
// Returns an error if a certificate could not be parsed, or if the data does not contain any certificates.
func ParseCertsPEM(pemCerts []byte) ([]*x509.Certificate, error) {
	ok := false
	certs := []*x509.Certificate{}
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		// Only use PEM "CERTIFICATE" blocks without extra headers
		if block.Type != CertificateBlockType || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return certs, err
		}

		certs = append(certs, cert)
		ok = true
	}

	if !ok {
		return certs, errors.New("data does not contain any valid RSA or ECDSA certificates")
	}
	return certs, nil
}

// NewPrivateKey creates an RSA private key.
func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// NewSignedCert creates a signed certificate using the given CA certificate and key.
func NewSignedCert(cfg certutil.Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d).UTC(),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}

	// keyEncipherment only applies to keys that can be used for encryption, which
	// rules out ECDSA. RSA certificates keep the usage they have always had.
	if _, isRSA := key.Public().(*rsa.PublicKey); isRSA {
		certTmpl.KeyUsage |= x509.KeyUsageKeyEncipherment
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}
