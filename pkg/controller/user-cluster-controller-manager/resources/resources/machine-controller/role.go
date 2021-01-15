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

package machinecontroller

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/machinecontroller"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSystemRoleCreator returns the func to create/update the Role for the machine controller
// to allow reading secrets/confirmaps/leases for the leaderelection
func KubeSystemRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.MachineControllerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.MachineControllerRoleName
			r.Namespace = metav1.NamespaceSystem
			r.Labels = resources.BaseAppLabels(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets", "configmaps"},
					Verbs: []string{
						"create",
						"update",
						"list",
						"watch",
					},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"endpoints"},
					ResourceNames: []string{"machine-controller"},
					Verbs:         []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"endpoints"},
					Verbs:     []string{"create"},
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

// EndpointReaderRoleCreator returns the func to create/update the Role for the machine controller to allow reading the kubernetes api endpoints
func EndpointReaderRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.MachineControllerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.MachineControllerRoleName
			r.Namespace = metav1.NamespaceDefault
			r.Labels = resources.BaseAppLabels(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"endpoints"},
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

// ClusterInfoReaderRoleCreator returns the func to create/update the Role for the machine controller to allow
// the kubelet & kubeadm to read the cluster-info reading the cluster-info ConfigMap without authentication.
func ClusterInfoReaderRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.ClusterInfoReaderRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.ClusterInfoReaderRoleName
			r.Namespace = metav1.NamespacePublic
			r.Labels = resources.BaseAppLabels(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					ResourceNames: []string{"cluster-info"},
					Resources:     []string{"configmaps"},
					Verbs:         []string{"get"},
				},
			}
			return r, nil
		}
	}
}

// KubePublicRoleCreator returns the func to create/update the Role for the machine controller to allow
// reading all configmaps in kube-public.
func KubePublicRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.MachineControllerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.MachineControllerRoleName
			r.Namespace = metav1.NamespacePublic
			r.Labels = resources.BaseAppLabels(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
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
