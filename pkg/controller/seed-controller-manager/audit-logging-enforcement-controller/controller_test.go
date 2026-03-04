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

package auditloggingenforcement

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	datacenterName = "test-dc"
	seedName       = "test-seed"
)

func TestReconcile(t *testing.T) {
	// This test verifies the audit logging enforcement controller behavior.
	// The controller updates the Cluster CR's spec.auditLogging field.
	// The Kubernetes controller (apiserver/deployment.go) then:
	//   - Reads cluster.Spec.AuditLogging.Enabled
	//   - When Enabled=true: creates audit ConfigMaps, Secrets, and fluent-bit sidecar
	//   - When Enabled=false: removes all audit logging resources
	//   - When nil: no audit logging resources are created
	// This ensures disabled state (Enabled=false) is properly propagated to user clusters.

	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	testCases := []struct {
		name                     string
		cluster                  *kubermaticv1.Cluster
		seed                     *kubermaticv1.Seed
		expectedAuditLogging     *kubermaticv1.AuditLoggingSettings
		shouldSkipReconciliation bool
	}{
		{
			name:                 "scenario 1: enforce audit logging from seed to cluster",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
		},
		{
			name:                 "scenario 2: skip enforcement when cluster has opt-out annotation",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, true),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: nil,
		},
		{
			name:                 "scenario 3: leave cluster unchanged when datacenter has EnforceAuditLogging disabled",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			expectedAuditLogging: nil,
		},
		{
			name:                     "scenario 4: skip enforcement when cluster is paused",
			cluster:                  genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                     genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging:     nil,
			shouldSkipReconciliation: true,
		},
		{
			name:                 "scenario 5: no update when audit logging already matches",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
		},
		{
			name:                 "scenario 6: update audit logging policy when seed changes",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyMetadata), false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
		},
		{
			name:                 "scenario 7: enforce enabled even when seed has no audit logging config",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, nil, true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{Enabled: true},
		},
		{
			name:                 "scenario 8: enforce enabled even when seed disables audit logging",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(false, ""), true),
			expectedAuditLogging: genAuditLoggingSettings(true, ""),
		},
		{
			name:                 "scenario 9: enforce enabled with empty policy preset",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			seed:                 genSeedWithAuditLogging(datacenterName, &kubermaticv1.AuditLoggingSettings{Enabled: false}, true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{Enabled: true},
		},
		{
			name:                 "scenario 10: enforce when seed has config and datacenter enforcement is enabled",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
		},
		{
			name:    "scenario 11: enforce includes EnforcedAuditWebhookSettings from datacenter",
			cluster: genClusterWithAuditLogging(datacenterName, nil, false),
			seed:    genSeedWithAuditLoggingAndWebhook(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "audit-webhook-secret",
						Namespace: "kube-system",
					},
				},
			},
		},
		{
			name:    "scenario 12: enforce=true, seed=nil, DC has EnforcedAuditWebhookSettings",
			cluster: genClusterWithAuditLogging(datacenterName, nil, false),
			seed:    genSeedWithAuditLoggingAndWebhook(datacenterName, nil, true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled: true,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "audit-webhook-secret",
						Namespace: "kube-system",
					},
				},
			},
		},
		{
			name:                 "scenario 13: enforce=false, DC has EnforcedAuditWebhookSettings are ignored",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLoggingAndWebhook(datacenterName, nil, false),
			expectedAuditLogging: nil,
		},
		{
			name: "scenario 14: no-op when cluster already matches including WebhookBackend",
			cluster: genClusterWithAuditLogging(datacenterName, &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "audit-webhook-secret",
						Namespace: "kube-system",
					},
				},
			}, false),
			seed: genSeedWithAuditLoggingAndWebhook(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "audit-webhook-secret",
						Namespace: "kube-system",
					},
				},
			},
		},
		{
			name: "scenario 15: DC enforced webhook overrides cluster's existing webhook",
			cluster: genClusterWithAuditLogging(datacenterName, &kubermaticv1.AuditLoggingSettings{
				Enabled: true,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "old-webhook-secret",
						Namespace: "kube-system",
					},
				},
			}, false),
			seed: genSeedWithAuditLoggingAndWebhook(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "audit-webhook-secret",
						Namespace: "kube-system",
					},
				},
			},
		},
		{
			name: "scenario 16: removing DC webhook settings clears cluster's WebhookBackend",
			cluster: genClusterWithAuditLogging(datacenterName, &kubermaticv1.AuditLoggingSettings{
				Enabled: true,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "old-webhook-secret",
						Namespace: "kube-system",
					},
				},
			}, false),
			seed:                 genSeedWithAuditLogging(datacenterName, nil, true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{Enabled: true},
		},
		{
			name:    "scenario 17: DC webhook overrides seed's own WebhookBackend",
			cluster: genClusterWithAuditLogging(datacenterName, nil, false),
			seed: genSeedWithDCWebhook(datacenterName, &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "seed-webhook-secret",
						Namespace: "kube-system",
					},
				},
			}, true, &kubermaticv1.AuditWebhookBackendSettings{
				AuditWebhookConfig: &corev1.SecretReference{
					Name:      "dc-webhook-secret",
					Namespace: "kube-system",
				},
			}),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "dc-webhook-secret",
						Namespace: "kube-system",
					},
				},
			},
		},
		{
			name:    "scenario 18: seed SidecarSettings are propagated to cluster",
			cluster: genClusterWithAuditLogging(datacenterName, nil, false),
			seed: genSeedWithAuditLogging(datacenterName, &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				SidecarSettings: &kubermaticv1.AuditSidecarSettings{
					ExtraEnvs: []corev1.EnvVar{
						{Name: "TEST_VAR", Value: "test-value"},
					},
				},
			}, true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
				SidecarSettings: &kubermaticv1.AuditSidecarSettings{
					ExtraEnvs: []corev1.EnvVar{
						{Name: "TEST_VAR", Value: "test-value"},
					},
				},
			},
		},
		{
			name: "scenario 19: seed=nil replaces cluster's existing AuditLogging with bare Enabled=true",
			cluster: genClusterWithAuditLogging(datacenterName, &kubermaticv1.AuditLoggingSettings{
				Enabled:      false,
				PolicyPreset: kubermaticv1.AuditPolicyMinimal,
				WebhookBackend: &kubermaticv1.AuditWebhookBackendSettings{
					AuditWebhookConfig: &corev1.SecretReference{
						Name:      "old-webhook",
						Namespace: "kube-system",
					},
				},
			}, false),
			seed:                 genSeedWithAuditLogging(datacenterName, nil, true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{Enabled: true},
		},
		{
			name:                 "scenario 20: cluster with empty DatacenterName is skipped",
			cluster:              genClusterWithAuditLogging("", nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: nil,
		},
		{
			name:                 "scenario 21: cluster's datacenter not found in seed is skipped",
			cluster:              genClusterWithAuditLogging("nonexistent-dc", nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: nil,
		},
		{
			name:                 "scenario 22: opt-out annotation 'false' does not skip enforcement",
			cluster:              genClusterWithAnnotationValue(datacenterName, nil, "false"),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Handle paused cluster scenario
			if tc.shouldSkipReconciliation {
				tc.cluster.Spec.Pause = true
			}

			seedClient := fake.
				NewClientBuilder().
				WithObjects(tc.cluster, tc.seed).
				Build()

			seedGetter := func() (*kubermaticv1.Seed, error) {
				return tc.seed, nil
			}

			r := &reconciler{
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				recorder:                &events.FakeRecorder{},
				seedGetter:              seedGetter,
				seedClient:              seedClient,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.cluster.Name}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			// Get the updated cluster
			updatedCluster := &kubermaticv1.Cluster{}
			err = seedClient.Get(ctx, types.NamespacedName{Name: tc.cluster.Name}, updatedCluster)
			if err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			// Check if audit logging matches expected
			if !diff.SemanticallyEqual(tc.expectedAuditLogging, updatedCluster.Spec.AuditLogging) {
				t.Fatalf("audit logging config mismatch:\n%v", diff.ObjectDiff(tc.expectedAuditLogging, updatedCluster.Spec.AuditLogging))
			}
		})
	}
}

