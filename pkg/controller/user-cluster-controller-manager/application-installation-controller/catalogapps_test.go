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

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	systemToleration    = corev1.Toleration{Key: "node-role", Operator: corev1.TolerationOpEqual, Value: "system", Effect: corev1.TaintEffectNoSchedule}
	systemTolerationRaw = map[string]any{"key": "node-role", "operator": "Equal", "value": "system", "effect": "NoSchedule"}
)

func TestGenerateApplicationTolerationValues(t *testing.T) {
	configured := []corev1.Toleration{systemToleration}

	t.Run("application without a known toleration key is not covered", func(t *testing.T) {
		require.Nil(t, generateApplicationTolerationValues("kubevirt", configured, nil))
	})

	t.Run("nothing configured emits no values", func(t *testing.T) {
		require.Nil(t, generateApplicationTolerationValues("metallb", nil, nil))
	})

	t.Run("every component of a multi-component chart is covered", func(t *testing.T) {
		values := generateApplicationTolerationValues("metallb", configured, nil)

		for _, component := range []string{"controller", "speaker"} {
			tolerations, found, err := unstructured.NestedSlice(values, component, "tolerations")
			require.NoError(t, err)
			require.True(t, found, "%s is missing", component)
			require.Equal(t, []any{systemTolerationRaw}, tolerations)
		}
	})

	t.Run("deeply nested keys are built", func(t *testing.T) {
		values := generateApplicationTolerationValues("nginx", configured, nil)

		tolerations, found, err := unstructured.NestedSlice(values, "controller", "admissionWebhooks", "patch", "tolerations")
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []any{systemTolerationRaw}, tolerations)
	})

	t.Run("chart defaults are repeated so Helm does not drop them", func(t *testing.T) {
		values := generateApplicationTolerationValues("kube-vip", configured, nil)

		require.Equal(t, []any{
			map[string]any{"key": "node-role.kubernetes.io/control-plane", "operator": "Exists", "effect": "NoSchedule"},
			systemTolerationRaw,
		}, values["tolerations"])
	})

	t.Run("tolerations already configured in the values are kept", func(t *testing.T) {
		// node-exporter ships tolerations in its own defaultValuesBlock.
		current := map[string]any{"tolerations": []any{
			map[string]any{"operator": "Exists", "effect": "NoSchedule"},
			map[string]any{"operator": "Exists", "effect": "NoExecute"},
		}}

		values := generateApplicationTolerationValues("node-exporter", configured, current)

		require.Equal(t, []any{
			map[string]any{"operator": "Exists", "effect": "NoSchedule"},
			map[string]any{"operator": "Exists", "effect": "NoExecute"},
			systemTolerationRaw,
		}, values["tolerations"])
	})

	t.Run("re-running does not duplicate", func(t *testing.T) {
		first := generateApplicationTolerationValues("trivy", configured, nil)
		second := generateApplicationTolerationValues("trivy", configured, first)

		require.Equal(t, first, second)
	})
}

func TestIsAdminPushed(t *testing.T) {
	testCases := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{name: "installed by a user", expected: false},
		{name: "enforced by the admin", annotations: map[string]string{appskubermaticv1.ApplicationEnforcedAnnotation: "true"}, expected: true},
		{name: "defaulted by the admin", annotations: map[string]string{appskubermaticv1.ApplicationDefaultedAnnotation: "true"}, expected: true},
		{name: "annotation turned off again", annotations: map[string]string{appskubermaticv1.ApplicationDefaultedAnnotation: "false"}, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := &appskubermaticv1.ApplicationInstallation{ObjectMeta: metav1.ObjectMeta{Annotations: tc.annotations}}

			require.Equal(t, tc.expected, isAdminPushed(app))
		})
	}
}
