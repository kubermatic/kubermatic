/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package groupprojectbinding

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                        string
		bindingName                 string
		groupName                   string
		roleName                    string
		projectName                 string
		existingResources           []ctrlruntimeclient.Object
		expectedClusterRoleBindings []rbacv1.ClusterRoleBinding
		expectedRoleBindings        []rbacv1.RoleBinding
	}{
		{
			name:        "bind group to global ClusterRole for editors",
			bindingName: "group-project-binding",
			groupName:   "external-group",
			roleName:    "editors",
			projectName: "test",
			existingResources: []ctrlruntimeclient.Object{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeys:editors",
						Labels: map[string]string{
							"authz.k8c.io/role": "editors",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"kubermatic.k8c.io"},
							Resources: []string{"usersshkeys"},
							Verbs:     []string{"create"},
						},
					},
				},
			},
			expectedClusterRoleBindings: []rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkeys:editors:group-project-binding",
						Labels: map[string]string{
							"authz.k8c.io/group-project-binding": "group-project-binding",
							"authz.k8c.io/role":                  "editors",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8c.io/v1",
								Kind:       "GroupProjectBinding",
								Name:       "group-project-binding",
							},
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkeys:editors",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "external-group",
						},
					},
				},
			},
			expectedRoleBindings: []rbacv1.RoleBinding{},
		},
		{
			name:        "bind group to Role in kubermatic namespace for owners",
			bindingName: "group-project-binding",
			groupName:   "external-group",
			roleName:    "owners",
			projectName: "test",
			existingResources: []ctrlruntimeclient.Object{
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners",
						Namespace: "kubermatic",
						Labels: map[string]string{
							"authz.k8c.io/role": "owners",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"screts"},
							Verbs:     []string{"create"},
						},
					},
				},
			},
			expectedClusterRoleBindings: []rbacv1.ClusterRoleBinding{},
			expectedRoleBindings: []rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secrets:owners:group-project-binding",
						Namespace: "kubermatic",
						Labels: map[string]string{
							"authz.k8c.io/group-project-binding": "group-project-binding",
							"authz.k8c.io/role":                  "owners",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8c.io/v1",
								Kind:       "GroupProjectBinding",
								Name:       "group-project-binding",
							},
						},
					},
					RoleRef: rbacv1.RoleRef{

						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "kubermatic:secrets:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "external-group",
						},
					},
				},
			},
		},
		{
			name:        "bind group to ClusterRoles and Roles for a specific project",
			bindingName: "group-project-binding",
			groupName:   "external-group",
			roleName:    "owners",
			projectName: "test",
			existingResources: []ctrlruntimeclient.Object{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-test:owners-test",
						Labels: map[string]string{
							"authz.k8c.io/role": "owners-test",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{"kubermatic.k8c.io"},
							ResourceNames: []string{"test"},
							Resources:     []string{"projects"},
							Verbs:         []string{"get", "update", "patch", "delete"},
						},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:addons:owners",
						Namespace: "cluster-fake",
						Labels: map[string]string{
							"authz.k8c.io/role": "owners-test",
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"kubermatic.k8c.io"},
							Resources: []string{"addons"},
							Verbs:     []string{"get", "list", "create", "update", "delete"},
						},
					},
				},
			},
			expectedClusterRoleBindings: []rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-test:owners-test:group-project-binding",
						Labels: map[string]string{
							"authz.k8c.io/group-project-binding": "group-project-binding",
							"authz.k8c.io/role":                  "owners",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8c.io/v1",
								Kind:       "GroupProjectBinding",
								Name:       "group-project-binding",
							},
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-test:owners-test",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "external-group",
						},
					},
				},
			},
			expectedRoleBindings: []rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:addons:owners:group-project-binding",
						Namespace: "cluster-fake",
						Labels: map[string]string{
							"authz.k8c.io/group-project-binding": "group-project-binding",
							"authz.k8c.io/role":                  "owners",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8c.io/v1",
								Kind:       "GroupProjectBinding",
								Name:       "group-project-binding",
							},
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "kubermatic:addons:owners",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "external-group",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			project := generateProject(tc.projectName)
			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingResources...).
				WithObjects(project).
				WithObjects(genGroupProjectBinding(tc.bindingName, tc.groupName, tc.roleName, tc.projectName)).
				Build()

			r := &Reconciler{
				log:      kubermaticlog.Logger,
				recorder: &record.FakeRecorder{},
				Client:   client,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.bindingName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			for _, expectedClusterRoleBinding := range tc.expectedClusterRoleBindings {
				clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
				if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: expectedClusterRoleBinding.Name}, clusterRoleBinding); err != nil {
					t.Fatalf("did not find expected ClusterRoleBinding: %v", err)
				}

				if !equality.Semantic.DeepEqual(clusterRoleBinding.OwnerReferences, expectedClusterRoleBinding.OwnerReferences) ||
					!equality.Semantic.DeepEqual(clusterRoleBinding.Labels, expectedClusterRoleBinding.Labels) ||
					!equality.Semantic.DeepEqual(clusterRoleBinding.RoleRef, expectedClusterRoleBinding.RoleRef) ||
					!equality.Semantic.DeepEqual(clusterRoleBinding.Subjects, expectedClusterRoleBinding.Subjects) {
					t.Fatalf(
						"ClusterRoleBinding does not match expected resource, diff: %s",
						diff.ObjectGoPrintSideBySide(clusterRoleBinding, expectedClusterRoleBinding),
					)
				}
			}

			for _, expectedRoleBinding := range tc.expectedRoleBindings {
				roleBinding := &rbacv1.RoleBinding{}
				if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: expectedRoleBinding.Name, Namespace: expectedRoleBinding.Namespace}, roleBinding); err != nil {
					t.Fatalf("did not find expected RoleBinding: %v", err)
				}

				if !equality.Semantic.DeepEqual(roleBinding.OwnerReferences, expectedRoleBinding.OwnerReferences) ||
					!equality.Semantic.DeepEqual(roleBinding.Labels, expectedRoleBinding.Labels) ||
					!equality.Semantic.DeepEqual(roleBinding.RoleRef, expectedRoleBinding.RoleRef) ||
					!equality.Semantic.DeepEqual(roleBinding.Subjects, expectedRoleBinding.Subjects) {
					t.Fatalf(
						"RoleBinding does not match expected resource, diff: %s",
						diff.ObjectGoPrintSideBySide(roleBinding, expectedRoleBinding),
					)
				}
			}
		})
	}
}

func generateProject(name string) *kubermaticv1.Project {
	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: fmt.Sprintf("project-%s", name),
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: kubermaticv1.ProjectActive,
		},
	}
	return project
}

func genGroupProjectBinding(bindingName, groupName, roleName, projectID string) *kubermaticv1.GroupProjectBinding {
	binding := &kubermaticv1.GroupProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
		Spec: kubermaticv1.GroupProjectBindingSpec{
			Group:     groupName,
			Role:      roleName,
			ProjectID: projectID,
		},
	}

	return binding
}
