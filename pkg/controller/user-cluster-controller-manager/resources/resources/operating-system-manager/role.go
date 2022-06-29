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

package operatingsystemmanager

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/operatingsystemmanager"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSystemRoleCreator returns the func to create/update the Role for the OSM
// to facilitate leaderelection.
func KubeSystemRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.OperatingSystemManagerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.OperatingSystemManagerRoleName
			r.Namespace = metav1.NamespaceSystem
			r.Labels = resources.BaseAppLabels(operatingsystemmanager.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs: []string{
						"create",
						"update",
						"get",
						"list",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs: []string{
						"create",
						"patch",
					},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"*"},
				},
			}
			return r, nil
		}
	}
}

// KubePublicRoleCreator returns the func to create/update the Role for the OSM
// to facilitate leaderelection.
func KubePublicRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.OperatingSystemManagerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.OperatingSystemManagerRoleName
			r.Namespace = metav1.NamespacePublic
			r.Labels = resources.BaseAppLabels(operatingsystemmanager.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"configmaps"},
					ResourceNames: []string{"cluster-info"},
					Verbs: []string{
						"get",
					},
				},
			}
			return r, nil
		}
	}
}

// DefaultRoleCreator returns the func to create/update the Role for the OSM
// to facilitate leaderelection.
func DefaultRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.OperatingSystemManagerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.OperatingSystemManagerRoleName
			r.Namespace = metav1.NamespaceDefault
			r.Labels = resources.BaseAppLabels(operatingsystemmanager.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"endpoints"},
					ResourceNames: []string{"kubernetes"},
					Verbs: []string{
						"get",
					},
				},
			}
			return r, nil
		}
	}
}

func CloudInitSettingsRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.OperatingSystemManagerRoleName,
			func(r *rbacv1.Role) (*rbacv1.Role, error) {
				r.Rules = []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs: []string{
							"get",
							"list",
							"create",
							"delete",
						},
					},
				}
				return r, nil
			}
	}
}

func MachineDeploymentsClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.OperatingSystemManagerClusterRoleName,
			func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
				r.Rules = []rbacv1.PolicyRule{
					{
						APIGroups: []string{"cluster.k8s.io"},
						Resources: []string{"machinedeployments"},
						Verbs: []string{
							"get",
							"list",
							"watch",
							"patch",
							"update",
						},
					},
				}
				return r, nil
			}
	}
}
