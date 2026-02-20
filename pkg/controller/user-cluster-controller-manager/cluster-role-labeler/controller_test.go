/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package clusterrolelabeler

import (
	"context"
	"testing"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name           string
		clusterRole    *rbacv1.ClusterRole
		requestName    string
		expectedLabels map[string]string
	}{
		{
			name: "cluster role not found, no error",
		},
		{
			name: "label added to view cluster role",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name: "view",
			}},
			requestName:    "view",
			expectedLabels: map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
		},
		{
			name: "label added to edit cluster role",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name: "edit",
			}},
			requestName:    "edit",
			expectedLabels: map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
		},
		{
			name: "label added to admin cluster role",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name: "admin",
			}},
			requestName:    "admin",
			expectedLabels: map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			clientBuilder := fake.NewClientBuilder()
			if tc.clusterRole != nil {
				clientBuilder.WithObjects(tc.clusterRole)
			}

			client := clientBuilder.Build()
			ctx := context.Background()
			r := &reconciler{
				log:      kubermaticlog.Logger,
				client:   client,
				recorder: events.NewFakeRecorder(10),
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			if tc.clusterRole == nil {
				return
			}

			clusterRole := &rbacv1.ClusterRole{}
			if err := client.Get(ctx, request.NamespacedName, clusterRole); err != nil {
				t.Fatalf("failed to get cluster role: %v", err)
			}

			if d := diff.ObjectDiff(tc.expectedLabels, clusterRole.Labels); d != "" {
				t.Errorf("cluster role doesn't have expected labels:\n%v", d)
			}
		})
	}
}
