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

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	RoleName        = "nodeport-proxy"
	RoleBindingName = "nodeport-proxy"
)

func RoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
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

func RoleBindingCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
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

func ClusterRoleName(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:nodeport-proxy", cfg.Namespace)
}

func ClusterRoleCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
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

func ClusterRoleBindingName(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:nodeport-proxy", cfg.Namespace)
}

func ClusterRoleBindingCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
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

func ServiceAccountCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return ServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}
