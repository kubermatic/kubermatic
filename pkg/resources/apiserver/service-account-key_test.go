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

package apiserver

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

type fakeServiceAccountKeyData struct {
	cluster *kubermaticv1.Cluster
}

func (f fakeServiceAccountKeyData) Cluster() *kubermaticv1.Cluster {
	return f.cluster
}

func clusterWithServiceAccountKey(spec *kubermaticv1.KeySpec) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{}
	if spec != nil {
		cluster.Spec.KeyConfiguration = &kubermaticv1.KeyConfiguration{ServiceAccountKey: spec}
	}
	return cluster
}

func reconcileServiceAccountKey(t *testing.T, cluster *kubermaticv1.Cluster, secret *corev1.Secret) *corev1.Secret {
	t.Helper()

	name, reconciler := ServiceAccountKeyReconciler(fakeServiceAccountKeyData{cluster: cluster})()
	if name != resources.ServiceAccountKeySecretName {
		t.Fatalf("expected secret name %q, got %q", resources.ServiceAccountKeySecretName, name)
	}

	result, err := reconciler(secret)
	if err != nil {
		t.Fatalf("failed to reconcile the service account key: %v", err)
	}
	return result
}

func TestServiceAccountKeyAlgorithm(t *testing.T) {
	testCases := []struct {
		name              string
		spec              *kubermaticv1.KeySpec
		expectedBlockType string
		check             func(t *testing.T, key any)
	}{
		{
			name:              "no configuration keeps the legacy RSA-2048 key",
			spec:              nil,
			expectedBlockType: "RSA PRIVATE KEY",
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
			name:              "RSA-4096",
			spec:              &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmRSA, RSAKeySize: 4096},
			expectedBlockType: "RSA PRIVATE KEY",
			check: func(t *testing.T, key any) {
				rsaKey, ok := key.(*rsa.PrivateKey)
				if !ok {
					t.Fatalf("expected an RSA key, got %T", key)
				}
				if size := rsaKey.N.BitLen(); size != 4096 {
					t.Errorf("expected 4096 bits, got %d", size)
				}
			},
		},
		{
			name:              "ECDSA P-256",
			spec:              &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA},
			expectedBlockType: "EC PRIVATE KEY",
			check: func(t *testing.T, key any) {
				ecKey, ok := key.(*ecdsa.PrivateKey)
				if !ok {
					t.Fatalf("expected an ECDSA key, got %T", key)
				}
				if ecKey.Curve != elliptic.P256() {
					t.Errorf("expected P-256, got %s", ecKey.Curve.Params().Name)
				}
			},
		},
		{
			name:              "ECDSA P-384",
			spec:              &kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA, ECDSACurve: kubermaticv1.ECDSACurveP384},
			expectedBlockType: "EC PRIVATE KEY",
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
			secret := reconcileServiceAccountKey(t, clusterWithServiceAccountKey(test.spec), &corev1.Secret{})

			keyPEM, exists := secret.Data[resources.ServiceAccountKeySecretKey]
			if !exists {
				t.Fatalf("no %q was written", resources.ServiceAccountKeySecretKey)
			}

			block, _ := pem.Decode(keyPEM)
			if block == nil {
				t.Fatal("sa.key is not valid PEM")
			}
			if block.Type != test.expectedBlockType {
				t.Errorf("expected PEM block type %q, got %q", test.expectedBlockType, block.Type)
			}

			var (
				key any
				err error
			)
			switch block.Type {
			case "RSA PRIVATE KEY":
				key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			case "EC PRIVATE KEY":
				key, err = x509.ParseECPrivateKey(block.Bytes)
			}
			if err != nil {
				t.Fatalf("failed to parse sa.key: %v", err)
			}
			test.check(t, key)

			// sa.pub has to describe the very same key, whatever the algorithm.
			pubPEM, exists := secret.Data[resources.ServiceAccountKeyPublicKey]
			if !exists {
				t.Fatalf("no %q was written", resources.ServiceAccountKeyPublicKey)
			}
			pubBlock, _ := pem.Decode(pubPEM)
			if pubBlock == nil {
				t.Fatal("sa.pub is not valid PEM")
			}
			if pubBlock.Type != "PUBLIC KEY" {
				t.Errorf("expected PEM block type %q, got %q", "PUBLIC KEY", pubBlock.Type)
			}
			pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
			if err != nil {
				t.Fatalf("failed to parse sa.pub: %v", err)
			}
			if !publicKeysMatch(pub, key) {
				t.Error("sa.pub does not belong to sa.key")
			}
		})
	}
}

// TestServiceAccountKeyIsWrittenOnce is the guarantee that protects existing
// clusters: even when the cluster asks for a different algorithm, a secret that
// already holds a key must be handed back untouched. Rotating it would
// invalidate every service account token in the user cluster.
func TestServiceAccountKeyIsWrittenOnce(t *testing.T) {
	existing := &corev1.Secret{
		Data: map[string][]byte{
			resources.ServiceAccountKeySecretKey: []byte("-----BEGIN RSA PRIVATE KEY-----\nnot actually a key\n-----END RSA PRIVATE KEY-----\n"),
		},
	}
	original := existing.Data[resources.ServiceAccountKeySecretKey]

	cluster := clusterWithServiceAccountKey(&kubermaticv1.KeySpec{Algorithm: kubermaticv1.KeyAlgorithmECDSA})
	result := reconcileServiceAccountKey(t, cluster, existing)

	if !bytes.Equal(result.Data[resources.ServiceAccountKeySecretKey], original) {
		t.Error("an existing service account key was replaced")
	}
	if _, exists := result.Data[resources.ServiceAccountKeyPublicKey]; exists {
		t.Error("an existing secret was modified")
	}
}

// TestServiceAccountKeyRejectsUnknownAlgorithm makes sure an unusable spec fails
// loudly instead of silently generating something else.
func TestServiceAccountKeyRejectsUnknownAlgorithm(t *testing.T) {
	cluster := clusterWithServiceAccountKey(&kubermaticv1.KeySpec{Algorithm: "Ed25519"})

	_, reconciler := ServiceAccountKeyReconciler(fakeServiceAccountKeyData{cluster: cluster})()
	if _, err := reconciler(&corev1.Secret{}); err == nil {
		t.Error("expected an error for an unknown algorithm, but got none")
	}
}

func publicKeysMatch(pub any, key any) bool {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		other, ok := pub.(*rsa.PublicKey)
		return ok && k.PublicKey.Equal(other)
	case *ecdsa.PrivateKey:
		other, ok := pub.(*ecdsa.PublicKey)
		return ok && k.PublicKey.Equal(other)
	}
	return false
}
