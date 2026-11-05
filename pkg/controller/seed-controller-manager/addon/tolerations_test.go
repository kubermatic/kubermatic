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

package addon

import (
	"testing"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func TestInjectTolerations(t *testing.T) {
	configured := []corev1.Toleration{{Key: "node-role", Operator: corev1.TolerationOpEqual, Value: "system", Effect: corev1.TaintEffectNoSchedule}}

	configuredRaw := map[string]any{"key": "node-role", "operator": "Equal", "value": "system", "effect": "NoSchedule"}
	tolerateEverythingRaw := map[string]any{"operator": "Exists"}

	testCases := []struct {
		name       string
		manifest   string
		configured []corev1.Toleration
		path       []string
		expected   []any
	}{
		{
			name: "Deployment without tolerations",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: csi-controller
spec:
  template:
    spec:
      containers: []
`,
			configured: configured,
			path:       []string{"spec", "template", "spec", "tolerations"},
			expected:   []any{configuredRaw},
		},
		{
			name: "DaemonSet keeps its own tolerations",
			manifest: `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: canal
spec:
  template:
    spec:
      tolerations:
      - operator: Exists
`,
			configured: configured,
			path:       []string{"spec", "template", "spec", "tolerations"},
			expected:   []any{tolerateEverythingRaw, configuredRaw},
		},
		{
			name: "toleration shipped with the addon is not duplicated",
			manifest: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test
spec:
  template:
    spec:
      tolerations:
      - key: node-role
        operator: Equal
        value: system
        effect: NoSchedule
`,
			configured: configured,
			path:       []string{"spec", "template", "spec", "tolerations"},
			expected:   []any{configuredRaw},
		},
		{
			name: "CronJob",
			manifest: `
apiVersion: batch/v1
kind: CronJob
metadata:
  name: hubble-generate-certs
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers: []
`,
			configured: configured,
			path:       []string{"spec", "jobTemplate", "spec", "template", "spec", "tolerations"},
			expected:   []any{configuredRaw},
		},
		{
			name: "nothing configured",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers: []
`,
			path:     []string{"spec", "template", "spec", "tolerations"},
			expected: nil,
		},
		{
			name: "CustomResourceDefinition is left alone",
			manifest: `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test
spec:
  names:
    kind: Deployment
`,
			configured: configured,
			path:       []string{"spec", "template", "spec", "tolerations"},
			expected:   nil,
		},
		{
			name: "custom resource sharing a workload kind is left alone",
			manifest: `
apiVersion: example.com/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec: {}
`,
			configured: configured,
			path:       []string{"spec", "template", "spec", "tolerations"},
			expected:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj := &metav1unstructured.Unstructured{}
			require.NoError(t, yaml.Unmarshal([]byte(tc.manifest), &obj.Object))

			require.NoError(t, injectTolerations(obj, tc.configured))

			tolerations, _, err := metav1unstructured.NestedSlice(obj.Object, tc.path...)
			require.NoError(t, err)
			require.Equal(t, tc.expected, tolerations)
		})
	}
}

func TestShouldReconcileClusterOnTolerations(t *testing.T) {
	oldCluster := setupTestCluster("10.240.16.0/20")

	newCluster := oldCluster.DeepCopy()
	newCluster.Spec.ComponentsOverride.UserClusterWorkloads = &kubermaticv1.UserClusterWorkloadSettings{
		Tolerations: []corev1.Toleration{{Key: "node-role", Operator: corev1.TolerationOpExists}},
	}

	unchanged, err := shouldReconcileCluster(oldCluster, oldCluster.DeepCopy())
	require.NoError(t, err)
	require.False(t, unchanged)

	changed, err := shouldReconcileCluster(oldCluster, newCluster)
	require.NoError(t, err)
	require.True(t, changed)
}
