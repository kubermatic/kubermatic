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

package gatekeeper

import (
	rbacv1 "k8s.io/api/rbac/v1"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

const clusterRoleName = "gatekeeper-manager-role"

func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return clusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
				{
					APIGroups: []string{"config.gatekeeper.sh"},
					Resources: []string{"configs"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
				{
					APIGroups: []string{"config.gatekeeper.sh"},
					Resources: []string{"configs/status"},
					Verbs:     []string{"get", "patch", "update"},
				},
				{
					APIGroups: []string{"constraints.gatekeeper.sh"},
					Resources: []string{"*"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
				{
					APIGroups: []string{"templates.gatekeeper.sh"},
					Resources: []string{"constrainttemplates"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
				{
					APIGroups: []string{"templates.gatekeeper.sh"},
					Resources: []string{"constrainttemplates/status"},
					Verbs:     []string{"get", "patch", "update"},
				},
				{
					APIGroups:     []string{"admissionregistration.k8s.io"},
					ResourceNames: []string{"gatekeeper-validating-webhook-configuration"},
					Resources:     []string{"validatingwebhookconfigurations"},
					Verbs:         []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
			}

			return cr, nil
		}
	}
}
