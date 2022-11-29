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

// DefaultRoleBindingCreator returns the func to create/update the RoleBinding for the machine-controller.
func DefaultRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	// RoleBindingDataProvider actually not needed, no ownerrefs set in user-cluster
	return RoleBindingCreator()
}

// KubeSystemRoleBinding returns the RoleBinding for the machine-controller in kube-system ns.
func KubeSystemRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return RoleBindingCreator()
}

// KubePublicRoleBinding returns the RoleBinding for the machine-controller in kube-public ns.
func KubePublicRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return RoleBindingCreator()
}

func RoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.MachineControllerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(machinecontroller.Name, nil)

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.MachineControllerRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					Name:     resources.MachineControllerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}

// ClusterInfoAnonymousRoleBindingCreator returns a func to create/update the RoleBinding to allow anonymous access to the cluster-info ConfigMap.
func ClusterInfoAnonymousRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.ClusterInfoAnonymousRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Namespace = metav1.NamespacePublic

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.ClusterInfoReaderRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					APIGroup: rbacv1.GroupName,
					Kind:     rbacv1.UserKind,
					Name:     "system:anonymous",
				},
			}
			return rb, nil
		}
	}
}
