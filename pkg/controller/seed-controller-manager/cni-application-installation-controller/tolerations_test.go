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

package cniapplicationinstallationcontroller

import (
	"testing"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
)

func TestOverrideValuesTolerations(t *testing.T) {
	configured := []corev1.Toleration{{Key: "node-role", Operator: corev1.TolerationOpEqual, Value: "system", Effect: corev1.TaintEffectNoSchedule}}
	configuredRaw := map[string]any{"key": "node-role", "operator": "Equal", "value": "system", "effect": "NoSchedule"}

	operatorDefaultsRaw := []any{
		map[string]any{"key": "node-role.kubernetes.io/control-plane", "operator": "Exists"},
		map[string]any{"key": "node-role.kubernetes.io/master", "operator": "Exists"},
		map[string]any{"key": "node.kubernetes.io/not-ready", "operator": "Exists"},
		map[string]any{"key": "node.cloudprovider.kubernetes.io/uninitialized", "operator": "Exists"},
	}

	testCases := []struct {
		name             string
		ciliumVersion    string
		configured       []corev1.Toleration
		expectedOperator any
		expectedOthers   any
	}{
		{
			name:          "nothing configured: no toleration values are enforced",
			ciliumVersion: "1.19.7",
		},
		{
			name:             "operator keeps the chart defaults",
			ciliumVersion:    "1.19.7",
			configured:       configured,
			expectedOperator: append(append([]any{}, operatorDefaultsRaw...), configuredRaw),
			expectedOthers:   []any{configuredRaw},
		},
		{
			name:           "operator tolerating everything is left alone",
			ciliumVersion:  "1.17.16",
			configured:     configured,
			expectedOthers: []any{configuredRaw},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					CNIPlugin: &kubermaticv1.CNIPluginSettings{
						Type:    kubermaticv1.CNIPluginTypeCilium,
						Version: tc.ciliumVersion,
					},
				},
			}
			if tc.configured != nil {
				cluster.Spec.ComponentsOverride.UserClusterWorkloads = &kubermaticv1.UserClusterWorkloadSettings{Tolerations: tc.configured}
			}

			values := getAppInstallOverrideValues(cluster, "")

			hubble := values["hubble"].(map[string]any)

			require.Equal(t, tc.expectedOperator, values["operator"].(map[string]any)["tolerations"])
			require.Equal(t, tc.expectedOthers, values["certgen"].(map[string]any)["tolerations"])
			require.Equal(t, tc.expectedOthers, hubble["relay"].(map[string]any)["tolerations"])
			require.Equal(t, tc.expectedOthers, hubble["ui"].(map[string]any)["tolerations"])
			require.NotContains(t, values, "tolerations")
			require.NotContains(t, values["envoy"], "tolerations")
		})
	}
}
