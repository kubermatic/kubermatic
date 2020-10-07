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

package gatekeeper

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	roleName               = "gatekeeper-manager-role"
	roleBindingName        = "gatekeeper-manager-rolebinding"
	clusterRoleName        = "gatekeeper-manager-role"
	clusterRoleBindingName = "gatekeeper-manager-rolebinding"
)

// ServiceAccountCreator returns a func to create/update the ServiceAccount used by gatekeeper.
func ServiceAccountCreator() (string, reconciling.ServiceAccountCreator) {
	return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		sa.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
		return sa, nil
	}
}

func RoleCreator() (string, reconciling.RoleCreator) {
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

// gatekeeperClusterRoleBindingData is the data needed to construct the Gatekeeper clusterRoleBinding
type gatekeeperClusterRoleBindingData interface {
	Cluster() *kubermaticv1.Cluster
}

func ClusterRoleBindingCreator(data gatekeeperClusterRoleBindingData) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := fmt.Sprintf("%s.%s", data.Cluster().Name, clusterRoleBindingName)
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: data.Cluster().Name,
				},
			}

			return crb, nil
		}
	}
}
