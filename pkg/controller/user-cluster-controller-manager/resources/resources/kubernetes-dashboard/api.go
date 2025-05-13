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
	"k8c.io/kubermatic/v2/pkg/resources"
	seedresources "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	apiAppName         = "kubernetes-dashboard-api"
	apiRoleName        = "kubernetes-dashboard-api"
	apiRoleBindingName = "kubernetes-dashboard-api"
)

// APIRoleReconciler creates the role for the Kubernetes Dashboard.
func APIRoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return apiRoleName, func(role *rbacv1.Role) (*rbacv1.Role, error) {
			role.Labels = resources.BaseAppLabels(apiAppName, nil)
			role.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"services/proxy"},
					ResourceNames: []string{msServiceName, "http:" + msServiceName},
					Verbs:         []string{"get"},
				},
			}
			return role, nil
		}
	}
}

// APIRoleBindingReconciler creates the role binding for the Kubernetes Dashboard.
func APIRoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return apiRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(apiAppName, nil)
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     apiRoleName,
			}

			rb.Subjects = []rbacv1.Subject{
				{
					APIGroup: rbacv1.GroupName,
					Kind:     "User",
					Name:     seedresources.CertUsername,
				},
			}

			return rb, nil
		}
	}
}
