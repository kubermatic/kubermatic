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

package modifier

import (
	"testing"

	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	tolerateEverything = corev1.Toleration{Operator: corev1.TolerationOpExists}
	tolerateSystem     = corev1.Toleration{Key: "node-role", Operator: corev1.TolerationOpEqual, Value: "system", Effect: corev1.TaintEffectNoSchedule}
	tolerateGPU        = corev1.Toleration{Key: "gpu", Operator: corev1.TolerationOpExists}
)

func podTolerations(t *testing.T, obj ctrlruntimeclient.Object) []corev1.Toleration {
	t.Helper()

	switch asserted := obj.(type) {
	case *appsv1.Deployment:
		return asserted.Spec.Template.Spec.Tolerations
	case *appsv1.StatefulSet:
		return asserted.Spec.Template.Spec.Tolerations
	case *appsv1.DaemonSet:
		return asserted.Spec.Template.Spec.Tolerations
	}

	t.Fatalf("unexpected type %T", obj)

	return nil
}

func TestTolerationsSetsSupportedObjects(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		resource ctrlruntimeclient.Object
	}{
		{name: "deployment", resource: &appsv1.Deployment{}},
		{name: "statefulset", resource: &appsv1.StatefulSet{}},
		{name: "daemonset", resource: &appsv1.DaemonSet{}},
	}

	baseReconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return existing, nil
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obj, err := Tolerations([]corev1.Toleration{tolerateSystem})(baseReconciler)(tc.resource)
			require.NoError(t, err)
			require.Equal(t, []corev1.Toleration{tolerateSystem}, podTolerations(t, obj))
			require.Contains(t, obj.GetAnnotations(), AppliedTolerationsAnnotation)
		})
	}
}

func TestTolerationsPanicsOnUnsupportedObject(t *testing.T) {
	t.Parallel()

	reconciler := Tolerations(nil)(func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return &corev1.ConfigMap{}, nil
	})

	require.Panics(t, func() {
		_, _ = reconciler(&corev1.ConfigMap{})
	})
}

func TestTolerationsAcrossReconciles(t *testing.T) {
	t.Parallel()

	// keepingReconciler behaves like most usercluster reconcilers, which never assign tolerations.
	keepingReconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return existing, nil
	}

	// settingReconciler behaves like the reconcilers which assign their tolerations on every run.
	settingReconciler := func(tolerations ...corev1.Toleration) func(ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			existing.(*appsv1.Deployment).Spec.Template.Spec.Tolerations = tolerations
			return existing, nil
		}
	}

	testCases := []struct {
		name       string
		reconciler func(ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error)
		existing   []corev1.Toleration
		runs       [][]corev1.Toleration
		expected   []corev1.Toleration
		annotated  bool
	}{
		{
			name:       "nothing configured leaves the object alone",
			reconciler: keepingReconciler,
			existing:   []corev1.Toleration{tolerateGPU},
			runs:       [][]corev1.Toleration{nil},
			expected:   []corev1.Toleration{tolerateGPU},
		},
		{
			name:       "repeated reconciles do not duplicate",
			reconciler: keepingReconciler,
			runs:       [][]corev1.Toleration{{tolerateSystem}, {tolerateSystem}},
			expected:   []corev1.Toleration{tolerateSystem},
			annotated:  true,
		},
		{
			name:       "changed configuration replaces the applied toleration",
			reconciler: keepingReconciler,
			runs:       [][]corev1.Toleration{{tolerateSystem}, {tolerateGPU}},
			expected:   []corev1.Toleration{tolerateGPU},
			annotated:  true,
		},
		{
			name:       "cleared configuration removes the applied toleration",
			reconciler: keepingReconciler,
			existing:   []corev1.Toleration{tolerateGPU},
			runs:       [][]corev1.Toleration{{tolerateSystem}, nil},
			expected:   []corev1.Toleration{tolerateGPU},
		},
		{
			name:       "tolerations of the reconciler are kept",
			reconciler: settingReconciler(tolerateGPU),
			runs:       [][]corev1.Toleration{{tolerateSystem}, {tolerateSystem}},
			expected:   []corev1.Toleration{tolerateGPU, tolerateSystem},
			annotated:  true,
		},
		{
			name:       "toleration already set by the reconciler is not recorded as applied",
			reconciler: settingReconciler(tolerateEverything, tolerateSystem),
			runs:       [][]corev1.Toleration{{tolerateSystem}, nil},
			expected:   []corev1.Toleration{tolerateEverything, tolerateSystem},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var obj ctrlruntimeclient.Object = &appsv1.Deployment{}
			obj.(*appsv1.Deployment).Spec.Template.Spec.Tolerations = tc.existing

			for _, extra := range tc.runs {
				var err error

				obj, err = Tolerations(extra)(tc.reconciler)(obj)
				require.NoError(t, err)
			}

			require.Equal(t, tc.expected, podTolerations(t, obj))

			_, annotated := obj.GetAnnotations()[AppliedTolerationsAnnotation]
			require.Equal(t, tc.annotated, annotated)
		})
	}
}
