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

package kubermatic

import (
	"testing"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaceReconciler(t *testing.T) {
	testCases := []struct {
		name     string
		existing *corev1.Namespace
		validate func(t *testing.T, ns *corev1.Namespace)
	}{
		{
			name:     "namespace gets gateway-access label when labels is nil",
			existing: &corev1.Namespace{},
			validate: func(t *testing.T, ns *corev1.Namespace) {
				if ns.Labels == nil {
					t.Error("Expected labels to be initialized")
					return
				}
				if ns.Labels[common.GatewayAccessLabelKey] != "true" {
					t.Errorf("Expected label %q=true, got %q",
						common.GatewayAccessLabelKey, ns.Labels[common.GatewayAccessLabelKey])
				}
			},
		},
		{
			name: "namespace gets gateway-access label when labels exists",
			existing: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"existing-label": "value",
					},
				},
			},
			validate: func(t *testing.T, ns *corev1.Namespace) {
				if ns.Labels[common.GatewayAccessLabelKey] != "true" {
					t.Errorf("Expected label %q=true", common.GatewayAccessLabelKey)
				}
				// Ensure existing labels are preserved
				if ns.Labels["existing-label"] != "value" {
					t.Error("Existing label was not preserved")
				}
			},
		},
		{
			name: "namespace gateway-access label is overwritten if already set",
			existing: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						common.GatewayAccessLabelKey: "false",
					},
				},
			},
			validate: func(t *testing.T, ns *corev1.Namespace) {
				if ns.Labels[common.GatewayAccessLabelKey] != "true" {
					t.Errorf("Expected label %q to be overwritten to 'true'", common.GatewayAccessLabelKey)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creatorGetter := NamespaceReconciler("kubermatic")
			_, creator := creatorGetter()

			reconciled, err := creator(tc.existing)
			if err != nil {
				t.Fatalf("NamespaceReconciler failed: %v", err)
			}

			if tc.validate != nil {
				tc.validate(t, reconciled)
			}
		})
	}
}
