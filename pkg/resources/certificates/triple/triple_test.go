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
	config            KeyConfig
	expectedBlockType string
	check             func(t *testing.T, key any)
}{
	"legacy default": {
		config:            KeyConfig{},
		expectedBlockType: RSAPrivateKeyBlockType,
		check:             expectRSA(2048),
	},
	"RSA-2048": {
		config:            KeyConfig{RSAKeySize: 2048},
		expectedBlockType: RSAPrivateKeyBlockType,
		check:             expectRSA(2048),
	},
	"RSA-3072": {
		config:            KeyConfig{RSAKeySize: 3072},
		expectedBlockType: RSAPrivateKeyBlockType,
		check:             expectRSA(3072),
	},
	"RSA-4096": {
		config:            KeyConfig{RSAKeySize: 4096},
		expectedBlockType: RSAPrivateKeyBlockType,
		check:             expectRSA(4096),
	},
	"ECDSA P-256": {
		config:            KeyConfig{ECDSACurve: elliptic.P256()},
		expectedBlockType: ECPrivateKeyBlockType,
		check:             expectECDSA(elliptic.P256()),
	},
	"ECDSA P-384": {
		config:            KeyConfig{ECDSACurve: elliptic.P384()},
		expectedBlockType: ECPrivateKeyBlockType,
		check:             expectECDSA(elliptic.P384()),
	},
}

func expectRSA(bits int) func(t *testing.T, key any) {
	return func(t *testing.T, key any) {
		t.Helper()
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			t.Fatalf("expected an RSA key, got %T", key)
		}
		if size := rsaKey.N.BitLen(); size != bits {
			t.Errorf("expected %d bits, got %d", bits, size)
		}
	}
}

func expectECDSA(curve elliptic.Curve) func(t *testing.T, key any) {
	return func(t *testing.T, key any) {
		t.Helper()
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			t.Fatalf("expected an ECDSA key, got %T", key)
		}
		if ecKey.Curve != curve {
			t.Errorf("expected %s, got %s", curve.Params().Name, ecKey.Curve.Params().Name)
		}
	}
}

func TestGenerateKey(t *testing.T) {
	for name, test := range keyConfigs {
		t.Run(name, func(t *testing.T) {
			key, err := test.config.GenerateKey()
			if err != nil {
				t.Fatalf("failed to generate key: %v", err)
			}
			test.check(t, key)
		})
	}
}

func TestMarshalPrivateKeyPEMRoundTrip(t *testing.T) {
	for name, test := range keyConfigs {
		t.Run(name, func(t *testing.T) {
			key, err := test.config.GenerateKey()
			if err != nil {
				t.Fatalf("failed to generate key: %v", err)
			}

			keyPEM, err := MarshalPrivateKeyPEM(key)
			if err != nil {
				t.Fatalf("failed to encode key: %v", err)
			}

			block, _ := pem.Decode(keyPEM)
			if block == nil {
				t.Fatal("the encoded key is not valid PEM")
			}
			if block.Type != test.expectedBlockType {
				t.Errorf("expected PEM block type %q, got %q", test.expectedBlockType, block.Type)
			}

			parsed, err := ParsePrivateKeyPEM(keyPEM)
			if err != nil {
				t.Fatalf("failed to parse the encoded key: %v", err)
			}
			test.check(t, parsed)
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

func TestKeyPairsVerifyAgainstTheirCA(t *testing.T) {
	for name, test := range keyConfigs {
		t.Run(name, func(t *testing.T) {
			ca, err := NewCAWithConfig("test-ca", test.config)
			if err != nil {
				t.Fatalf("failed to create CA: %v", err)
			}
			test.check(t, ca.Key)

			server, err := NewServerKeyPairWithConfig(ca, "test-server", "svc", "ns", "cluster.local", []string{"1.2.3.4"}, []string{"example.com"}, test.config)
			if err != nil {
				t.Fatalf("failed to create server key pair: %v", err)
			}
			test.check(t, server.Key)

			client, err := NewClientKeyPairWithConfig(ca, "test-client", []string{"test-org"}, test.config)
			if err != nil {
				t.Fatalf("failed to create client key pair: %v", err)
			}
			test.check(t, client.Key)

			pool := x509.NewCertPool()
			pool.AddCert(ca.Cert)

			for leafName, leaf := range map[string]*KeyPair{"server": server, "client": client} {
				if _, err := leaf.Cert.Verify(x509.VerifyOptions{
					Roots:     pool,
					KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
				}); err != nil {
					t.Errorf("the %s certificate does not verify against its CA: %v", leafName, err)
				}
			}
		})
	}
}

// TestKeyEnciphermentIsOnlySetForRSA guards the one certificate property that
// depends on the algorithm: keyEncipherment is meaningless for ECDSA keys.
func TestKeyEnciphermentIsOnlySetForRSA(t *testing.T) {
	for name, test := range keyConfigs {
		t.Run(name, func(t *testing.T) {
			ca, err := NewCAWithConfig("test-ca", test.config)
			if err != nil {
				t.Fatalf("failed to create CA: %v", err)
			}

			client, err := NewClientKeyPairWithConfig(ca, "test-client", nil, test.config)
			if err != nil {
				t.Fatalf("failed to create client key pair: %v", err)
			}

			_, isRSA := client.Key.(*rsa.PrivateKey)
			hasKeyEncipherment := client.Cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0

			if isRSA && !hasKeyEncipherment {
				t.Error("an RSA certificate lost its keyEncipherment usage")
			}
			if !isRSA && hasKeyEncipherment {
				t.Error("an ECDSA certificate must not carry the keyEncipherment usage")
			}
			if client.Cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
				t.Error("the certificate lost its digitalSignature usage")
			}
		})
	}
}

func TestParseKeyPair(t *testing.T) {
	for name, test := range keyConfigs {
		t.Run(name, func(t *testing.T) {
			ca, err := NewCAWithConfig("test-ca", test.config)
			if err != nil {
				t.Fatalf("failed to create CA: %v", err)
			}

			keyPEM, err := MarshalPrivateKeyPEM(ca.Key)
			if err != nil {
				t.Fatalf("failed to encode key: %v", err)
			}
			certPEM := EncodeCertPEM(ca.Cert)

			parsed, err := ParseKeyPair(certPEM, keyPEM)
			if err != nil {
				t.Fatalf("failed to parse the key pair: %v", err)
			}
			test.check(t, parsed.Key)

			// ParseRSAKeyPair still rejects everything that is not RSA.
			_, err = ParseRSAKeyPair(certPEM, keyPEM)
			if _, isRSA := ca.Key.(*rsa.PrivateKey); isRSA {
				if err != nil {
					t.Errorf("expected an RSA key pair to parse, but got: %v", err)
				}
			} else if err == nil {
				t.Error("expected ParseRSAKeyPair to reject a non-RSA key pair")
			}
		})
	}
}
