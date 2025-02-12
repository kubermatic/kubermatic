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

package kyvernopolicycontroller

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// type fakeClientProvider struct {
// 	client ctrlruntimeclient.Client
// }

// func (f *fakeClientProvider) GetClient(_ context.Context, _ *kubermaticv1.Cluster, _ ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
// 	return f.client, nil
// }

// func TestReconcile(t *testing.T) {
// 	tests := []struct {
// 		name               string
// 		policyBinding      *kubermaticv1.PolicyBinding
// 		policyTemplate     *kubermaticv1.PolicyTemplate
// 		clusters           []*kubermaticv1.Cluster
// 		existingPolicies   []*kyvernov1.ClusterPolicy
// 		expectedConditions []metav1.Condition
// 		expectedPolicies   []*kyvernov1.ClusterPolicy
// 	}{
// 		{
// 			name: "successfully creates policies for global binding with project selector",
// 			policyBinding: &kubermaticv1.PolicyBinding{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "test-binding",
// 				},
// 				Spec: kubermaticv1.PolicyBindingSpec{
// 					Scope: kubermaticv1.PolicyBindingScopeGlobal,
// 					PolicyTemplateRef: corev1.ObjectReference{
// 						Name: "test-template",
// 					},
// 					Target: kubermaticv1.PolicyTargetSpec{
// 						Projects: kubermaticv1.ResourceSelector{
// 							Name: []string{"project-1"},
// 						},
// 					},
// 				},
// 			},
// 			policyTemplate: &kubermaticv1.PolicyTemplate{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "test-template",
// 				},
// 				Spec: kubermaticv1.PolicyTemplateSpec{
// 					Visibility: kubermaticv1.PolicyTemplateGlobalVisibility,
// 					KyvernoPolicySpec: kyvernov1.Spec{
// 						Rules: []kyvernov1.Rule{
// 							{
// 								Name: "test-rule",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			clusters: []*kubermaticv1.Cluster{
// 				{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name: "cluster-1",
// 						Labels: map[string]string{
// 							"project-id": "project-1",
// 						},
// 					},
// 				},
// 				{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name: "cluster-2",
// 						Labels: map[string]string{
// 							"project-id": "project-2",
// 						},
// 					},
// 				},
// 			},
// 			expectedPolicies: []*kyvernov1.ClusterPolicy{
// 				{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name: "test-template-test-binding",
// 						Labels: map[string]string{
// 							"app.kubernetes.io/managed-by": "kubermatic",
// 							"policy-binding":               "test-binding",
// 							"policy-template":              "test-template",
// 						},
// 					},
// 					Spec: kyvernov1.Spec{
// 						Rules: []kyvernov1.Rule{
// 							{
// 								Name: "test-rule",
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			name: "successfully creates policies for project binding with cluster selector",
// 			policyBinding: &kubermaticv1.PolicyBinding{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "test-binding",
// 				},
// 				Spec: kubermaticv1.PolicyBindingSpec{
// 					Scope: kubermaticv1.PolicyBindingScopeProject,
// 					PolicyTemplateRef: corev1.ObjectReference{
// 						Name: "test-template",
// 					},
// 					Target: kubermaticv1.PolicyTargetSpec{
// 						Clusters: kubermaticv1.ResourceSelector{
// 							Name: []string{"cluster-1"},
// 						},
// 					},
// 				},
// 			},
// 			policyTemplate: &kubermaticv1.PolicyTemplate{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "test-template",
// 				},
// 				Spec: kubermaticv1.PolicyTemplateSpec{
// 					Visibility: kubermaticv1.PolicyTemplateProjectVisibility,
// 					ProjectID:  "project-1",
// 					KyvernoPolicySpec: kyvernov1.Spec{
// 						Rules: []kyvernov1.Rule{
// 							{
// 								Name: "test-rule",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			clusters: []*kubermaticv1.Cluster{
// 				{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name: "cluster-1",
// 						Labels: map[string]string{
// 							"project-id": "project-1",
// 						},
// 					},
// 				},
// 				{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name: "cluster-2",
// 						Labels: map[string]string{
// 							"project-id": "project-1",
// 						},
// 					},
// 				},
// 			},
// 			expectedConditions: []metav1.Condition{
// 				{
// 					Type:    kubermaticv1.PolicyReadyCondition,
// 					Status:  metav1.ConditionTrue,
// 					Reason:  kubermaticv1.PolicyAppliedSuccessfully,
// 					Message: "Successfully applied policy to all target clusters",
// 				},
// 				{
// 					Type:    kubermaticv1.PolicyEnforcedCondition,
// 					Status:  metav1.ConditionTrue,
// 					Reason:  kubermaticv1.PolicyAppliedSuccessfully,
// 					Message: "Successfully applied policy to all target clusters",
// 				},
// 			},
// 			expectedPolicies: []*kyvernov1.ClusterPolicy{
// 				{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name: "test-template-test-binding",
// 						Labels: map[string]string{
// 							"app.kubernetes.io/managed-by": "kubermatic",
// 							"policy-binding":               "test-binding",
// 							"policy-template":              "test-template",
// 						},
// 					},
// 					Spec: kyvernov1.Spec{
// 						Rules: []kyvernov1.Rule{
// 							{
// 								Name: "test-rule",
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			// Create fake clients
// 			seedClient := fake.NewClientBuilder().
// 				WithObjects(tc.policyBinding, tc.policyTemplate).
// 				Build()

// 			for _, cluster := range tc.clusters {
// 				if err := seedClient.Create(context.Background(), cluster); err != nil {
// 					t.Fatalf("failed to create test cluster: %v", err)
// 				}
// 			}

