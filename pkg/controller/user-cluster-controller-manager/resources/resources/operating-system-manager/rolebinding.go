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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSystemRoleBindingReconciler returns the RoleBinding for the OSM in kube-system ns.
func KubeSystemRoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return RoleBindingReconciler()
}

func RoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return resources.OperatingSystemManagerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(resources.OperatingSystemManagerDeploymentName, nil)

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

// KubePublicRoleBindingReconciler returns the RoleBinding for the OSM in kube-system ns.
func KubePublicRoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
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

// DefaultRoleBindingReconciler returns the RoleBinding for the OSM in kube-system ns.
func DefaultRoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
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

func CloudInitSettingsRoleBindingReconciler() reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
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

func ClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
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
