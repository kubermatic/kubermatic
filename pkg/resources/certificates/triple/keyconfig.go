/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package triple

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// KeyConfig describes the algorithm and size of a private key to generate.
//
// The zero value describes the legacy default, RSA-2048, so that any code path
// which does not explicitly configure a key keeps producing exactly what KKP
// produced before this type existed.
type KeyConfig struct {
	// ECDSACurve selects ECDSA with the given curve. When nil, an RSA key is
	// generated instead.
	ECDSACurve elliptic.Curve

	// RSAKeySize is the modulus size in bits for RSA keys. Ignored when
	// ECDSACurve is set; zero means the legacy default of 2048.
	RSAKeySize int
}

// KeyConfigGetter resolves the key parameters to use for a piece of generated
// key material. It is resolved while reconciling rather than when reconcilers
// are assembled, so that an unusable configuration surfaces as a reconcile
// error instead of being swallowed.
type KeyConfigGetter = func() (KeyConfig, error)

// LegacyKeyConfigGetter is the KeyConfigGetter for key material that does not
// belong to a single user cluster and therefore stays RSA-2048.
func LegacyKeyConfigGetter() (KeyConfig, error) {
	return KeyConfig{}, nil
}

// GenerateKey returns a new private key according to the configuration.
func (c KeyConfig) GenerateKey() (crypto.Signer, error) {
	if c.ECDSACurve != nil {
		key, err := ecdsa.GenerateKey(c.ECDSACurve, rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
		}
		return key, nil
	}

	size := c.RSAKeySize
	if size == 0 {
		size = rsaKeySize
	}

	key, err := rsa.GenerateKey(rand.Reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	return key, nil
}

// String returns a human readable description of the key parameters, for use
// in log messages and errors.
func (c KeyConfig) String() string {
	if c.ECDSACurve != nil {
		return fmt.Sprintf("ECDSA-%s", c.ECDSACurve.Params().Name)
	}
	size := c.RSAKeySize
	if size == 0 {
		size = rsaKeySize
	}
	return fmt.Sprintf("RSA-%d", size)
}

// MarshalPrivateKeyPEM returns PEM-encoded private key data for any supported
// key type. RSA keys are encoded as PKCS#1 in an "RSA PRIVATE KEY" block, which
// is byte-identical to what EncodePrivateKeyPEM has always produced; ECDSA keys
// are encoded as SEC 1 in an "EC PRIVATE KEY" block.
func MarshalPrivateKeyPEM(key crypto.Signer) ([]byte, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return pem.EncodeToMemory(&pem.Block{
			Type:  RSAPrivateKeyBlockType,
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}), nil

	case *ecdsa.PrivateKey:
		der, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ECDSA key: %w", err)
		}
		return pem.EncodeToMemory(&pem.Block{
			Type:  ECPrivateKeyBlockType,
			Bytes: der,
		}), nil

	default:
		return nil, fmt.Errorf("unsupported private key type %T", key)
	}
}

// MarshalPublicKeyPEM returns the PKIX PEM encoding of a private key's public
// half. This works for every algorithm supported by KeyConfig.
func MarshalPublicKeyPEM(key crypto.Signer) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  PublicKeyBlockType,
		Bytes: der,
	}), nil
}
