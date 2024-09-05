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

package resources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	operatorClusterRoleName = "flatcar-linux-update-operator"
	agentClusterRoleName    = "flatcar-linux-update-agent"
)

func OperatorClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return operatorClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"update",
					},
				},
			}
			return cr, nil
		}
	}
}

func AgentClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return agentClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"update",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs: []string{
						"get",
						"list",
						"delete",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods/eviction"},
					Verbs: []string{
						"create",
					},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"daemonsets"},
					Verbs: []string{
						"get",
					},
				},
			}
			return cr, nil
		}
	}
}
