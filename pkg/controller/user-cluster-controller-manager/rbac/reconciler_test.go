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

package rbacusercluster

import (
	"context"
	"sort"
	"sync"
	"testing"

	"go.uber.org/zap"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	owners  = "system:kubermatic:owners"
	editors = "system:kubermatic:editors"
	viewers = "system:kubermatic:viewers"
)

func TestReconcileCreate(t *testing.T) {
	tests := []struct {
		name                      string
		resourceNames             []string
		expectedRoleNumber        int
		expectedRoleBindingNumber int
		expectedRoles             []rbacv1.ClusterRole
		expectedRoleBindings      []rbacv1.ClusterRoleBinding
	}{
		{
			name:                      "scenario 1: test Reconcile method for creating system:kubermatic:owners",
			resourceNames:             []string{owners},
			expectedRoleNumber:        1,
			expectedRoleBindingNumber: 1,
			expectedRoles:             []rbacv1.ClusterRole{genTestClusterRole(t, owners)},
			expectedRoleBindings:      []rbacv1.ClusterRoleBinding{genTestClusterRoleBinding(t, owners)},
		},
		{
			name:                      "scenario 1: test Reconcile method for all types",
			resourceNames:             []string{owners, editors, viewers},
			expectedRoleNumber:        3,
			expectedRoleBindingNumber: 3,
			expectedRoles:             []rbacv1.ClusterRole{genTestClusterRole(t, owners), genTestClusterRole(t, editors), genTestClusterRole(t, viewers)},
			expectedRoleBindings:      []rbacv1.ClusterRoleBinding{genTestClusterRoleBinding(t, owners), genTestClusterRoleBinding(t, editors), genTestClusterRoleBinding(t, viewers)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			r := reconciler{
				Client: fakeClient,
				logger: zap.NewNop().Sugar(),
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
				rLock: &sync.Mutex{},
			}

			// create scenario
			for _, name := range test.resourceNames {
				_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: name}})
				if err != nil {
					t.Fatalf("Reconcile error: %v", err)
				}
			}

			roles := &rbacv1.ClusterRoleList{}
			if err := r.List(ctx, roles); err != nil {
				t.Fatalf("getting cluster roles error: %v", err)
			}

			if len(roles.Items) != test.expectedRoleNumber {
				t.Fatalf("incorrect number of cluster roles were returned, got: %d, want: %d, ", len(roles.Items), test.expectedRoleNumber)
			}

			sortClusterRoles(roles.Items)
			sortClusterRoles(test.expectedRoles)

			if !equality.Semantic.DeepEqual(roles.Items, test.expectedRoles) {
				t.Fatalf("incorrect roles were returned, got: %v, want: %v", roles.Items, test.expectedRoles)
			}

			rolesBindings := &rbacv1.ClusterRoleBindingList{}
			if err := r.List(ctx, rolesBindings); err != nil {
				t.Fatalf("getting cluster role bindings error: %v", err)
			}

			if len(rolesBindings.Items) != test.expectedRoleBindingNumber {
				t.Fatalf("incorrect number of cluster role bindings were returned, got: %d, want: %d, ", len(rolesBindings.Items), test.expectedRoleBindingNumber)
			}

			sortClusterRoleBindings(rolesBindings.Items)
			sortClusterRoleBindings(test.expectedRoleBindings)

			if !equality.Semantic.DeepEqual(rolesBindings.Items, test.expectedRoleBindings) {
				t.Fatalf("incorrect roles were returned, got: %v, want: %v", rolesBindings.Items, test.expectedRoleBindings)
			}
		})
	}
}

func sortClusterRoles(roles []rbacv1.ClusterRole) {
	sort.SliceStable(roles, func(i, j int) bool {
		mi, mj := roles[i], roles[j]
		if mi.Namespace == mj.Namespace {
			return mi.Name < mj.Name
		}
		return mi.Namespace < mj.Namespace
	})
}

func sortClusterRoleBindings(roles []rbacv1.ClusterRoleBinding) {
	sort.SliceStable(roles, func(i, j int) bool {
		mi, mj := roles[i], roles[j]
		if mi.Namespace == mj.Namespace {
			return mi.Name < mj.Name
		}
		return mi.Namespace < mj.Namespace
	})
}

func TestReconcileUpdateRole(t *testing.T) {
	tests := []struct {
		name         string
		roleName     string
		updatedRole  *rbacv1.ClusterRole
		expectedRole rbacv1.ClusterRole
	}{
		{
			name:     "scenario 1: test Reconcile method when cluster role system:kubermatic:editors was changed",
			roleName: editors,

			updatedRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: editors,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"node"},
						Verbs:     []string{"test"},
					},
				},
			},
			expectedRole: genTestClusterRole(t, editors),
		},
		{
			name:     "scenario 2: test Reconcile method when cluster role system:kubermatic:viewers was changed",
			roleName: viewers,
			updatedRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: viewers,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"test"},
						Resources: []string{"test"},
						Verbs:     []string{"test"},
					},
				},
			},
			expectedRole: genTestClusterRole(t, viewers),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			r := reconciler{
				Client: fake.NewClientBuilder().Build(),
				logger: zap.NewNop().Sugar(),
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
				rLock: &sync.Mutex{},
			}

			if err := r.Create(ctx, test.updatedRole); err != nil {
				t.Fatalf("failed to create ClusterRole: %v", err)
			}

			// check for updates
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: test.roleName}})
			if err != nil {
				t.Fatalf("Reconcile method error: %v", err)
			}

			role := &rbacv1.ClusterRole{}
			if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: test.roleName}, role); err != nil {
				t.Fatalf("can't find cluster role %v", err)
			}

			// compare roles
			if !equality.Semantic.DeepEqual(role.Rules, test.expectedRole.Rules) {
				t.Fatalf("incorrect cluster role rules were returned, got: %v, want: %v", role.Rules, test.expectedRole.Rules)
			}

			if !equality.Semantic.DeepEqual(role.OwnerReferences, test.expectedRole.OwnerReferences) {
				t.Fatalf("incorrect cluster role OwnerReferences were returned, got: %v, want: %v", role.OwnerReferences, test.expectedRole.OwnerReferences)
			}
		})
	}
}

func genTestClusterRole(t *testing.T, resourceName string) rbacv1.ClusterRole {
	creatorGetter, err := newClusterRoleReconciler(resourceName)
	if err != nil {
		t.Fatalf("failed to create ClusterRole getter: %v", err)
	}

	_, creator := creatorGetter()

	clusterRole, err := creator(&rbacv1.ClusterRole{})
	if err != nil {
		t.Fatalf("failed to run ClusterRole creator: %v", err)
	}

	clusterRole.Name = resourceName
	clusterRole.ResourceVersion = "1"

	return *clusterRole
}

func genTestClusterRoleBinding(t *testing.T, resourceName string) rbacv1.ClusterRoleBinding {
	creatorGetter, err := newClusterRoleBindingReconciler(resourceName)
	if err != nil {
		t.Fatalf("failed to create ClusterRoleBinding getter: %v", err)
	}

	_, creator := creatorGetter()

	clusterRoleBinding, err := creator(&rbacv1.ClusterRoleBinding{})
	if err != nil {
		t.Fatalf("failed to run ClusterRoleBinding creator: %v", err)
	}

	clusterRoleBinding.Name = resourceName
	clusterRoleBinding.ResourceVersion = "1"

	return *clusterRoleBinding
}
