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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
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
		expectUpdate             bool
		shouldSkipReconciliation bool
	}{
		{
			name:                 "scenario 1: enforce audit logging from seed to cluster",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
			expectUpdate:         true,
		},
		{
			name:                 "scenario 2: skip enforcement when cluster has opt-out annotation",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, true),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: nil,
			expectUpdate:         false,
		},
		{
			name:                 "scenario 3: disable audit logging when datacenter has EnforceAuditLogging disabled",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{Enabled: false},
			expectUpdate:         true,
		},
		{
			name:                     "scenario 4: skip enforcement when cluster is paused",
			cluster:                  genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                     genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging:     nil,
			expectUpdate:             false,
			shouldSkipReconciliation: true,
		},
		{
			name:                 "scenario 5: no update when audit logging already matches",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
			expectUpdate:         false,
		},
		{
			name:                 "scenario 6: update audit logging policy when seed changes",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyMetadata), false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
			expectUpdate:         true,
		},
		{
			name:                 "scenario 7: skip enforcement when seed has no audit logging config",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			seed:                 genSeedWithAuditLogging(datacenterName, nil, true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
			expectUpdate:         false,
		},
		{
			name:                 "scenario 8: enforce disabled state when seed explicitly disables audit logging",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(false, ""), true),
			expectedAuditLogging: genAuditLoggingSettings(false, ""),
			expectUpdate:         true,
		},
		{
			name:                 "scenario 9: enforce disabled state with empty policy preset",
			cluster:              genClusterWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), false),
			seed:                 genSeedWithAuditLogging(datacenterName, &kubermaticv1.AuditLoggingSettings{Enabled: false}, true),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{Enabled: false},
			expectUpdate:         true,
		},
		{
			name:                 "scenario 10: enforce when seed has config and datacenter enforcement is enabled",
			cluster:              genClusterWithAuditLogging(datacenterName, nil, false),
			seed:                 genSeedWithAuditLogging(datacenterName, genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended), true),
			expectedAuditLogging: genAuditLoggingSettings(true, kubermaticv1.AuditPolicyRecommended),
			expectUpdate:         true,
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
				recorder:                &record.FakeRecorder{},
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

func genAuditLoggingSettings(enabled bool, policyPreset kubermaticv1.AuditPolicyPreset) *kubermaticv1.AuditLoggingSettings {
	return &kubermaticv1.AuditLoggingSettings{
		Enabled:      enabled,
		PolicyPreset: policyPreset,
	}
}
