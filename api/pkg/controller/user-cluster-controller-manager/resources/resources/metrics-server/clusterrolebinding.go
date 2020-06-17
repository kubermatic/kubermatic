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
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingResourceReaderCreator returns the ClusterRoleBinding required for the metrics server to read all required resources
func ClusterRoleBindingResourceReaderCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return resources.MetricsServerResourceReaderClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(Name, nil)

			crb.RoleRef = rbacv1.RoleRef{
				Name:     resources.MetricsServerClusterRoleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.MetricsServerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil
		}
	}

}

// ClusterRoleBindingAuthDelegatorCreator returns the ClusterRoleBinding required for the metrics server to create token review requests
func ClusterRoleBindingAuthDelegatorCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return resources.ClusterRoleBindingAuthDelegatorCreator(resources.MetricsServerCertUsername)
}
