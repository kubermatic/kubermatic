/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package encryptionsecretsynchonizer

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testClusterName   = "test-cluster"
	testSecretName    = "encryption-key-cluster-test-cluster"
	masterNamespace   = "kubermatic"
	clusterNamespace  = "cluster-test-cluster"
	testSeedName      = "test-seed"
	testEncryptionKey = "dGVzdC1lbmNyeXB0aW9uLWtleS10aGF0LWlzLTMyLWJ5dGVz"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name             string
		masterSecret     *corev1.Secret
		existingCluster  *kubermaticv1.Cluster
		existingSecret   *corev1.Secret
		expectedSecret   *corev1.Secret
		expectError      bool
		expectSecretInNS bool
	}{
		{
			name: "scenario 1: secret with annotation, cluster exists with encryption enabled",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingCluster: generateCluster(testClusterName, true),
			expectedSecret: generateEncryptionSecret(testSecretName, clusterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			expectSecretInNS: true,
		},
		{
			name: "scenario 2: secret with annotation, cluster exists but encryption disabled",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingCluster:  generateCluster(testClusterName, false),
			expectSecretInNS: false,
		},
		{
			name: "scenario 3: secret with annotation, but cluster doesn't exist",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			expectSecretInNS: false,
		},
		{
			name: "scenario 4: secret without cluster annotation",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				"other-annotation": "other-value",
			}),
			existingCluster:  generateCluster(testClusterName, true),
			expectError:      true,
			expectSecretInNS: false,
		},
		{
			name: "scenario 5: cluster exists but not ready, should trigger requeue",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: testClusterName,
				},
				Spec: kubermaticv1.ClusterSpec{
					EncryptionConfiguration: &kubermaticv1.EncryptionConfiguration{
						Enabled: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{},
			},
			expectError:      false,
			expectSecretInNS: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Setup master client
			var masterObjects []ctrlruntimeclient.Object
			if tc.masterSecret != nil {
				masterObjects = append(masterObjects, tc.masterSecret)
			}
			masterClient := fake.NewClientBuilder().WithObjects(masterObjects...).Build()

			// Setup seed client
			var seedObjects []ctrlruntimeclient.Object
			if tc.existingCluster != nil {
				seedObjects = append(seedObjects, tc.existingCluster)
			}
			if tc.existingSecret != nil {
				seedObjects = append(seedObjects, tc.existingSecret)
			}
			seedClient := fake.NewClientBuilder().WithObjects(seedObjects...).Build()

			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{testSeedName: seedClient},
				namespace:    masterNamespace,
			}

			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testSecretName,
					Namespace: masterNamespace,
				},
			}

			// Execute reconcile
			result, err := r.Reconcile(ctx, request)

			// Check error expectation
			if tc.expectError && err == nil {
				t.Errorf("expected an error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			// For scenario 5, check that requeue is triggered (no error, but RequeueAfter is set)
			if tc.name == "scenario 5: cluster exists but not ready, should trigger requeue" {
				if err != nil {
					t.Errorf("expected no error for requeue scenario but got: %v", err)
				}
				if result.RequeueAfter == 0 {
					t.Errorf("expected RequeueAfter to be set but got: %v", result.RequeueAfter)
				}
				if result.RequeueAfter != 30*time.Second {
					t.Errorf("expected RequeueAfter to be 30 seconds but got: %v", result.RequeueAfter)
				}
			}

			// Check if secret should exist in cluster namespace
			resultSecret := &corev1.Secret{}
			err = seedClient.Get(ctx, types.NamespacedName{
				Name:      testSecretName,
				Namespace: clusterNamespace,
			}, resultSecret)

			if tc.expectSecretInNS {
				if err != nil {
					t.Errorf("expected secret to exist in cluster namespace but got error: %v", err)
					return
				}

				if tc.expectedSecret != nil {
					if resultSecret.Name != tc.expectedSecret.Name {
						t.Errorf("expected secret name to be %q, got %q", tc.expectedSecret.Name, resultSecret.Name)
					}
					if resultSecret.Namespace != tc.expectedSecret.Namespace {
						t.Errorf("expected secret namespace to be %q, got %q", tc.expectedSecret.Namespace, resultSecret.Namespace)
					}
					if !reflect.DeepEqual(resultSecret.Data, tc.expectedSecret.Data) {
						t.Errorf("expected secret data to be %q, got %q", tc.expectedSecret.Data, resultSecret.Data)
					}
					if !reflect.DeepEqual(resultSecret.Annotations, tc.expectedSecret.Annotations) {
						t.Errorf("expected secret annotations to be %q, got %q", tc.expectedSecret.Annotations, resultSecret.Annotations)
					}
				}
			} else {
				if err == nil {
					t.Errorf("expected secret not to exist in cluster namespace but it was found")
				} else if !apierrors.IsNotFound(err) {
					t.Errorf("expected NotFound error but got: %v", err)
				}
			}
		})
	}
}

