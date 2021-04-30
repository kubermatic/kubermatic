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

package rbac

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac/test"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	k8scorev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestSyncProjectResourcesClusterWide(t *testing.T) {
	tests := []struct {
		name                        string
		dependantToSync             ctrlruntimeclient.Object
		expectedClusterRoles        []*rbacv1.ClusterRole
		existingClusterRoles        []*rbacv1.ClusterRole
		expectedClusterRoleBindings []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindings []*rbacv1.ClusterRoleBinding
		expectedActions             []string
		expectError                 bool
	}{
		// scenario 1
		{
			name:            "scenario 1: a proper set of RBAC Role/Binding is generated for a cluster",
			expectedActions: []string{"create", "create", "create", "create", "create", "create", "get", "create", "get", "create", "get", "create", "get", "create", "get", "create", "get", "create", "create", "get", "create", "get", "create", "get", "create", "get", "create"},

			dependantToSync: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					UID:  types.UID("abcdID"),
					Labels: map[string]string{
						kubermaticv1.ProjectIDLabelKey: "thunderball",
					},
				},
				Spec:    kubermaticv1.ClusterSpec{},
				Address: kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-abcd",
				},
			},

			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get"},
						},
					},
				},
			},

			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:cluster-abcd:owners-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:cluster-abcd:editors-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:cluster-abcd:viewers-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:etcd-launcher",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							Namespace: "cluster-abcd",
							Kind:      "ServiceAccount",
							Name:      "etcd-launcher",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:cluster-abcd:viewers-thunderball",
					},
				},
			},
		},

		// scenario 2
		{
			name:            "scenario 2: a proper set of RBAC Role/Binding is generated for an ssh key",
			expectedActions: []string{"create", "create", "create", "create", "create", "create"},

			dependantToSync: &kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					UID:  types.UID("abcdID"),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ProjectKindName,
							Name:       "thunderball",
							UID:        "thunderballID",
						},
					},
				},
				Spec: kubermaticv1.SSHKeySpec{},
			},

			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.SSHKeyResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.SSHKeyResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.SSHKeyResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get"},
						},
					},
				},
			},

			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkey-abcd:owners-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkey-abcd:editors-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkey-abcd:viewers-thunderball",
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: a proper set of RBAC Role/Binding is generated for a userprojectbinding resource",
			expectedActions: []string{"create", "create"},

			dependantToSync: &kubermaticv1.UserProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					UID:  types.UID("abcdID"),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ProjectKindName,
							Name:       "thunderball",
							UID:        "thunderballID",
						},
					},
					ResourceVersion: "1",
				},
				Spec: kubermaticv1.UserProjectBindingSpec{
					UserEmail: "bob@acme.com",
					ProjectID: "thunderball",
					Group:     "owners-thunderball",
				},
			},

			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:userprojectbinding-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.UserProjectBindingKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.UserProjectBindingResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
			},

			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:userprojectbinding-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.UserProjectBindingKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:userprojectbinding-abcd:owners-thunderball",
					},
				},
			},
		},

		// scenario 4
		{
			name:        "scenario 4 an error is returned when syncing a cluster that doesn't belong to a project",
			expectError: true,
			dependantToSync: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					UID:  types.UID("abcdID"),
				},
				Spec:    kubermaticv1.ClusterSpec{},
				Address: kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-abcd",
				},
			},
		},

		// scenario 5
		{
			name:            "scenario 5: a proper set of RBAC Role/Binding is generated for an external cluster",
			expectedActions: []string{"create", "create", "create", "create", "create", "create"},

			dependantToSync: &kubermaticv1.ExternalCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					UID:  types.UID("abcdID"),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.ProjectKindName,
							Name:       "thunderball",
							UID:        "thunderballID",
						},
					},
				},
				Spec: kubermaticv1.ExternalClusterSpec{},
			},

			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:externalcluster-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ExternalClusterKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ExternalClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:externalcluster-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ExternalClusterKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ExternalClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:externalcluster-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ExternalClusterKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ExternalClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get"},
						},
					},
				},
			},

			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:externalcluster-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ExternalClusterKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:externalcluster-abcd:owners-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:externalcluster-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ExternalClusterKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:externalcluster-abcd:editors-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:externalcluster-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ExternalClusterKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:externalcluster-abcd:viewers-thunderball",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()

			objs := []ctrlruntimeclient.Object{test.dependantToSync}
			for _, existingClusterRole := range test.existingClusterRoles {
				objs = append(objs, existingClusterRole)
			}

			for _, existingClusterRoleBinding := range test.existingClusterRoleBindings {
				objs = append(objs, existingClusterRoleBinding)
			}

			fakeMasterClusterClient := fakectrlruntimeclient.NewClientBuilder().WithObjects(objs...).Build()

			// act
			target := resourcesController{
				client:     fakeMasterClusterClient,
				restMapper: getFakeRestMapper(t),
				objectType: test.dependantToSync.DeepCopyObject().(ctrlruntimeclient.Object),
			}
			objmeta, err := meta.Accessor(test.dependantToSync)
			assert.NoError(t, err)
			_, err = target.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: objmeta.GetNamespace(),
				Name:      objmeta.GetName(),
			}})

			// validate
			if err != nil && !test.expectError {
				t.Fatal(err)
			}
			if test.expectError && err == nil {
				t.Fatal("expected an error but got nothing")
			}
			if test.expectError {
				return
			}

			{
				var clusterRoleBindings rbacv1.ClusterRoleBindingList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoleBindings)
				assert.NoError(t, err)

				assert.Len(t, clusterRoleBindings.Items, len(test.expectedClusterRoleBindings),
					"cluster contains an different number of ClusterRoleBindings than expected (%d != %d)", len(clusterRoleBindings.Items), len(test.expectedClusterRoleBindings))

			expectedClusterRoleBindingsLoop:
				for _, expectedClusterRoleBinding := range test.expectedClusterRoleBindings {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					for _, existingClusterRoleBinding := range clusterRoleBindings.Items {
						if reflect.DeepEqual(*expectedClusterRoleBinding, existingClusterRoleBinding) {
							continue expectedClusterRoleBindingsLoop
						}
					}
					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedClusterRoleBinding.Name)
				}
			}

			{
				var clusterRoles rbacv1.ClusterRoleList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoles)
				assert.NoError(t, err)

				assert.Len(t, clusterRoles.Items, len(test.expectedClusterRoles),
					"cluster contains an different number of ClusterRoles than expected (%d != %d)", len(clusterRoles.Items), len(test.expectedClusterRoles))

			expectedClusterRolesLoop:
				for _, expectedClusterRole := range test.expectedClusterRoles {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range clusterRoles.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedClusterRolesLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}
			}
		})
	}
}

