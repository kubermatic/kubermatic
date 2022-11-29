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

package metricsserver

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RolebindingAuthReaderCreator returns a func to create/update the RoleBinding used by the metrics-server to get access to the token subject review API.
func RolebindingAuthReaderCreator(isKonnectivityEnabled bool) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.MetricsServerAuthReaderRoleName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(Name, nil)

			rb.RoleRef = rbacv1.RoleRef{
				Name:     "extension-apiserver-authentication-reader",
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}

			if isKonnectivityEnabled {
				// metrics server running in the user cluster - ServiceAccount
				rb.Subjects = []rbacv1.Subject{
					{
						Kind:      rbacv1.ServiceAccountKind,
						Name:      resources.MetricsServerServiceAccountName,
						Namespace: metav1.NamespaceSystem,
					},
				}
			} else {
				// metrics server running in the seed cluster - User
				rb.Subjects = []rbacv1.Subject{
					{
						Kind:     rbacv1.UserKind,
						Name:     resources.MetricsServerCertUsername,
						APIGroup: rbacv1.GroupName,
					},
				}
			}

			return rb, nil
		}
	}
}
