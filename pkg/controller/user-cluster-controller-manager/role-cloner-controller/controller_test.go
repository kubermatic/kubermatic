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

package roleclonercontroller

import (
	"context"
	"reflect"
	"sort"
	"testing"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var nowTime metav1.Time

func TestReconcile(t *testing.T) {
	// Enable debug logging
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	nowTime = metav1.Now()
	testCases := []struct {
		name             string
		objects          []ctrlruntimeclient.Object
		expectedRoles    []rbacv1.Role
		requestName      string
		requestNamespace string
	}{
		{
			name: "role not found, no error",
		},
		{
			name:          "delete role view for all namespaces",
			expectedRoles: []rbacv1.Role{},
			objects: []ctrlruntimeclient.Object{
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "view",
						Namespace:         "kube-system",
						Finalizers:        []string{cleanupFinalizer},
						Labels:            map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
						DeletionTimestamp: &nowTime,
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
			requestName:      "view",
			requestNamespace: "kube-system",
		},
		{
			name: "clone role for all namespaces",
			expectedRoles: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{cleanupFinalizer},
						Labels:     map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
				},
			},
			objects: []ctrlruntimeclient.Object{
				&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{
					Name:      "view",
					Namespace: "kube-system",
					Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
				}},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
			requestName:      "view",
			requestNamespace: "kube-system",
		},
		{
			name: "update role view for all namespaces",
			expectedRoles: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{cleanupFinalizer},
						Labels:     map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
			},
			objects: []ctrlruntimeclient.Object{
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{cleanupFinalizer},
						Labels:     map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
			requestName:      "view",
			requestNamespace: "kube-system",
		},
		{
			name: "do not clone into deleted namespace",
			expectedRoles: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{cleanupFinalizer},
						Labels:     map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
			},
			objects: []ctrlruntimeclient.Object{
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{cleanupFinalizer},
						Labels:     map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test",
						DeletionTimestamp: nowPtr(),
						Finalizers:        []string{"dummy"},
					},
				},
			},
			requestName:      "view",
			requestNamespace: "kube-system",
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			clientBuilder := fake.NewClientBuilder()
			if tc.expectedRoles != nil {
				clientBuilder.WithObjects(tc.objects...)
			}

			r := &reconciler{
				log:      kubermaticlog.Logger,
				client:   clientBuilder.Build(),
				recorder: events.NewFakeRecorder(10),
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
			}

			ctx := context.Background()
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName, Namespace: tc.requestNamespace}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			if tc.expectedRoles == nil {
				return
			}

			existingRoleList := &rbacv1.RoleList{}
			if err := r.client.List(ctx, existingRoleList, ctrlruntimeclient.MatchingLabels{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue}); err != nil {
				t.Fatalf("failed to get role: %v", err)
			}

			existingRoles := existingRoleList.Items
			if len(existingRoles) != len(tc.expectedRoles) {
				t.Fatalf("roles are not equal, expected length %d got %d", len(tc.expectedRoles), len(existingRoles))
			}

			var newExistingRoles []rbacv1.Role
			// get rid of time format differences
			for _, role := range existingRoles {
				role.TypeMeta = metav1.TypeMeta{}
				role.ResourceVersion = ""
				role.DeletionTimestamp = nil
				newExistingRoles = append(newExistingRoles, role)
			}
			sortRoles(newExistingRoles)
			sortRoles(tc.expectedRoles)

			if len(tc.expectedRoles) > 0 {
				if !reflect.DeepEqual(newExistingRoles, tc.expectedRoles) {
					t.Fatalf("roles are not equal, expected %v got %v", tc.expectedRoles, newExistingRoles)
				}
			} else {
				if len(newExistingRoles) != 0 {
					t.Fatalf("expected 0 roles, but found %d", len(newExistingRoles))
				}
			}
		})
	}
}

func sortRoles(roles []rbacv1.Role) {
	sort.SliceStable(roles, func(i, j int) bool {
		mi, mj := roles[i], roles[j]
		if mi.Namespace == mj.Namespace {
			return mi.Name < mj.Name
		}
		return mi.Namespace < mj.Namespace
	})
}

func nowPtr() *metav1.Time {
	now := metav1.Now()
	return &now
}
