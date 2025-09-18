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
			name: "scenario 1: cluster exists with encryption enabled and secret exists",
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
			name: "scenario 2: cluster exists but encryption disabled",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingCluster:  generateCluster(testClusterName, false),
			expectSecretInNS: false,
		},
		{
			name:             "scenario 3: cluster exists with encryption enabled but secret doesn't exist",
			existingCluster:  generateCluster(testClusterName, true),
			expectSecretInNS: false,
		},
		{
			name: "scenario 4: cluster exists but not ready (no namespace)",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: testClusterName,
					Labels: map[string]string{
						kubermaticv1.WorkerNameLabelKey: "test-worker",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					EncryptionConfiguration: &kubermaticv1.EncryptionConfiguration{
						Enabled: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{}, // No NamespaceName set
			},
			expectError:      false,
			expectSecretInNS: false,
		},
		{
			name: "scenario 5: cluster exists with wrong worker name",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: testClusterName,
					Labels: map[string]string{
						kubermaticv1.WorkerNameLabelKey: "wrong-worker",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					EncryptionConfiguration: &kubermaticv1.EncryptionConfiguration{
						Enabled: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: clusterNamespace,
				},
			},
			expectError:      false,
			expectSecretInNS: false,
		},
		{
			name: "scenario 6: cluster exists but encryption disabled - should cleanup secrets",
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingCluster:  generateCluster(testClusterName, false), // encryption disabled
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
				workerName:   "test-worker",
			}

			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testClusterName,
					Namespace: testSeedName,
				},
			}

			// Execute reconcile
			_, err := r.Reconcile(ctx, request)

			// Check error expectation
			if tc.expectError && err == nil {
				t.Errorf("expected an error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			// Check if secret should exist in cluster namespace
			if tc.existingCluster != nil && tc.existingCluster.Status.NamespaceName != "" {
				resultSecret := &corev1.Secret{}
				err = seedClient.Get(ctx, types.NamespacedName{
					Name:      testSecretName,
					Namespace: tc.existingCluster.Status.NamespaceName,
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
			}
		})
	}
}

func TestClusterDeletion(t *testing.T) {
	testCases := []struct {
		name                      string
		clusterToDelete           *kubermaticv1.Cluster
		masterSecret              *corev1.Secret
		existingSecretInUC        *corev1.Secret
		expectMasterSecretDeleted bool
		expectUCSecretDeleted     bool
	}{
		{
			name: "cluster deletion should remove secrets from both kubermatic and UC namespace",
			clusterToDelete: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              testClusterName,
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{EncryptionSecretCleanupFinalizer},
					Labels: map[string]string{
						kubermaticv1.WorkerNameLabelKey: "test-worker",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					EncryptionConfiguration: &kubermaticv1.EncryptionConfiguration{
						Enabled: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: clusterNamespace,
				},
			},
			masterSecret: generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
				ClusterNameAnnotation: testClusterName,
			}),
			existingSecretInUC: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: clusterNamespace,
				},
				Data: map[string][]byte{
					"key": []byte(testEncryptionKey),
				},
			},
			expectMasterSecretDeleted: true,
			expectUCSecretDeleted:     true,
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

			// Setup seed client with existing cluster and secret
			var seedObjects []ctrlruntimeclient.Object
			if tc.clusterToDelete != nil {
				seedObjects = append(seedObjects, tc.clusterToDelete)
			}
			if tc.existingSecretInUC != nil {
				seedObjects = append(seedObjects, tc.existingSecretInUC)
			}
			seedClient := fake.NewClientBuilder().WithObjects(seedObjects...).Build()

			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{testSeedName: seedClient},
				namespace:    masterNamespace,
				workerName:   "test-worker",
			}

			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testClusterName,
					Namespace: testSeedName,
				},
			}

			_, err := r.Reconcile(ctx, request)
			if err != nil {
				t.Errorf("Reconcile failed: %v", err)
			}

			// Check if master secret was deleted
			masterSecret := &corev1.Secret{}
			err = masterClient.Get(ctx, types.NamespacedName{
				Name:      testSecretName,
				Namespace: masterNamespace,
			}, masterSecret)

			if tc.expectMasterSecretDeleted {
				if err == nil {
					t.Errorf("expected master secret to be deleted but it still exists")
				} else if !apierrors.IsNotFound(err) {
					t.Errorf("expected NotFound error for master secret but got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected master secret to remain but got error: %v", err)
				}
			}

			// Check if UC secret was deleted
			ucSecret := &corev1.Secret{}
			err = seedClient.Get(ctx, types.NamespacedName{
				Name:      testSecretName,
				Namespace: clusterNamespace,
			}, ucSecret)

			if tc.expectUCSecretDeleted {
				if err == nil {
					t.Errorf("expected UC secret to be deleted but it still exists")
				} else if !apierrors.IsNotFound(err) {
					t.Errorf("expected NotFound error for UC secret but got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected UC secret to remain but got error: %v", err)
				}
			}
		})
	}
}

