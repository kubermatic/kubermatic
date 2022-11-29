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

// KubeSystemRoleBindingCreator returns the RoleBinding for the OSM in kube-system ns.
func KubeSystemRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return RoleBindingCreator()
}

func RoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.OperatingSystemManagerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(operatingsystemmanager.Name, nil)

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.OperatingSystemManagerRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					Name:     resources.OperatingSystemManagerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}

// KubePublicRoleBindingCreator returns the RoleBinding for the OSM in kube-system ns.
func KubePublicRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.OperatingSystemManagerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Namespace = metav1.NamespacePublic

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.OperatingSystemManagerRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					Name:     resources.OperatingSystemManagerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}

// DefaultRoleBindingCreator returns the RoleBinding for the OSM in kube-system ns.
func DefaultRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.OperatingSystemManagerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Namespace = metav1.NamespaceDefault

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.OperatingSystemManagerRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					Name:     resources.OperatingSystemManagerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}

func CloudInitSettingsRoleBindingCreator() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.OperatingSystemManagerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.OperatingSystemManagerRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					Name:     resources.OperatingSystemManagerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}

func MachineDeploymentsClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return resources.OperatingSystemManagerClusterRoleBindingName,
			func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
				crb.RoleRef = rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "ClusterRole",
					Name:     resources.OperatingSystemManagerClusterRoleName,
				}
				crb.Subjects = []rbacv1.Subject{
					{
						Kind:     rbacv1.UserKind,
						Name:     resources.OperatingSystemManagerCertUsername,
						APIGroup: rbacv1.GroupName,
					},
				}
				return crb, nil
			}
	}
}
