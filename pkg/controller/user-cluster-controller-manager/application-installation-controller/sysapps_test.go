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

package applicationinstallationcontroller

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
)

func TestGenerateClusterAutoscalerValues(t *testing.T) {
	tolerations := []corev1.Toleration{{Key: "node-role", Operator: corev1.TolerationOpEqual, Value: "system", Effect: corev1.TaintEffectNoSchedule}}

	t.Run("without tolerations the key is not emitted", func(t *testing.T) {
		values := generateClusterAutoscalerValues(nil, "", nil)

		require.NotContains(t, values, "tolerations")
		require.Contains(t, values, "image")
	})

	t.Run("configured tolerations are emitted", func(t *testing.T) {
		values := generateClusterAutoscalerValues(nil, "", tolerations)

		require.Equal(t, []any{
			map[string]any{"key": "node-role", "operator": "Equal", "value": "system", "effect": "NoSchedule"},
		}, values["tolerations"])
	})

	t.Run("registry override is independent of tolerations", func(t *testing.T) {
		values := generateClusterAutoscalerValues(nil, "registry.example.com", tolerations)

		image := values["image"].(map[string]any)
		require.Equal(t, "registry.example.com/autoscaling/cluster-autoscaler", image["repository"])
		require.Contains(t, values, "tolerations")
	})
}
