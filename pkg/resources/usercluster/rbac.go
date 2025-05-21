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

package usercluster

import (
	"fmt"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ServiceAccountName = "kubermatic-usercluster-controller-manager"
	roleName           = "kubermatic:usercluster-controller-manager"
	roleBindingName    = "kubermatic:usercluster-controller-manager"
)

func ServiceAccountReconciler() (string, reconciling.ServiceAccountReconciler) {
	return ServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func RoleReconciler() (string, reconciling.RoleReconciler) {
	return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
				},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				ResourceNames: []string{
					resources.AdminKubeconfigSecretName,
				},
				Verbs: []string{"update"},
			},
			{
				APIGroups: []string{"kubermatic.k8c.io"},
				Resources: []string{"constraints"},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"patch",
					"update",
				},
			},
			{
				APIGroups: []string{"kubermatic.k8c.io"},
				Resources: []string{"addons"},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"delete",
				},
			},
		}
		return r, nil
	}
}

func RoleBindingReconciler() (string, reconciling.RoleBindingReconciler) {
	return roleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
		rb.RoleRef = rbacv1.RoleRef{
			Name:     roleName,
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: ServiceAccountName,
			},
		}
		return rb, nil
	}
}

func ClusterRole() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return roleName, func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"clusters", "clusters/status"},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"patch",
						"update",
					},
				},
				// This should be removed with KKP 2.23 when we remove the Migration for OSM.
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"get", "list", "watch"},
				},
				// This should be removed with KKP 2.23 when we remove the Migration for OSM.
				{
					APIGroups: []string{"operatingsystemmanager.k8c.io"},
					Resources: []string{"operatingsystemprofiles", "operatingsystemconfigs"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{appskubermaticv1.GroupName},
					Resources: []string{appskubermaticv1.ApplicationDefinitionResourceName},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"policytemplates"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"kubermatic.k8c.io"},
					Resources: []string{"policybindings"},
					Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
				},
			}
			return r, nil
		}
	}
}

func ClusterRoleBinding(namespace *corev1.Namespace) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return GenClusterRoleBindingName(namespace), func(rb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			rb.OwnerReferences = []metav1.OwnerReference{genOwnerReference(namespace)}
			rb.RoleRef = rbacv1.RoleRef{
				Name:     roleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      ServiceAccountName,
					Namespace: namespace.Name,
				},
			}
			return rb, nil
		}
	}
}

// genOwnerReference returns an owner ref pointing to the cluster namespace. This
// ensures that when a cluster is deleted, the ClusterRole/Binding are deleted automatically
// *after* the cluster namespace is gone. Previously, we manually deleted them, but
// this could lead to cases where the usercluster-ctrl-mgr is still running and
// producing errors because it cannot access Cluster objects anymore.
func genOwnerReference(namespace *corev1.Namespace) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: namespace.APIVersion,
		Kind:       namespace.Kind,
		Name:       namespace.Name,
		UID:        namespace.UID,
	}
}

func GenClusterRoleBindingName(targetNamespace *corev1.Namespace) string {
	return fmt.Sprintf("%s-%s", roleBindingName, targetNamespace.Name)
}
