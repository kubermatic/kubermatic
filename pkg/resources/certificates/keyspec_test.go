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

package certificates

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
)

func TestResolveKeySpec(t *testing.T) {
	testCases := []struct {
		name        string
		spec        *kubermaticv1.KeySpec
		expectedRSA int
		expectedEC  elliptic.Curve
		expectedErr bool
	}{
		{
			name:        "a nil spec is the legacy default",
			spec:        nil,
			expectedRSA: 2048,
		},
		{
			name:        "an empty spec is the legacy default",
			spec:        &kubermaticv1.KeySpec{},
			expectedRSA: 2048,
		},
		{
			name:        "RSA without a size is the legacy default",
			spec:        &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmRSA},
			expectedRSA: 2048,
		},
		{
			name:        "RSA with an explicit size",
			spec:        &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmRSA, RSAKeySize: 4096},
			expectedRSA: 4096,
		},
		{
			name:        "a size without an algorithm still means RSA",
			spec:        &kubermaticv1.KeySpec{RSAKeySize: 3072},
			expectedRSA: 3072,
		},
		{
			name:       "ECDSA without a curve defaults to P-256",
			spec:       &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA},
			expectedEC: elliptic.P256(),
		},
		{
			name:       "ECDSA with an explicit curve",
			spec:       &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA, ECDSACurve: kubermaticv1.ECDSACurveP384},
			expectedEC: elliptic.P384(),
		},
		{
			name:        "an unknown algorithm is an error, never a silent fallback",
			spec:        &kubermaticv1.KeySpec{Algorithm: "Ed25519"},
			expectedErr: true,
		},
		{
			name:        "an unknown curve is an error, never a silent fallback",
			spec:        &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA, ECDSACurve: "P521"},
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config, err := ResolveKeySpec(test.spec)
			if test.expectedErr {
				if err == nil {
					t.Fatalf("expected an error, but got config %v", config)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			if config.ECDSACurve != test.expectedEC {
				t.Errorf("expected curve %v, got %v", test.expectedEC, config.ECDSACurve)
			}

			// A zero RSAKeySize is only acceptable for the legacy default, where the
			// generator falls back to 2048 itself.
			if test.expectedEC == nil {
				key, err := config.GenerateKey()
				if err != nil {
					t.Fatalf("failed to generate key: %v", err)
				}
				rsaKey, ok := key.(*rsa.PrivateKey)
				if !ok {
					t.Fatalf("expected an RSA key, got %T", key)
				}
				if size := rsaKey.N.BitLen(); size != test.expectedRSA {
					t.Errorf("expected a %d bit key, got %d bits", test.expectedRSA, size)
				}
			}
		})
	}
}

func TestGenerateKeyProducesTheConfiguredAlgorithm(t *testing.T) {
	testCases := []struct {
		name  string
		spec  *kubermaticv1.KeySpec
		check func(t *testing.T, key any)
	}{
		{
			name: "nil spec generates RSA-2048",
			spec: nil,
			check: func(t *testing.T, key any) {
				rsaKey, ok := key.(*rsa.PrivateKey)
				if !ok {
					t.Fatalf("expected an RSA key, got %T", key)
				}
				if size := rsaKey.N.BitLen(); size != 2048 {
					t.Errorf("expected 2048 bits, got %d", size)
				}
			},
		},
		{
			name: "ECDSA P-384",
			spec: &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA, ECDSACurve: kubermaticv1.ECDSACurveP384},
			check: func(t *testing.T, key any) {
				ecKey, ok := key.(*ecdsa.PrivateKey)
				if !ok {
					t.Fatalf("expected an ECDSA key, got %T", key)
				}
				if ecKey.Curve != elliptic.P384() {
					t.Errorf("expected P-384, got %s", ecKey.Curve.Params().Name)
				}
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config, err := ResolveKeySpec(test.spec)
			if err != nil {
				t.Fatalf("failed to resolve key spec: %v", err)
			}
			key, err := config.GenerateKey()
			if err != nil {
				t.Fatalf("failed to generate key: %v", err)
			}
			test.check(t, key)
		})
	}
}

func TestKeyConfigAccessorsHandleLegacyClusters(t *testing.T) {
	// A cluster created before the KeyConfiguration field existed must resolve to
	// RSA-2048 for both key families, without any defaulting webhook involved.
	cluster := &kubermaticv1.Cluster{}

	for name, resolve := range map[string]func(*kubermaticv1.Cluster) (triple.KeyConfig, error){
		"service account key": ServiceAccountKeyConfig,
		"certificates":        CertificateKeyConfig,
	} {
		t.Run(name, func(t *testing.T) {
			config, err := resolve(cluster)
			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}
			if config.ECDSACurve != nil {
				t.Errorf("expected an RSA configuration, got %s", config)
			}

			key, err := config.GenerateKey()
			if err != nil {
				t.Fatalf("failed to generate key: %v", err)
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				t.Fatalf("expected an RSA key, got %T", key)
			}
			if size := rsaKey.N.BitLen(); size != 2048 {
				t.Errorf("expected 2048 bits, got %d", size)
			}
		})
	}
}
