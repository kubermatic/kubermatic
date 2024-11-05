/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubernetesdashboard

import (
	seedresources "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	webSettingsConfigMapName = "kubernetes-dashboard-web-settings"
	webRoleName              = "kubernetes-dashboard-web"
	webRoleBindingName       = "kubernetes-dashboard-web"
)

func WebConfigMapReconciler() reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return webSettingsConfigMapName, func(existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			return existing, nil
		}
	}
}

func WebRoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return webRoleName, func(existing *rbacv1.Role) (*rbacv1.Role, error) {
			existing.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"configmaps"},
					ResourceNames: []string{webSettingsConfigMapName},
					Verbs:         []string{"get", "update"},
				},
			}

			return existing, nil
		}
	}
}

func WebRoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return webRoleBindingName, func(existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			existing.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     webRoleName,
			}

			existing.Subjects = []rbacv1.Subject{
				{
					APIGroup: rbacv1.GroupName,
					Kind:     "User",
					Name:     seedresources.CertUsername,
				},
			}

			return existing, nil
		}
	}
}