func TestSyncProjectResourcesNamespaced(t *testing.T) {
	tests := []struct {
		name                 string
		dependantToSync      ctrlruntimeclient.Object
		expectedRoles        []*rbacv1.Role
		existingRoles        []*rbacv1.Role
		expectedRoleBindings []*rbacv1.RoleBinding
		existingRoleBindings []*rbacv1.RoleBinding
		expectedActions      []string
		expectError          bool
	}{
		// scenario 1
		{
			name:            "scenario 1: a proper set of RBAC Role/Binding is generated for secrets in kubermatic namespace",
			expectedActions: []string{"create", "create"},

			dependantToSync: &k8scorev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "abcd",
					Namespace: "kubermatic",
					UID:       types.UID("abcdID"),
					Labels: map[string]string{
						kubermaticv1.ProjectIDLabelKey: "thunderball",
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"token": {0xFF, 0xFF},
				},
			},

			expectedRoles: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secret-abcd:owners-thunderball",
						Namespace: "kubermatic",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: k8scorev1.SchemeGroupVersion.String(),
								Kind:       "Secret",
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{k8scorev1.SchemeGroupVersion.Group},
							Resources:     []string{"secrets"},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
			},

			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secret-abcd:owners-thunderball",
						Namespace: "kubermatic",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: k8scorev1.SchemeGroupVersion.String(),
								Kind:       "Secret",
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secret-abcd:owners-thunderball",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()

			objs := []ctrlruntimeclient.Object{test.dependantToSync}
			for _, existingRole := range test.existingRoles {
				objs = append(objs, existingRole)
			}

			for _, existingRoleBinding := range test.existingRoleBindings {
				objs = append(objs, existingRoleBinding)
			}

			fakeMasterClusterClient := fakectrlruntimeclient.NewClientBuilder().WithObjects(objs...).Build()
			// act
			target := resourcesController{
				client:     fakeMasterClusterClient,
				restMapper: getFakeRestMapper(t),
				objectType: test.dependantToSync.DeepCopyObject().(ctrlruntimeclient.Object),
			}
			objmeta, err := meta.Accessor(test.dependantToSync)
			assert.NoError(t, err)
			_, err = target.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: objmeta.GetNamespace(),
				Name:      objmeta.GetName(),
			}})

			// validate
			if !test.expectError {
				assert.NoError(t, err)
			}
			if test.expectError {
				assert.Error(t, err)
				return
			}

			{
				var roles rbacv1.RoleList
				err = fakeMasterClusterClient.List(context.Background(), &roles)
				assert.NoError(t, err)

				assert.Len(t, roles.Items, len(test.expectedRoles),
					"cluster contains an different number of Roles than expected (%d != %d)", len(roles.Items), len(test.expectedRoles))

			expectedRolesLoop:
				for _, expectedRole := range test.expectedRoles {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingRole := range roles.Items {
						if reflect.DeepEqual(*expectedRole, existingRole) {
							continue expectedRolesLoop
						}
					}
					t.Fatalf("expected RoleBinding %q not found in cluster", expectedRole.Name)
				}
			}

			{
				var roleBindings rbacv1.RoleBindingList
				err = fakeMasterClusterClient.List(context.Background(), &roleBindings)
				assert.NoError(t, err)

				assert.Len(t, roleBindings.Items, len(test.expectedRoleBindings),
					"cluster contains an different number of RoleBindings than expected (%d != %d)", len(roleBindings.Items), len(test.expectedRoleBindings))

			expectedRoleBindingsLoop:
				for _, expectedRoleBinding := range test.expectedRoleBindings {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingRoleBinding := range roleBindings.Items {
						if reflect.DeepEqual(*expectedRoleBinding, existingRoleBinding) {
							continue expectedRoleBindingsLoop
						}
					}
					t.Fatalf("expected RoleBinding %q not found in cluster", expectedRoleBinding.Name)
				}
			}
		})
	}
}

