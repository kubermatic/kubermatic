/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package mlacontroller

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/cortex"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var testScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(kubermaticv1.AddToScheme(testScheme))
}

func newTestAlertmanagerReconciler(objects []ctrlruntimeclient.Object) (*alertmanagerReconciler, cortex.Client) {
	fakeClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		WithScheme(testScheme).
		Build()

	cClient := cortex.NewFakeClient()

	return &alertmanagerReconciler{
		seedClient: fakeClient,
		log:        zap.NewNop().Sugar(),
		recorder:   record.NewFakeRecorder(10),
		versions:   kubermatic.NewFakeVersions(edition.CommunityEdition),
		cortexClientProvider: func() cortex.Client {
			return cClient
		},
	}, cClient
}

type alertmanagerConfigStatus struct {
	clusterStatus      kubermaticv1.HealthStatus                    // Alertmanager config status in the Cluster CR
	alertmanagerStatus kubermaticv1.AlertmanagerConfigurationStatus // Alertmanager config status in the Alertmanager CR
}

func TestAlertmanagerReconcile(t *testing.T) {
	ctx := context.Background()
	clusterName := "test"
	clusterNamespace := fmt.Sprintf("cluster-%s", clusterName)

	testCases := []struct {
		name           string
		objects        []ctrlruntimeclient.Object
		expectedErr    bool
		hasFinalizer   bool
		hasResources   bool
		expectedStatus alertmanagerConfigStatus
	}{
		{
			name: "create default alertmanager configuration when no alertmanager is created",
			objects: []ctrlruntimeclient.Object{
				generateCluster(clusterName, true, false, false),
			},
			hasFinalizer:   true,
			hasResources:   true,
			expectedStatus: getAlertmanagerConfigStatusUp(),
		},
		{
			name: "clean up alertmanager configuration when mla is disabled",
			objects: []ctrlruntimeclient.Object{
				generateCluster(clusterName, false, false, false),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config-secret",
						Namespace: clusterNamespace,
					},
					Data: map[string][]byte{
						resources.AlertmanagerConfigSecretKey: []byte(generateAlertmanagerConfig("test-user")),
					},
				},
				&kubermaticv1.Alertmanager{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.AlertmanagerName,
						Namespace: clusterNamespace,
					},
					Spec: kubermaticv1.AlertmanagerSpec{
						ConfigSecret: corev1.LocalObjectReference{
							Name: "config-secret",
						},
					},
				},
			},
			hasFinalizer: false,
			hasResources: false,
		},
		{
			name: "clean up alertmanager configuration when cluster is removed",
			objects: []ctrlruntimeclient.Object{
				generateCluster(clusterName, false, true, true),
			},
			hasFinalizer: false,
			hasResources: false,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler, _ := newTestAlertmanagerReconciler(tc.objects)
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}}

			_, err := reconciler.Reconcile(ctx, request)
			if tc.expectedErr != (err != nil) {
				t.Fatalf("ExpectedErr = %v, but got: %v", tc.expectedErr, err)
			}

			cluster := &kubermaticv1.Cluster{}
			if err := reconciler.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
				t.Fatalf("Failed to get cluster: %v", err)
			}

			if tc.hasFinalizer != kubernetes.HasFinalizer(cluster, alertmanagerFinalizer) {
				t.Fatalf("Expected Finalizer=%v, failed to assert that.", tc.hasFinalizer)
			}

			alertmanager := &kubermaticv1.Alertmanager{}
			err = reconciler.seedClient.Get(ctx, types.NamespacedName{
				Name:      resources.AlertmanagerName,
				Namespace: cluster.Status.NamespaceName,
			}, alertmanager)

			if tc.hasResources {
				if err != nil {
					t.Fatalf("Failed to get Alertmanager: %v", err)
				}

				secret := &corev1.Secret{}
				err = reconciler.seedClient.Get(ctx, types.NamespacedName{
					Name:      alertmanager.Spec.ConfigSecret.Name,
					Namespace: cluster.Status.NamespaceName,
				}, secret)
				if err != nil {
					t.Fatalf("Failed to get Alertmanager secret: %v", err)
				}

				if tc.expectedStatus.clusterStatus != *cluster.Status.ExtendedHealth.AlertmanagerConfig {
					t.Fatalf("Expected clusterstatus to be %v, but got: %v", tc.expectedStatus.clusterStatus, *cluster.Status.ExtendedHealth.AlertmanagerConfig)
				}

				if tc.expectedStatus.alertmanagerStatus.Status != alertmanager.Status.ConfigStatus.Status {
					t.Fatalf("Expected alertmanager status to be %v, but got: %v", tc.expectedStatus.alertmanagerStatus.Status, alertmanager.Status.ConfigStatus.Status)
				}

				if tc.expectedStatus.alertmanagerStatus.ErrorMessage != alertmanager.Status.ConfigStatus.ErrorMessage {
					t.Fatalf("Expected alertmanager error message to be %v, but got: %v", tc.expectedStatus.alertmanagerStatus.ErrorMessage, alertmanager.Status.ConfigStatus.ErrorMessage)
				}

				if tc.expectedStatus.alertmanagerStatus.Status == corev1.ConditionTrue {
					if alertmanager.Status.ConfigStatus.LastUpdated.IsZero() {
						t.Fatal("Expected to a non-zero LastUpdated time, but it was zero.")
					}
				}
			} else {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("Expected NotFound error, but got: %v", err)
				}

				secretList := &corev1.SecretList{}
				err = reconciler.seedClient.List(ctx, secretList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
				if err != nil {
					t.Fatalf("Failed to list Secrets: %v", err)
				}
				if len(secretList.Items) > 0 {
					t.Fatalf("Expected to find no more Secrets, but found %d.", len(secretList.Items))
				}
				// No alertmanager config status any more in Cluster CR
				if cfg := cluster.Status.ExtendedHealth.AlertmanagerConfig; cfg != nil {
					t.Fatalf("Expected to find nil AlertmanagerConfig, but got: %v", cfg)
				}
			}
		})
	}
}

// getAlertmanagerConfigStatusUp returns the needed information when the alertmanager config status is OK.
func getAlertmanagerConfigStatusUp() alertmanagerConfigStatus {
	return alertmanagerConfigStatus{
		clusterStatus: kubermaticv1.HealthStatusUp,
		alertmanagerStatus: kubermaticv1.AlertmanagerConfigurationStatus{
			Status:      corev1.ConditionTrue,
			LastUpdated: metav1.Now(),
		},
	}
}

func generateCluster(name string, monitoringEnabled, loggingEnabled, deleted bool) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: name,
			MLA: &kubermaticv1.MLASettings{
				MonitoringEnabled: monitoringEnabled,
				LoggingEnabled:    loggingEnabled,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: fmt.Sprintf("cluster-%s", name),
		},
	}

	if deleted {
		deleteTime := metav1.Now()
		cluster.DeletionTimestamp = &deleteTime

		// to validate objects after reconciling, we want to prevent the fakeclient
		// from deleting them once all finalizers are gone; we achieve this by
		// attached a dummy finalizer
		kubernetes.AddFinalizer(cluster, "just-a-test-do-not-delete-thanks")
	}

	return cluster
}

func generateAlertmanagerConfig(name string) string {
	return fmt.Sprintf(`
alertmanager_config: |
  global:
    smtp_smarthost: 'localhost:25'
    smtp_from: '%s@example.org'
  route:
    receiver: "test"
  receivers:
    - name: "test"
      email_configs:
      - to: '%s@example.org'
`, name, name)
}