func TestEncryptionDisabledCleanup(t *testing.T) {
	testCases := []struct {
		name                string
		clusterHasFinalizer bool
		secretExists        bool
		expectSecretDeleted bool
		expectCleanupCalled bool
	}{
		{
			name:                "encryption disabled with finalizer - should cleanup",
			clusterHasFinalizer: true,
			secretExists:        true,
			expectSecretDeleted: true,
			expectCleanupCalled: true,
		},
		{
			name:                "encryption disabled without finalizer - should skip cleanup",
			clusterHasFinalizer: false,
			secretExists:        true,
			expectSecretDeleted: false,
			expectCleanupCalled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Setup master client with secret if needed
			var masterObjects []ctrlruntimeclient.Object
			if tc.secretExists {
				masterSecret := generateEncryptionSecret(testSecretName, masterNamespace, map[string]string{
					ClusterNameAnnotation: testClusterName,
				})
				masterObjects = append(masterObjects, masterSecret)
			}
			masterClient := fake.NewClientBuilder().WithObjects(masterObjects...).Build()

			// Cluster with encryption DISABLED
			clusterWithEncryptionDisabled := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: testClusterName,
					Labels: map[string]string{
						kubermaticv1.WorkerNameLabelKey: "test-worker",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					// No encryption configuration = disabled
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: clusterNamespace,
				},
			}

			if tc.clusterHasFinalizer {
				clusterWithEncryptionDisabled.Finalizers = []string{EncryptionSecretCleanupFinalizer}
			}

			seedClient := fake.NewClientBuilder().WithObjects(clusterWithEncryptionDisabled).Build()

			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{testSeedName: seedClient},
				namespace:    masterNamespace,
				workerName:   "test-worker",
			}

			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testClusterName,
					Namespace: testSeedName,
				},
			}

			// Execute reconcile
			_, err := r.Reconcile(ctx, request)
			if err != nil {
				t.Errorf("Reconcile failed: %v", err)
			}

			// Check if secret was deleted as expected
			if tc.secretExists {
				masterSecret := &corev1.Secret{}
				err = masterClient.Get(ctx, types.NamespacedName{
					Name:      testSecretName,
					Namespace: masterNamespace,
				}, masterSecret)

				if tc.expectSecretDeleted {
					if err == nil {
						t.Errorf("Expected secret to be deleted but it still exists")
					} else if !apierrors.IsNotFound(err) {
						t.Errorf("Expected NotFound error but got: %v", err)
					}
				} else {
					if err != nil {
						t.Errorf("Expected secret to remain but got error: %v", err)
					}
				}
			}

			// Verify that finalizer was always removed (regardless of cleanup)
			updatedCluster := &kubermaticv1.Cluster{}
			err = seedClient.Get(ctx, types.NamespacedName{Name: testClusterName}, updatedCluster)
			if err != nil {
				t.Errorf("Failed to get updated cluster: %v", err)
			}

			for _, finalizer := range updatedCluster.Finalizers {
				if finalizer == EncryptionSecretCleanupFinalizer {
					t.Errorf("Expected finalizer to be removed when encryption is disabled, but it still exists")
				}
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
			Labels: map[string]string{
				kubermaticv1.WorkerNameLabelKey: "test-worker",
			},
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
