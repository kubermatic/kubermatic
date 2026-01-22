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
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// KubeSystemRoleReconciler returns the func to create/update the Role for the OSM
// to retrieve kube-apiserver address from the cluster-info configmap.
func KubeSystemRoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return resources.OperatingSystemManagerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = resources.BaseAppLabels(resources.OperatingSystemManagerDeploymentName, nil)

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
					Resources: []string{"secrets"},
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

// KubePublicRoleReconciler returns the func to create/update the Role for the OSM
// to facilitate leaderelection.
func KubePublicRoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return resources.OperatingSystemManagerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = resources.BaseAppLabels(resources.OperatingSystemManagerDeploymentName, nil)

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

// DefaultRoleReconciler returns the func to create/update the Role for the OSM
// to retrieve kube-apiserver address from the Kubernetes EndpointSlices.
func DefaultRoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return resources.OperatingSystemManagerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = resources.BaseAppLabels(resources.OperatingSystemManagerDeploymentName, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"discovery.k8s.io"},
					Resources: []string{"endpointslices"},
					Verbs: []string{
						"get",
						"list",
					},
				},
			}
			return r, nil
		}
	}
}

func CloudInitSettingsRoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
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
							"update",
						},
					},
				}
				return r, nil
			}
	}
}

func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return resources.OperatingSystemManagerClusterRoleName,
			func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
				r.Rules = []rbacv1.PolicyRule{
					{
						APIGroups: []string{"operatingsystemmanager.k8c.io"},
						Resources: []string{"operatingsystemprofiles", "operatingsystemconfigs"},
						Verbs:     []string{"*"},
					},
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
					{
						APIGroups: []string{"apps"},
						Resources: []string{"deployments"},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets", "configmaps"},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
				}
				return r, nil
			}
	}
}
