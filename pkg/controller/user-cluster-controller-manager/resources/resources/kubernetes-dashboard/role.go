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

package kubernetesdashboard

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleCreator creates the role for the Kubernetes Dashboard
func RoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.KubernetesDashboardRoleName, func(role *rbacv1.Role) (*rbacv1.Role, error) {
			role.Labels = resources.BaseAppLabels(AppName, nil)
			role.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"secrets"},
					ResourceNames: []string{"kubernetes-dashboard-key-holder", "kubernetes-dashboard-csrf"},
					Verbs:         []string{"get", "update", "delete"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"configmaps"},
					ResourceNames: []string{"kubernetes-dashboard-settings"},
					Verbs:         []string{"get", "update"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"services"},
					ResourceNames: []string{"heapster", "dashboard-metrics-scraper"},
					Verbs:         []string{"proxy"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"services/proxy"},
					ResourceNames: []string{"heapster", "http:heapster:", "https:heapster:", "dashboard-metrics-scraper", "http:dashboard-metrics-scraper"},
					Verbs:         []string{"get"},
				},
			}
			return role, nil
		}
	}
}
