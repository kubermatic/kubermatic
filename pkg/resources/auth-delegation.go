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

package resources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBindingAuthenticationReaderReconciler returns a function to create the RoleBinding which is needed for extension apiserver which do auth delegation.
func RoleBindingAuthenticationReaderReconciler(username string) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return username + "-authentication-reader", func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				Name:     "extension-apiserver-authentication-reader",
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     username,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}

// ClusterRoleBindingAuthDelegatorReconciler returns a function to create the ClusterRoleBinding which is needed for extension apiserver which do auth delegation.
func ClusterRoleBindingAuthDelegatorReconciler(username string) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return username + "-auth-delegator", func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				Name:     "system:auth-delegator",
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     username,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil
		}
	}
}