func TestEnsureProjectClusterRBACRoleBindingForNamedResource(t *testing.T) {
	tests := []struct {
		name                        string
		projectToSync               *kubermaticv1.Project
		expectedClusterRoleBindings []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindings []*rbacv1.ClusterRoleBinding
		expectedActions             []string
	}{
		// scenario 1
		{
			name:            "scenario 1: desired RBAC Role Bindings for a project resource are created",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"create", "create", "create"},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Role Bindings exist",
			projectToSync: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: update when existing binding doesn't match desired ones",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"update", "update", "update"},
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "wrong-subject-name",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "wrong-subject-name",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "2",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "2",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []ctrlruntimeclient.Object{}
			for _, existingClusterRoleBinding := range test.existingClusterRoleBindings {
				objs = append(objs, existingClusterRoleBinding)
			}
			fakeMasterClusterClient := fakectrlruntimeclient.NewClientBuilder().WithObjects(objs...).Build()

			// act
			err := ensureClusterRBACRoleBindingForNamedResource(context.Background(), fakeMasterClusterClient, test.projectToSync.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta())
			assert.NoError(t, err)

			{
				var clusterRoleBindings rbacv1.ClusterRoleBindingList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoleBindings)
				assert.NoError(t, err)

				assert.Len(t, clusterRoleBindings.Items, len(test.expectedClusterRoleBindings),
					"cluster contains an different number of ClusterRoleBindings than expected (%d != %d)", len(clusterRoleBindings.Items), len(test.expectedClusterRoleBindings))

			expectedClusterRoleBindingsLoop:
				for _, expectedClusterRoleBinding := range test.expectedClusterRoleBindings {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRoleBinding := range clusterRoleBindings.Items {
						if reflect.DeepEqual(*expectedClusterRoleBinding, existingClusterRoleBinding) {
							continue expectedClusterRoleBindingsLoop
						}
					}
					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedClusterRoleBinding.Name)
				}
			}
		})
	}
}

