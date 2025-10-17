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

package cloudcontroller

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterRoleBindingReconciler returns a func to create/update the ClusterRoleBinding for external cloud controllers.
func ClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return resources.CloudControllerManagerRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(resources.CloudControllerManagerRoleBindingName, nil)

			crb.RoleRef = rbacv1.RoleRef{
				// Can probably be tightened up a bit but for now I'm following the documentation.
				// https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/
				Name:     "cluster-admin",
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.CloudControllerManagerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      resources.CloudControllerManagerServiceAccountName,
					Namespace: metav1.NamespaceSystem,
				},
			}
			return crb, nil
		}
	}
}
