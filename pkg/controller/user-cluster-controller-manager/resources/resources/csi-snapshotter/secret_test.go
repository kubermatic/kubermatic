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

package csisnapshotter

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"

	corev1 "k8s.io/api/core/v1"
)

// TestTLSServingCertificateFollowsTheKeyConfig asserts that the webhook serving
// certificate is generated with the algorithm the cluster was created with,
// and that no configuration at all still yields the legacy RSA-2048 key.
func TestTLSServingCertificateFollowsTheKeyConfig(t *testing.T) {
	testCases := map[string]struct {
		config            triple.KeyConfig
		expectedBlockType string
		check             func(t *testing.T, key any)
	}{
		"no configuration keeps RSA-2048": {
			config:            triple.KeyConfig{},
			expectedBlockType: triple.RSAPrivateKeyBlockType,
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
		"ECDSA P-384": {
			config:            triple.KeyConfig{ECDSACurve: elliptic.P384()},
			expectedBlockType: triple.ECPrivateKeyBlockType,
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

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			ca, err := triple.NewCAWithConfig("test-ca", test.config)
			if err != nil {
				t.Fatalf("failed to create CA: %v", err)
			}

			_, reconciler := TLSServingCertificateReconciler(
				resources.CSISnapshotValidationWebhookName,
				ca,
				func() (triple.KeyConfig, error) { return test.config, nil },
			)()

			secret, err := reconciler(&corev1.Secret{})
			if err != nil {
				t.Fatalf("failed to reconcile the serving certificate: %v", err)
			}

			block, _ := pem.Decode(secret.Data[resources.CSIWebhookServingCertKeyKeyName])
			if block == nil {
				t.Fatal("the serving cert key is not valid PEM")
			}
			if block.Type != test.expectedBlockType {
				t.Errorf("expected PEM block type %q, got %q", test.expectedBlockType, block.Type)
			}

			key, err := triple.ParsePrivateKeyPEM(secret.Data[resources.CSIWebhookServingCertKeyKeyName])
			if err != nil {
				t.Fatalf("failed to parse the serving cert key: %v", err)
			}
			test.check(t, key)

			certs, err := triple.ParseCertsPEM(secret.Data[resources.CSIWebhookServingCertCertKeyName])
			if err != nil {
				t.Fatalf("failed to parse the serving certificate: %v", err)
			}

			pool := x509.NewCertPool()
			pool.AddCert(ca.Cert)
			if _, err := certs[0].Verify(x509.VerifyOptions{
				Roots:     pool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
			}); err != nil {
				t.Errorf("the serving certificate does not verify against its CA: %v", err)
			}
		})
	}
}

// TestTLSServingCertificateReportsAnUnusableConfig makes sure a broken key
// configuration fails the reconcile instead of silently producing something
// else.
func TestTLSServingCertificateReportsAnUnusableConfig(t *testing.T) {
	ca, err := triple.NewCA("test-ca")
	if err != nil {
		t.Fatalf("failed to create CA: %v", err)
	}

	_, reconciler := TLSServingCertificateReconciler(
		resources.CSISnapshotValidationWebhookName,
		ca,
		func() (triple.KeyConfig, error) { return triple.KeyConfig{}, errors.New("unusable key configuration") },
	)()

	if _, err := reconciler(&corev1.Secret{}); err == nil {
		t.Error("expected the reconcile to fail, but it succeeded")
	}
}
