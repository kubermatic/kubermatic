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

package resources

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"testing"

	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestGetClusterRootCAAcceptsEveryAlgorithm covers the one function that used to
// require an RSA CA. It feeds every leaf reconciler and every internal
// kubeconfig, so rejecting a non-RSA CA here would fail a cluster's entire
// control plane reconcile at once.
func TestGetClusterRootCAAcceptsEveryAlgorithm(t *testing.T) {
	const namespace = "cluster-test"

	testCases := map[string]struct {
		config triple.KeyConfig
		check  func(t *testing.T, key any)
	}{
		"an RSA CA": {
			config: triple.KeyConfig{},
			check: func(t *testing.T, key any) {
				if _, ok := key.(*rsa.PrivateKey); !ok {
					t.Errorf("expected an RSA key, got %T", key)
				}
			},
		},
		"an ECDSA CA": {
			config: triple.KeyConfig{ECDSACurve: elliptic.P384()},
			check: func(t *testing.T, key any) {
				if _, ok := key.(*ecdsa.PrivateKey); !ok {
					t.Errorf("expected an ECDSA key, got %T", key)
				}
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			ca, err := triple.NewCAWithConfig("root-ca.test", test.config)
			if err != nil {
				t.Fatalf("failed to create CA: %v", err)
			}

			keyPEM, err := triple.MarshalPrivateKeyPEM(ca.Key)
			if err != nil {
				t.Fatalf("failed to encode the CA key: %v", err)
			}

			client := fake.NewClientBuilder().WithObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: CASecretName, Namespace: namespace},
				Data: map[string][]byte{
					CACertSecretKey: triple.EncodeCertPEM(ca.Cert),
					CAKeySecretKey:  keyPEM,
				},
			}).Build()

			keyPair, err := GetClusterRootCA(context.Background(), namespace, client)
			if err != nil {
				t.Fatalf("failed to read the cluster CA: %v", err)
			}

			test.check(t, keyPair.Key)

			if !keyPair.Cert.Equal(ca.Cert) {
				t.Error("a different certificate was returned")
			}

			// The CA has to remain usable for signing, which is the only thing
			// the callers actually need from it.
			if _, err := triple.NewClientKeyPairWithConfig(keyPair, "test-client", nil, test.config); err != nil {
				t.Errorf("the CA read back cannot sign: %v", err)
			}
		})
	}
}
