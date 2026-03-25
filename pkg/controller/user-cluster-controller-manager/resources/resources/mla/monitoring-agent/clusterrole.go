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

package monitoringagent

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return resources.MLAMonitoringAgentClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabels(appName, nil)

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"endpoints",
						"nodes",
						"nodes/proxy",
						"node/metrics",
						"pods",
						"services",
						"ingresses",
						"configmaps",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{"discovery.k8s.io"},
					Resources: []string{"endpointslices"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{
						"extensions",
						"networking.k8s.io",
					},
					Resources: []string{
						"ingresses",
						"ingresses/status",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					NonResourceURLs: []string{
						"/metrics",
					},
					Verbs: []string{
						"get",
					},
				},
			}
			return cr, nil
		}
	}
}