func TestEnsureProjectClusterRBACRoleForNamedResource(t *testing.T) {
	tests := []struct {
		name                 string
		projectToSync        *kubermaticv1.Project
		expectedClusterRoles []*rbacv1.ClusterRole
		existingClusterRoles []*rbacv1.ClusterRole
		expectedActions      []string
	}{
		// scenario 1
		{
			name:            "scenario 1: desired RBAC Roles for a project resource are created",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"create", "create", "create"},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Roles exist",
			projectToSync: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: update when desired are not the same as expected RBAC Roles",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"update", "update"},
			existingClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "2",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
						ResourceVersion: "1",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []ctrlruntimeclient.Object{}
			for _, existingClusterRole := range test.existingClusterRoles {
				objs = append(objs, existingClusterRole)
			}
			fakeMasterClusterClient := fakectrlruntimeclient.NewClientBuilder().WithObjects(objs...).Build()

			// act
			err := ensureClusterRBACRoleForNamedResource(context.Background(), fakeMasterClusterClient, test.projectToSync.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta())
			assert.NoError(t, err)

			{
				var clusterRoles rbacv1.ClusterRoleList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoles)
				assert.NoError(t, err)

				assert.Len(t, clusterRoles.Items, len(test.expectedClusterRoles),
					"cluster contains an different number of ClusterRole than expected (%d != %d)", len(clusterRoles.Items), len(test.expectedClusterRoles))

			expectedClusterRolesLoop:
				for _, expectedClusterRole := range test.expectedClusterRoles {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.
					expectedClusterRole.ResourceVersion = ""

					for _, existingClusterRole := range clusterRoles.Items {
						existingClusterRole.ResourceVersion = ""
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedClusterRolesLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}
			}
		})
	}
}

