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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

var keyConfigs = map[string]struct {
	config    KeyConfig
	blockType string
}{
	"legacy default": {KeyConfig{}, RSAPrivateKeyBlockType},
	"RSA-4096":       {KeyConfig{RSAKeySize: 4096}, RSAPrivateKeyBlockType},
	"ECDSA P-384":    {KeyConfig{ECDSACurve: elliptic.P384()}, ECPrivateKeyBlockType},
}

func checkKey(t *testing.T, config KeyConfig, key any) {
	t.Helper()

	if config.ECDSACurve != nil {
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			t.Fatalf("expected an ECDSA key, got %T", key)
		}
		if ecKey.Curve != config.ECDSACurve {
			t.Errorf("expected %s, got %s", config.ECDSACurve.Params().Name, ecKey.Curve.Params().Name)
		}
		return
	}

	expectedBits := config.RSAKeySize
	if expectedBits == 0 {
		expectedBits = 2048
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("expected an RSA key, got %T", key)
	}
	if size := rsaKey.N.BitLen(); size != expectedBits {
		t.Errorf("expected %d bits, got %d", expectedBits, size)
	}
}

func TestGenerateKeyPEMRoundTrip(t *testing.T) {
	for name, test := range keyConfigs {
		t.Run(name, func(t *testing.T) {
			key, err := test.config.GenerateKey()
			if err != nil {
				t.Fatalf("failed to generate key: %v", err)
			}
			checkKey(t, test.config, key)

			keyPEM, err := MarshalPrivateKeyPEM(key)
			if err != nil {
				t.Fatalf("failed to encode key: %v", err)
			}

			block, _ := pem.Decode(keyPEM)
			if block == nil {
				t.Fatal("the encoded key is not valid PEM")
			}
			if block.Type != test.blockType {
				t.Errorf("expected PEM block type %q, got %q", test.blockType, block.Type)
			}

			parsed, err := ParsePrivateKeyPEM(keyPEM)
			if err != nil {
				t.Fatalf("failed to parse the encoded key: %v", err)
			}
			checkKey(t, test.config, parsed)
		})
	}
}

// TestRSAEncodingIsUnchanged is the cheapest possible proof that the default
// path did not move: an RSA key still encodes exactly as it did before
// MarshalPrivateKeyPEM replaced the RSA-only encoder.
func TestRSAEncodingIsUnchanged(t *testing.T) {
	key, err := KeyConfig{}.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("the legacy default is not an RSA key but %T", key)
	}

	legacy := pem.EncodeToMemory(&pem.Block{
		Type:  RSAPrivateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	})

	current, err := MarshalPrivateKeyPEM(key)
	if err != nil {
		t.Fatalf("failed to encode key: %v", err)
	}

	if !bytes.Equal(legacy, current) {
		t.Error("the RSA encoding changed; existing clusters would see their key material rewritten")
	}
}

// TestLeafCertificates covers what the reconcilers depend on: a leaf uses the
// configured algorithm, verifies against its CA, and only carries the
// keyEncipherment usage when the key is RSA, for which it is meaningful.
func TestLeafCertificates(t *testing.T) {
	for name, test := range keyConfigs {
		t.Run(name, func(t *testing.T) {
			ca, err := NewCAWithConfig("test-ca", test.config)
			if err != nil {
				t.Fatalf("failed to create CA: %v", err)
			}
			checkKey(t, test.config, ca.Key)

			client, err := NewClientKeyPairWithConfig(ca, "test-client", []string{"test-org"}, test.config)
			if err != nil {
				t.Fatalf("failed to create client key pair: %v", err)
			}
			checkKey(t, test.config, client.Key)

			pool := x509.NewCertPool()
			pool.AddCert(ca.Cert)
			if _, err := client.Cert.Verify(x509.VerifyOptions{
				Roots:     pool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
			}); err != nil {
				t.Errorf("the client certificate does not verify against its CA: %v", err)
			}

			_, isRSA := client.Key.(*rsa.PrivateKey)
			hasKeyEncipherment := client.Cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0
			if isRSA != hasKeyEncipherment {
				t.Errorf("keyEncipherment is %v for an RSA=%v key", hasKeyEncipherment, isRSA)
			}
			if client.Cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
				t.Error("the certificate lost its digitalSignature usage")
			}
		})
	}
}
