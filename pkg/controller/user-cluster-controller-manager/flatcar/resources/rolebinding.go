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

package resources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	operatorRoleBindingName = "flatcar-linux-update-operator"
	agentRoleBindingName    = "flatcar-linux-update-agent"
)

func OperatorRoleBindingReconciler(operatorNamespace string) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return operatorRoleBindingName, func(crb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     operatorRoleName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      operatorServiceAccountName,
					Namespace: operatorNamespace,
				},
			}
			return crb, nil
		}
	}
}

// This RoleBinding is defined upstream, but points to a non-existing Role.
// cf. https://github.com/flatcar/flatcar-linux-update-operator/pull/163
// It seems harmless so we stay in-sync with upstream rather than deciding
// ourselves that the RoleBinding is unnecessary.

func AgentRoleBindingReconciler(operatorNamespace string) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return agentRoleBindingName, func(crb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     "flatcar-linux-update-agent",
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      agentServiceAccountName,
					Namespace: operatorNamespace,
				},
			}
			return crb, nil
		}
	}
}
