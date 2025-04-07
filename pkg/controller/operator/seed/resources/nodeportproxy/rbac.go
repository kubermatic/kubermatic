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

package nodeportproxy

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	RoleName        = "nodeport-proxy"
	RoleBindingName = "nodeport-proxy"
)

func RoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return RoleName, func(cr *rbacv1.Role) (*rbacv1.Role, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"services"},
					Verbs:         []string{"update"},
					ResourceNames: []string{ServiceName},
				},
			}

			return cr, nil
		}
	}
}

func RoleBindingReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return RoleBindingName, func(crb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     RoleName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Namespace: cfg.Namespace,
					Name:      ServiceAccountName,
				},
			}

			return crb, nil
		}
	}
}

func ClusterRoleName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:nodeport-proxy", cfg.Namespace)
}

func ClusterRoleReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return ClusterRoleName(cfg), func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"endpoints", "services"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return cr, nil
		}
	}
}

func ClusterRoleBindingName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:nodeport-proxy", cfg.Namespace)
}

func ClusterRoleBindingReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return ClusterRoleBindingName(cfg), func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     ClusterRoleName(cfg),
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Namespace: cfg.Namespace,
					Name:      ServiceAccountName,
				},
			}

			return crb, nil
		}
	}
}

func ServiceAccountReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return ServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}