func TestSyncClusterConstraintsRBAC(t *testing.T) {
	tests := []struct {
		name                 string
		dependantToSync      ctrlruntimeclient.Object
		expectedRoles        []*rbacv1.Role
		existingRoles        []*rbacv1.Role
		expectedRoleBindings []*rbacv1.RoleBinding
		existingRoleBindings []*rbacv1.RoleBinding
		expectError          bool
	}{
		// scenario 1
		{
			name: "scenario 1: a proper set of RBAC Role/Binding is generated for constraints",

			dependantToSync: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "clusterid",
					Labels: map[string]string{"project-id": "my-first-project"},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-clusterid",
				},
			},

			expectedRoles: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:constraint:owners",
						Namespace: "cluster-clusterid",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{kubermaticv1.ConstraintResourceName},
							Verbs:     []string{"get", "list", "create", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:constraint:editors",
						Namespace: "cluster-clusterid",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{kubermaticv1.ConstraintResourceName},
							Verbs:     []string{"get", "list", "create", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:constraint:viewers",
						Namespace: "cluster-clusterid",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{kubermaticv1.ConstraintResourceName},
							Verbs:     []string{"get", "list"},
						},
					},
				},
			},

			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:constraint:owners",
						Namespace:       "cluster-clusterid",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-my-first-project",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:constraint:owners",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:constraint:editors",
						Namespace:       "cluster-clusterid",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-my-first-project",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:constraint:editors",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:constraint:viewers",
						Namespace:       "cluster-clusterid",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-my-first-project",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:constraint:viewers",
					},
				},
			},
		},
		// scenario 2
		{
			name: "scenario 2: a mis-configured set of RBAC Role/Binding is updated for constraints",

			dependantToSync: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "clusterid",
					Labels: map[string]string{"project-id": "my-first-project"},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-clusterid",
				},
			},

			expectedRoles: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:constraint:owners",
						Namespace: "cluster-clusterid",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{kubermaticv1.ConstraintResourceName},
							Verbs:     []string{"get", "list", "create", "update", "delete"},
						},
					},
				},
			},

			existingRoles: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:constraint:owners",
						Namespace: "cluster-clusterid",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources: []string{kubermaticv1.ConstraintResourceName},
							Verbs:     []string{"get", "update", "delete"},
						},
					},
				},
			},

			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:constraint:owners",
						Namespace:       "cluster-clusterid",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-my-first-project",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:constraint:owners",
					},
				},
			},
			existingRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "kubermatic:constraint:owners",
						Namespace:       "cluster-clusterid",
						ResourceVersion: "1",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-my-first-project",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:constraint:owners",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()

			objs := []ctrlruntimeclient.Object{test.dependantToSync}
			for _, existingRole := range test.existingRoles {
				objs = append(objs, existingRole)
			}

			for _, existingRoleBinding := range test.existingRoleBindings {
				objs = append(objs, existingRoleBinding)
			}

			fakeMasterClusterClient := fakectrlruntimeclient.NewClientBuilder().WithObjects(objs...).Build()
			// act
			target := resourcesController{
				client:     fakeMasterClusterClient,
				restMapper: getFakeRestMapper(t),
				objectType: test.dependantToSync.DeepCopyObject().(ctrlruntimeclient.Object),
			}
			objmeta, err := meta.Accessor(test.dependantToSync)
			assert.NoError(t, err)
			_, err = target.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: objmeta.GetNamespace(),
				Name:      objmeta.GetName(),
			}})

			// validate
			if !test.expectError {
				assert.NoError(t, err)
			}
			if test.expectError {
				assert.Error(t, err)
				return
			}

			{
				var roles rbacv1.RoleList
				err = fakeMasterClusterClient.List(context.Background(), &roles)
				assert.NoError(t, err)

			expectedRolesLoop:
				for _, expectedRole := range test.expectedRoles {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingRole := range roles.Items {
						if strings.EqualFold(expectedRole.Name, existingRole.Name) && reflect.DeepEqual(expectedRole.Rules, existingRole.Rules) {
							continue expectedRolesLoop

						}
					}
					t.Fatalf("expected Role %q not found in cluster", expectedRole.Name)
				}
			}

			{
				var roleBindings rbacv1.RoleBindingList
				err = fakeMasterClusterClient.List(context.Background(), &roleBindings)
				assert.NoError(t, err)

			expectedRoleBindingsLoop:
				for _, expectedRoleBinding := range test.expectedRoleBindings {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingRoleBinding := range roleBindings.Items {
						if strings.EqualFold(expectedRoleBinding.Name, existingRoleBinding.Name) &&
							reflect.DeepEqual(expectedRoleBinding.RoleRef, existingRoleBinding.RoleRef) &&
							reflect.DeepEqual(expectedRoleBinding.Subjects, existingRoleBinding.Subjects) {
							continue expectedRoleBindingsLoop
						}
					}
					t.Fatalf("expected RoleBinding %q not found in cluster", expectedRoleBinding.Name)
				}
			}
		})
	}
}
