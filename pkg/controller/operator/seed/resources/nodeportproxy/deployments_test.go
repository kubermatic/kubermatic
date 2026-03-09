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

package nodeportproxy

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
)

func TestEnvoyDeploymentReconcilerReplicas(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name              string
		configuredReplica *int32
		expectedReplicas  int32
	}{
		{
			name:              "uses default when replicas are not configured",
			configuredReplica: nil,
			expectedReplicas:  DefaultEnvoyReplicas,
		},
		{
			name:              "uses configured replicas",
			configuredReplica: ptr.To[int32](5),
			expectedReplicas:  5,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seed := &kubermaticv1.Seed{}
			seed.Spec.NodeportProxy.Envoy.Replicas = tc.configuredReplica

			_, reconcile := EnvoyDeploymentReconciler(&kubermaticv1.KubermaticConfiguration{}, seed, false, kubermatic.Versions{
				KubermaticContainerTag: "v0.0.0-test",
			})()

			reconciled, err := reconcile(&appsv1.Deployment{})
			if err != nil {
				t.Fatalf("failed to reconcile deployment: %v", err)
			}

			if reconciled.Spec.Replicas == nil {
				t.Fatal("expected deployment replicas to be set")
			}

			if *reconciled.Spec.Replicas != tc.expectedReplicas {
				t.Fatalf("unexpected replica count: got %d, want %d", *reconciled.Spec.Replicas, tc.expectedReplicas)
			}
		})
	}
}
