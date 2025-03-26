/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubevirt

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// ServiceAccountsReconcilers returns the CSI serviceaccounts for KubeVirt.
func ServiceAccountsReconcilers(c *kubermaticv1.Cluster) []reconciling.NamedServiceAccountReconcilerFactory {
	creators := []reconciling.NamedServiceAccountReconcilerFactory{
		ControllerServiceAccountReconciler(c),
	}
	return creators
}

// ControllerServiceAccountReconciler returns the CSI serviceaccount for KubeVirt.
func ControllerServiceAccountReconciler(c *kubermaticv1.Cluster) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (name string, create reconciling.ServiceAccountReconciler) {
		return resources.KubeVirtCSIServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = resources.BaseAppLabels(resources.KubeVirtCSIServiceAccountName, nil)
			sa.Name = resources.KubeVirtCSIControllerName
			sa.Namespace = c.Status.NamespaceName
			return sa, nil
		}
	}
}

// ClusterRolesReconcilers returns the CSI cluster roles for KubeVirt.
func ClusterRolesReconcilers() []reconciling.NamedClusterRoleReconcilerFactory {
	creators := []reconciling.NamedClusterRoleReconcilerFactory{
		ControllerClusterRoleReconciler(),
	}
	return creators
}

// ControllerClusterRoleReconciler returns the CSI cluster role for KubeVirt.
func ControllerClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return resources.KubeVirtCSIClusterRoleName, func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Labels = resources.BaseAppLabels(resources.KubeVirtCSIClusterRoleName, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumes"},
					Verbs:     []string{"create", "delete", "get", "list", "watch", "update", "patch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumeclaims"},
					Verbs:     []string{"get", "list", "watch", "update"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumeclaims/status"},
					Verbs:     []string{"update", "patch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"storage.k8s.io"},
					Resources: []string{"volumeattachments"},
					Verbs:     []string{"get", "list", "watch", "update", "patch"},
				},
				{
					APIGroups: []string{"storage.k8s.io"},
					Resources: []string{"storageclasses"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"csi.storage.k8s.io"},
					Resources: []string{"csidrivers"},
					Verbs:     []string{"get", "list", "watch", "update", "create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"list", "watch", "create", "update", "patch"},
				},
				{
					APIGroups: []string{"snapshot.storage.k8s.io"},
					Resources: []string{"volumesnapshotclasses"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"snapshot.storage.k8s.io"},
					Resources: []string{"volumesnapshotcontents"},
					Verbs:     []string{"create", "get", "list", "watch", "update", "delete"},
				},
				{
					APIGroups: []string{"snapshot.storage.k8s.io"},
					Resources: []string{"volumesnapshots"},
					Verbs:     []string{"get", "list", "watch", "update"},
				},
				{
					APIGroups: []string{"snapshot.storage.k8s.io"},
					Resources: []string{"volumesnapshots/status"},
					Verbs:     []string{"update"},
				},
				{
					APIGroups: []string{"storage.k8s.io"},
					Resources: []string{"volumeattachments/status"},
					Verbs:     []string{"get", "list", "watch", "update", "patch"},
				},
				{
					APIGroups: []string{"storage.k8s.io"},
					Resources: []string{"csinodes"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups:     []string{"security.openshift.io"},
					Resources:     []string{"securitycontextconstraints"},
					Verbs:         []string{"use"},
					ResourceNames: []string{"privileged"},
				},
			}
			return r, nil
		}
	}
}

// RoleBindingsReconcilers returns the CSI rolebindings for KubeVirt.
func RoleBindingsReconcilers(c *kubermaticv1.Cluster) []reconciling.NamedRoleBindingReconcilerFactory {
	creators := []reconciling.NamedRoleBindingReconcilerFactory{
		ControllerRoleBindingReconciler(c),
	}
	return creators
}

// ControllerRoleBindingReconciler returns the CSI rolebinding for KubeVirt.
func ControllerRoleBindingReconciler(c *kubermaticv1.Cluster) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return resources.KubeVirtCSIRoleBindingName, func(r *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			r.Labels = resources.BaseAppLabels(resources.KubeVirtCSIClusterRoleName, nil)
			r.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      resources.KubeVirtCSIServiceAccountName,
					Namespace: c.Status.NamespaceName,
				},
			}
			r.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     resources.KubeVirtCSIControllerName,
			}

			return r, nil
		}
	}
}