func genClusterWithAuditLogging(datacenterName string, auditLogging *kubermaticv1.AuditLoggingSettings, optOut bool) *kubermaticv1.Cluster {
	cluster := generator.GenDefaultCluster()
	cluster.Spec.Cloud.DatacenterName = datacenterName
	cluster.Spec.AuditLogging = auditLogging

	if optOut {
		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		cluster.Annotations[kubermaticv1.SkipAuditLoggingEnforcementAnnotation] = "true"
	}

	return cluster
}

func genSeedWithAuditLogging(datacenterName string, auditLogging *kubermaticv1.AuditLoggingSettings, enforceAuditLogging bool) *kubermaticv1.Seed {
	seed := generator.GenTestSeed()
	seed.Name = seedName
	seed.Spec.AuditLogging = auditLogging
	seed.Spec.Datacenters = map[string]kubermaticv1.Datacenter{
		datacenterName: {
			Country:  "US",
			Location: "Test Location",
			Spec: kubermaticv1.DatacenterSpec{
				EnforceAuditLogging: enforceAuditLogging,
			},
		},
	}
	return seed
}

func genSeedWithAuditLoggingAndWebhook(datacenterName string, auditLogging *kubermaticv1.AuditLoggingSettings, enforceAuditLogging bool) *kubermaticv1.Seed {
	seed := genSeedWithAuditLogging(datacenterName, auditLogging, enforceAuditLogging)
	dc := seed.Spec.Datacenters[datacenterName]
	dc.Spec.EnforcedAuditWebhookSettings = &kubermaticv1.AuditWebhookBackendSettings{
		AuditWebhookConfig: &corev1.SecretReference{
			Name:      "audit-webhook-secret",
			Namespace: "kube-system",
		},
	}
	seed.Spec.Datacenters[datacenterName] = dc
	return seed
}

