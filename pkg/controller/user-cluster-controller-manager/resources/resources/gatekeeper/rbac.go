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

package gatekeeper

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	roleName        = "gatekeeper-manager-role"
	roleBindingName = "gatekeeper-manager-rolebinding"
)

// ServiceAccountCreator returns a func to create/update the ServiceAccount used by gatekeeper.
func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			return sa, nil
		}
	}
}

// RoleCreator creates the gatekeeper Role
func RoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
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
			}
			return r, nil
		}
	}
}

// RoleBindingCreator creates the gatekeeper RoleBinding
func RoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return roleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				Name:     roleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind: rbacv1.ServiceAccountKind,
					Name: serviceAccountName,
				},
			}
			return rb, nil
		}
	}
}

// ClusterRoleCreator creates the gatekeeper ClusterRole
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return roleName, func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
					},
				},
				{
					APIGroups: []string{"config.gatekeeper.sh"},
					Resources: []string{"configs"},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
					},
				},
				{
					APIGroups: []string{"config.gatekeeper.sh"},
					Resources: []string{"configs/status"},
					Verbs: []string{
						"get",
						"patch",
						"update",
					},
				},
				{
					APIGroups: []string{"constraints.gatekeeper.sh"},
					Resources: []string{"*"},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
					},
				},
				{
					APIGroups: []string{"policy"},
					Resources: []string{"podsecuritypolicies"},
					Verbs:     []string{"use"},
				},
				{
					APIGroups: []string{"status.gatekeeper.sh"},
					Resources: []string{"*"},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
					},
				},
				{
					APIGroups: []string{"templates.gatekeeper.sh"},
					Resources: []string{"constrainttemplates"},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
					},
				},
				{
					APIGroups: []string{"templates.gatekeeper.sh"},
					Resources: []string{"constrainttemplates/finalizers"},
					Verbs: []string{
						"delete",
						"get",
						"patch",
						"update",
					},
				},
				{
					APIGroups: []string{"templates.gatekeeper.sh"},
					Resources: []string{"constrainttemplates/status"},
					Verbs: []string{
						"get",
						"patch",
						"update",
					},
				},
				{
					APIGroups:     []string{"admissionregistration.k8s.io"},
					Resources:     []string{"validatingwebhookconfigurations"},
					ResourceNames: []string{resources.GatekeeperValidatingWebhookConfigurationName},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
					},
				},
			}
			return r, nil
		}
	}
}

// ClusterRoleBindingCreator creates the gatekeeper ClusterRoleBinding
func ClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return roleBindingName, func(rb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				Name:     roleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: resources.GatekeeperNamespace,
				},
			}
			return rb, nil
		}
	}
}
