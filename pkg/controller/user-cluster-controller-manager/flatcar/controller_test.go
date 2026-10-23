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

package flatcar

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/flatcar/resources"
	nodelabelerapi "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-labeler/api"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileOperatorDisabled(t *testing.T) {
	flatcarNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "flatcar-node",
			Labels: map[string]string{nodelabelerapi.DistributionLabelKey: nodelabelerapi.FlatcarLabelValue},
		},
	}

	existingOperator := []ctrlruntimeclient.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: resources.OperatorDeploymentName, Namespace: operatorNamespace},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: resources.AgentDaemonSetName, Namespace: operatorNamespace},
		},
	}

	testCases := []struct {
		name             string
		operatorDisabled bool
		paused           bool
		objects          []ctrlruntimeclient.Object
		wantOperator     bool
	}{
		{
			name:         "enabled with flatcar node installs the operator",
			objects:      []ctrlruntimeclient.Object{flatcarNode},
			wantOperator: true,
		},
		{
			name:             "disabled with flatcar node removes an existing operator",
			operatorDisabled: true,
			objects:          append([]ctrlruntimeclient.Object{flatcarNode}, existingOperator...),
			wantOperator:     false,
		},
		{
			name:             "disabled with flatcar node does not install the operator",
			operatorDisabled: true,
			objects:          []ctrlruntimeclient.Object{flatcarNode},
			wantOperator:     false,
		},
		{
			name:             "disabled but paused leaves an existing operator alone",
			operatorDisabled: true,
			paused:           true,
			objects:          append([]ctrlruntimeclient.Object{flatcarNode}, existingOperator...),
			wantOperator:     true,
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(tc.objects...).Build()

			r := &Reconciler{
				Client:           client,
				updateWindow:     kubermaticv1.UpdateWindow{},
				operatorDisabled: tc.operatorDisabled,
				clusterIsPaused: func(context.Context) (bool, error) {
					return tc.paused, nil
				},
			}

			if _, err := r.Reconcile(ctx, reconcile.Request{}); err != nil {
				t.Fatalf("Reconcile returned error: %v", err)
			}

			for _, obj := range []struct {
				name string
				obj  ctrlruntimeclient.Object
			}{
				{name: resources.OperatorDeploymentName, obj: &appsv1.Deployment{}},
				{name: resources.AgentDaemonSetName, obj: &appsv1.DaemonSet{}},
			} {
				err := client.Get(ctx, types.NamespacedName{Name: obj.name, Namespace: operatorNamespace}, obj.obj)
				exists := err == nil
				if err != nil && !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get %s: %v", obj.name, err)
				}
				if exists != tc.wantOperator {
					t.Errorf("expected %s present=%t, got %t", obj.name, tc.wantOperator, exists)
				}
			}
		})
	}
}
