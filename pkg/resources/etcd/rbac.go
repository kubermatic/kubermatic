/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package etcd

import (
	"fmt"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	ServiceAccountName = "etcd-launcher"
	roleName           = "etcd-launcher"
	roleBindingName    = "etcd-launcher"
)

// ServiceAccountReconciler returns a func to create/update the ServiceAccount used by etcd launcher.
func ServiceAccountReconciler() (string, reconciling.ServiceAccountReconciler) {
	return ServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func clusterRoleName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("kubermatic:%s:etcd-launcher", cluster.Name)
}

func clusterRoleBindingName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("kubermatic:%s:etcd-launcher", cluster.Name)
}

func RoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = resources.BaseAppLabels(name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "secrets"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"statefulsets"},
					Verbs:     []string{"get", "list"},
				},
			}
			return r, nil
		}
	}
}

func RoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return roleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(name, nil)
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     roleName,
			}
			rb.Subjects = []rbacv1.Subject{{
				Kind: "ServiceAccount",
				Name: ServiceAccountName,
			}}
			return rb, nil
		}
	}
}

func ClusterRoleReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleName(cluster), func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabels(name, nil)

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"kubermatic.k8c.io"},
					Resources:     []string{"clusters"},
					Verbs:         []string{"get"},
					ResourceNames: []string{cluster.Name},
				},
			}

			return cr, nil
		}
	}
}

func ClusterRoleBindingReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleBindingName(cluster), func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(name, nil)
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterRoleName(cluster),
			}
			crb.Subjects = []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      ServiceAccountName,
				Namespace: cluster.Status.NamespaceName,
			}}
			return crb, nil
		}
	}
}