// 			userClusterClient := fake.NewClientBuilder().Build()
// 			clientProvider := &fakeClientProvider{client: userClusterClient}

// 			// Create reconciler
// 			r := &Reconciler{
// 				log:            zap.NewNop().Sugar(),
// 				workerName:     "test",
// 				seedClient:     seedClient,
// 				clientProvider: clientProvider,
// 				recorder:       record.NewFakeRecorder(10),
// 			}

// 			// Run reconciliation
// 			if _, err := r.Reconcile(context.Background(), reconcile.Request{
// 				NamespacedName: types.NamespacedName{
// 					Name: tc.policyBinding.Name,
// 				},
// 			}); err != nil {
// 				t.Fatalf("reconciliation failed: %v", err)
// 			}

// 			// Verify policy binding conditions
// 			updatedBinding := &kubermaticv1.PolicyBinding{}
// 			if err := seedClient.Get(context.Background(), types.NamespacedName{Name: tc.policyBinding.Name}, updatedBinding); err != nil {
// 				t.Fatalf("failed to get updated policy binding: %v", err)
// 			}

// 			if len(updatedBinding.Status.Conditions) != len(tc.expectedConditions) {
// 				t.Errorf("expected %d conditions, got %d", len(tc.expectedConditions), len(updatedBinding.Status.Conditions))
// 			}

// 			for i, expectedCond := range tc.expectedConditions {
// 				actualCond := updatedBinding.Status.Conditions[i]
// 				if actualCond.Type != expectedCond.Type ||
// 					actualCond.Status != expectedCond.Status ||
// 					actualCond.Reason != expectedCond.Reason ||
// 					actualCond.Message != expectedCond.Message {
// 					t.Errorf("condition mismatch at index %d:\nexpected: %+v\ngot: %+v", i, expectedCond, actualCond)
// 				}
// 			}

// 			// Verify created/updated policies
// 			for _, expectedPolicy := range tc.expectedPolicies {
// 				actualPolicy := &kyvernov1.ClusterPolicy{}
// 				if err := userClusterClient.Get(context.Background(), types.NamespacedName{Name: expectedPolicy.Name}, actualPolicy); err != nil {
// 					t.Errorf("failed to get policy %s: %v", expectedPolicy.Name, err)
// 					continue
// 				}

// 				if !policyEqual(actualPolicy, expectedPolicy) {
// 					t.Errorf("policy mismatch:\nexpected: %+v\ngot: %+v", expectedPolicy, actualPolicy)
// 				}
// 			}
// 		})
// 	}
// }

// func policyEqual(a, b *kyvernov1.ClusterPolicy) bool {
// 	if a.Name != b.Name {
// 		return false
// 	}

// 	// Compare labels
// 	if len(a.Labels) != len(b.Labels) {
// 		return false
// 	}
// 	for k, v := range a.Labels {
// 		if b.Labels[k] != v {
// 			return false
// 		}
// 	}

// 	// Compare rules
// 	if len(a.Spec.Rules) != len(b.Spec.Rules) {
// 		return false
// 	}
// 	for i := range a.Spec.Rules {
// 		if a.Spec.Rules[i].Name != b.Spec.Rules[i].Name {
// 			return false
// 		}
// 	}

// 	return true
// }

func TestGetTargetProjectIDs(t *testing.T) {
	tests := []struct {
		name            string
		projectSelector kubermaticv1.ResourceSelector
		projects        []*kubermaticv1.Project
		expected        []string
		expectError     bool
	}{
		{
			name: "select all projects",
			projectSelector: kubermaticv1.ResourceSelector{
				SelectAll: true,
			},
			projects: []*kubermaticv1.Project{
				generator.GenProject("project-1", kubermaticv1.ProjectActive, time.Now()),
				generator.GenProject("project-2", kubermaticv1.ProjectActive, time.Now()),
			},
			expected: []string{"project-1-ID", "project-2-ID"},
		},
		{
			name: "select by name",
			projectSelector: kubermaticv1.ResourceSelector{
				Name: []string{"project-1"},
			},
			projects: []*kubermaticv1.Project{
				generator.GenProject("project-1", kubermaticv1.ProjectActive, time.Now()),
				generator.GenProject("project-2", kubermaticv1.ProjectActive, time.Now()),
			},
			expected: []string{"project-1"},
		},
		{
			name: "select by label",
			projectSelector: kubermaticv1.ResourceSelector{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"env": "prod",
					},
				},
			},
			projects: []*kubermaticv1.Project{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "project-1",
						Labels: map[string]string{
							"env": "prod",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "project-2",
						Labels: map[string]string{
							"env": "dev",
						},
					},
				},
			},
			expected: []string{"project-1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create fake client
			seedClient := fake.NewClientBuilder().Build()

			// Add test projects
			for _, project := range tc.projects {
				if err := seedClient.Create(context.Background(), project); err != nil {
					t.Fatalf("failed to create test project: %v", err)
				}
			}

			// Create reconciler
			r := &Reconciler{
				log:        zap.NewNop().Sugar(),
				seedClient: seedClient,
			}

			// Get target project IDs
			ids, err := r.getTargetProjectIDs(context.Background(), tc.projectSelector)
			if tc.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Compare results
			if len(ids) != len(tc.expected) {
				t.Errorf("expected %d project IDs, got %d", len(tc.expected), len(ids))
			}
			for i, id := range ids {
				if id != tc.expected[i] {
					t.Errorf("expected project ID %s at index %d, got %s", tc.expected[i], i, id)
				}
			}
		})
	}
}
