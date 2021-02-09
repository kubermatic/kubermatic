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

package usercluster

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

const (
	serviceAccountName = "kubermatic-usercluster-controller-manager"
	roleName           = "kubermatic:usercluster-controller-manager"
	roleBindingName    = "kubermatic:usercluster-controller-manager"
)

func ServiceAccountCreator() (string, reconciling.ServiceAccountCreator) {
	return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func RoleCreator() (string, reconciling.RoleCreator) {
	return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
				},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				ResourceNames: []string{
					resources.AdminKubeconfigSecretName,
				},
				Verbs: []string{"update"},
			},
			{
				APIGroups: []string{"kubermatic.k8s.io"},
				Resources: []string{"constraints"},
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

func RoleBindingCreator() (string, reconciling.RoleBindingCreator) {
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
