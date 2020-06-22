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

package clusterautoscaler

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for the clusterautoscaler
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.ClusterAutoscalerClusterRoleName,
			func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
				cr.Rules = []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{
							"nodes",
							"pods",
							"services",
							"persistentvolumeclaims",
							"persistentvolumes",
							"replicationcontrollers",
						},
						Verbs: []string{"list", "watch", "get"},
					},
					{
						// Needed to taint nodes prior to scaling down
						APIGroups: []string{""},
						Resources: []string{"nodes"},
						Verbs:     []string{"update"},
					},
					{
						APIGroups: []string{"apps", "extensions"},
						Resources: []string{
							"deployments",
							"replicasets",
							"daemonsets",
							"statefulsets",
						},
						Verbs: []string{"list", "watch", "get"},
					},
					{
						APIGroups: []string{"policy"},
						Resources: []string{"poddisruptionbudgets"},
						Verbs:     []string{"list", "watch", "get"},
					},
					{
						APIGroups: []string{"storage.k8s.io"},
						Resources: []string{"storageclasses"},
						Verbs:     []string{"list", "watch", "get"},
					},
					{
						APIGroups: []string{"batch"},
						Resources: []string{"jobs", "cronjobs"},
						Verbs:     []string{"list", "watch", "get"},
					},
					{
						APIGroups: []string{"cluster.k8s.io"},
						Resources: []string{"machinedeployments", "machinesets", "machines"},
						Verbs:     []string{"list", "watch", "get"},
					},
					{
						// Needed on MachineDeployments to change replicaCount and on
						// machines to set the scale down annotation
						APIGroups: []string{"cluster.k8s.io"},
						Resources: []string{"machinedeployments", "machines"},
						Verbs:     []string{"update"},
					},
				}
				return cr, nil
			}
	}
}

// ClusterRoleBinding returns a ClusterRoleBinding for clusterautoscaler
func ClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return resources.ClusterAutoscalerClusterRoleBindingName,
			func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
				crb.RoleRef = rbacv1.RoleRef{
					Name:     resources.ClusterAutoscalerClusterRoleName,
					Kind:     "ClusterRole",
					APIGroup: rbacv1.GroupName,
				}
				crb.Subjects = []rbacv1.Subject{{
					Kind:     "User",
					Name:     resources.ClusterAutoscalerCertUsername,
					APIGroup: rbacv1.GroupName,
				}}
				return crb, nil
			}
	}
}

func KubeSystemRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.ClusterAutoscalerClusterRoleName,
			func(r *rbacv1.Role) (*rbacv1.Role, error) {
				r.Rules = []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create", "patch"},
					},
					{
						APIGroups:     []string{""},
						Resources:     []string{"configmaps"},
						ResourceNames: []string{"cluster-autoscaler-status", "cluster-autoscaler"},
						Verbs:         []string{"get", "update", "delete"},
					},
				}
				return r, nil
			}
	}
}

func KubeSystemRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.ClusterAutoscalerClusterRoleBindingName,
			func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
				rb.RoleRef = rbacv1.RoleRef{
					Name:     resources.ClusterAutoscalerClusterRoleName,
					Kind:     "Role",
					APIGroup: rbacv1.GroupName,
				}
				rb.Subjects = []rbacv1.Subject{{
					Kind:     "User",
					Name:     resources.ClusterAutoscalerCertUsername,
					APIGroup: rbacv1.GroupName,
				}}

				return rb, nil
			}
	}
}

func DefaultRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.ClusterAutoscalerClusterRoleName,
			func(r *rbacv1.Role) (*rbacv1.Role, error) {
				r.Rules = []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"patch", "create"},
					},
				}
				return r, nil
			}
	}
}

func DefaultRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.ClusterAutoscalerClusterRoleBindingName,
			func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
				rb.RoleRef = rbacv1.RoleRef{
					Name:     resources.ClusterAutoscalerClusterRoleName,
					Kind:     "Role",
					APIGroup: rbacv1.GroupName,
				}
				rb.Subjects = []rbacv1.Subject{{
					Kind:     "User",
					Name:     resources.ClusterAutoscalerCertUsername,
					APIGroup: rbacv1.GroupName,
				}}

				return rb, nil
			}
	}
}