func TestHandleDeletion(t *testing.T) {
	testCases := []struct {
		name                string
		secretToDelete      *corev1.Secret
		existingSecretInNS  *corev1.Secret
		expectSecretDeleted bool
	}{
		{
			name: "deletion of encryption secret should remove secret from cluster namespace",
			secretToDelete: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: masterNamespace,
					Annotations: map[string]string{
						ClusterNameAnnotation: testClusterName,
					},
				},
			},
			existingSecretInNS: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: clusterNamespace,
				},
				Data: map[string][]byte{
					"key": []byte(testEncryptionKey),
				},
			},
			expectSecretDeleted: true,
		},
		{
			name: "deletion of non-encryption secret should be ignored",
			secretToDelete: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "regular-secret",
					Namespace: masterNamespace,
				},
			},
			existingSecretInNS: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "regular-secret",
					Namespace: clusterNamespace,
				},
				Data: map[string][]byte{
					"key": []byte("some-data"),
				},
			},
			expectSecretDeleted: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Setup seed client with existing secret
			var seedObjects []ctrlruntimeclient.Object
			if tc.existingSecretInNS != nil {
				seedObjects = append(seedObjects, tc.existingSecretInNS)
			}

			// Add test cluster with proper namespace set
			if strings.HasPrefix(tc.secretToDelete.Name, EncryptionSecretPrefix) {
				cluster := &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: testClusterName,
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: clusterNamespace,
					},
				}
				seedObjects = append(seedObjects, cluster)
			}

			seedClient := fake.NewClientBuilder().WithObjects(seedObjects...).Build()

			r := &reconciler{
				log:         kubermaticlog.Logger,
				recorder:    &record.FakeRecorder{},
				seedClients: map[string]ctrlruntimeclient.Client{testSeedName: seedClient},
				namespace:   masterNamespace,
			}

			err := r.handleDeletion(ctx, kubermaticlog.Logger, tc.secretToDelete)
			if err != nil {
				t.Errorf("handleDeletion failed: %v", err)
			}

			resultSecret := &corev1.Secret{}
			err = seedClient.Get(ctx, types.NamespacedName{
				Name:      tc.secretToDelete.Name,
				Namespace: clusterNamespace,
			}, resultSecret)

			if tc.expectSecretDeleted {
				if err == nil {
					t.Errorf("expected secret to be deleted from cluster namespace but it still exists")
				} else if !apierrors.IsNotFound(err) {
					t.Errorf("expected NotFound error but got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected secret to remain in cluster namespace but got error: %v", err)
				}
			}
		})
	}
}

func TestFindTargetCluster(t *testing.T) {
	testCases := []struct {
		name            string
		clusterName     string
		clusters        []*kubermaticv1.Cluster
		expectedCluster *kubermaticv1.Cluster
		expectedSeed    string
		expectError     bool
	}{
		{
			name:        "cluster found in first seed",
			clusterName: testClusterName,
			clusters: []*kubermaticv1.Cluster{
				generateCluster(testClusterName, true),
			},
			expectedCluster: generateCluster(testClusterName, true),
			expectedSeed:    testSeedName,
		},
		{
			name:        "cluster not found in any seed",
			clusterName: "non-existent-cluster",
			clusters: []*kubermaticv1.Cluster{
				generateCluster(testClusterName, true),
			},
			expectError: true,
		},
		{
			name:        "no clusters in seeds",
			clusterName: testClusterName,
			clusters:    []*kubermaticv1.Cluster{},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var seedObjects []ctrlruntimeclient.Object
			for _, cluster := range tc.clusters {
				seedObjects = append(seedObjects, cluster)
			}
			seedClient := fake.NewClientBuilder().WithObjects(seedObjects...).Build()

			r := &reconciler{
				log:         kubermaticlog.Logger,
				seedClients: map[string]ctrlruntimeclient.Client{testSeedName: seedClient},
			}

			cluster, seedName, err := r.findTargetCluster(ctx, tc.clusterName)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("expected no error but got: %v", err)
				return
			}

			if cluster.Name != tc.expectedCluster.Name {
				t.Errorf("expected cluster name to be %q, got %q", tc.expectedCluster.Name, cluster.Name)
			}

			if seedName != tc.expectedSeed {
				t.Errorf("expected seed name to be %q, got %q", tc.expectedSeed, seedName)
			}
		})
	}
}

func generateEncryptionSecret(name, namespace string, annotations map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Data: map[string][]byte{
			"key": []byte(testEncryptionKey),
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func generateCluster(name string, encryptionEnabled bool) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Features: map[string]bool{},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-" + name,
		},
	}

	if encryptionEnabled {
		cluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest] = true
		cluster.Spec.EncryptionConfiguration = &kubermaticv1.EncryptionConfiguration{
			Enabled:   true,
			Resources: []string{"secrets"},
			Secretbox: &kubermaticv1.SecretboxEncryptionConfiguration{
				Keys: []kubermaticv1.SecretboxKey{
					{
						Name: "test-key",
						SecretRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "encryption-key-cluster-" + name,
							},
							Key: "key",
						},
					},
				},
			},
		}
	}

	return cluster
}