func genAuditLoggingSettings(enabled bool, policyPreset kubermaticv1.AuditPolicyPreset) *kubermaticv1.AuditLoggingSettings {
	return &kubermaticv1.AuditLoggingSettings{
		Enabled:      enabled,
		PolicyPreset: policyPreset,
	}
}

func genSeedWithDCWebhook(datacenterName string, auditLogging *kubermaticv1.AuditLoggingSettings, enforceAuditLogging bool, dcWebhook *kubermaticv1.AuditWebhookBackendSettings) *kubermaticv1.Seed {
	seed := genSeedWithAuditLogging(datacenterName, auditLogging, enforceAuditLogging)
	if dcWebhook != nil {
		dc := seed.Spec.Datacenters[datacenterName]
		dc.Spec.EnforcedAuditWebhookSettings = dcWebhook
		seed.Spec.Datacenters[datacenterName] = dc
	}
	return seed
}

func genClusterWithAnnotationValue(datacenterName string, auditLogging *kubermaticv1.AuditLoggingSettings, annotationValue string) *kubermaticv1.Cluster {
	cluster := genClusterWithAuditLogging(datacenterName, auditLogging, false)
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	cluster.Annotations[kubermaticv1.SkipAuditLoggingEnforcementAnnotation] = annotationValue
	return cluster
}

func TestReconcileDeletedCluster(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	cluster := genClusterWithAuditLogging(datacenterName, nil, false)
	now := metav1.Now()
	cluster.DeletionTimestamp = &now
	cluster.Finalizers = []string{"test-finalizer"}

	seed := genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true)

	seedClient := fake.
		NewClientBuilder().
		WithObjects(cluster, seed).
		Build()

	r := &reconciler{
		log:                     kubermaticlog.Logger,
		workerNameLabelSelector: workerSelector,
		recorder:                &events.FakeRecorder{},
		seedGetter:              func() (*kubermaticv1.Seed, error) { return seed, nil },
		seedClient:              seedClient,
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}}
	if _, err := r.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconciling failed: %v", err)
	}

	updatedCluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(context.Background(), types.NamespacedName{Name: cluster.Name}, updatedCluster); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}

	if updatedCluster.Spec.AuditLogging != nil {
		t.Fatalf("expected nil AuditLogging for deleted cluster, got: %v", updatedCluster.Spec.AuditLogging)
	}
}

func TestReconcileSeedGetterError(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	cluster := genClusterWithAuditLogging(datacenterName, nil, false)
	seed := genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true)

	seedClient := fake.
		NewClientBuilder().
		WithObjects(cluster, seed).
		Build()

	r := &reconciler{
		log:                     kubermaticlog.Logger,
		workerNameLabelSelector: workerSelector,
		recorder:                &events.FakeRecorder{},
		seedGetter:              func() (*kubermaticv1.Seed, error) { return nil, fmt.Errorf("seed getter error") },
		seedClient:              seedClient,
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}}
	_, err = r.Reconcile(context.Background(), request)
	if err == nil {
		t.Fatal("expected error from seedGetter, got nil")
	}
}

func TestReconcileClusterNotFound(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	seed := genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true)

	seedClient := fake.
		NewClientBuilder().
		WithObjects(seed).
		Build()

	r := &reconciler{
		log:                     kubermaticlog.Logger,
		workerNameLabelSelector: workerSelector,
		recorder:                &events.FakeRecorder{},
		seedGetter:              func() (*kubermaticv1.Seed, error) { return seed, nil },
		seedClient:              seedClient,
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "nonexistent-cluster"}}
	_, err = r.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatalf("expected no error for not-found cluster, got: %v", err)
	}
}
