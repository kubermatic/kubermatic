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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRevisionHistoryLimitSetsSupportedObjects(t *testing.T) {
	t.Parallel()

	const desired int32 = 13

	testCases := []struct {
		name     string
		resource ctrlruntimeclient.Object
		actual   func(ctrlruntimeclient.Object) *int32
	}{
		{
			name:     "deployment",
			resource: &appsv1.Deployment{},
			actual: func(obj ctrlruntimeclient.Object) *int32 {
				return obj.(*appsv1.Deployment).Spec.RevisionHistoryLimit
			},
		},
		{
			name:     "statefulset",
			resource: &appsv1.StatefulSet{},
			actual: func(obj ctrlruntimeclient.Object) *int32 {
				return obj.(*appsv1.StatefulSet).Spec.RevisionHistoryLimit
			},
		},
		{
			name:     "daemonset",
			resource: &appsv1.DaemonSet{},
			actual: func(obj ctrlruntimeclient.Object) *int32 {
				return obj.(*appsv1.DaemonSet).Spec.RevisionHistoryLimit
			},
		},
	}

	baseReconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return existing, nil
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler := RevisionHistoryLimit(desired)(baseReconciler)
			obj, err := reconciler(tc.resource)
			require.NoError(t, err)

			limit := tc.actual(obj)
			require.NotNil(t, limit, "expected revision history limit to be set")
			require.Equal(t, desired, *limit, "revision history limit does not match desired value")
		})
	}
}

func TestRevisionHistoryLimitPanicsForUnsupportedType(t *testing.T) {
	t.Parallel()

	const desired int32 = 7

	baseReconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return existing, nil
	}

	reconciler := RevisionHistoryLimit(desired)(baseReconciler)
	unsupported := &corev1.ConfigMap{}

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered)

		msg := fmt.Sprintf("%v", recovered)
		require.Contains(t, msg, "RevisionHistoryLimit modifier used on incompatible type *v1.ConfigMap", "panic message")
	}()

	_, _ = reconciler(unsupported)
}

func TestRevisionHistoryLimitPropagatesReconcilerError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("fake error")
	obj := &appsv1.Deployment{}

	baseReconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return existing, expectedErr
	}

	reconciler := RevisionHistoryLimit(9)(baseReconciler)
	returnedObj, err := reconciler(obj)
	require.ErrorIs(t, expectedErr, err, "expected error to be propagated")
	require.Equal(t, obj, returnedObj, "expected original object to be returned on error")
	require.Nil(t, obj.Spec.RevisionHistoryLimit, "expected revision history limit to remain unchanged on error")
}
