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

package machinecontroller

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	Name = "machine-controller"
)

// ClusterRole returns a cluster role for the machine controller (user-cluster)
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.MachineControllerClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabels(Name, nil)

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups:     []string{"apiextensions.k8s.io"},
					Resources:     []string{"customresourcedefinitions"},
					ResourceNames: []string{"machines.machine.k8s.io"},
					Verbs:         []string{"*"},
				},
				{
					APIGroups: []string{"machine.k8s.io"},
					Resources: []string{"machines"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumes", "secrets", "configmaps"},
					Verbs:     []string{"list", "get", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods/eviction"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"create", "patch"},
				},
				{
					APIGroups: []string{"cluster.k8s.io"},
					Resources: []string{"machines", "machines/finalizers",
						"machinesets", "machinesets/status", "machinesets/finalizers",
						"machinedeployments", "machinedeployments/status", "machinedeployments/finalizers",
						"clusters", "clusters/status", "clusters/finalizers"},
					Verbs: []string{"*"},
				},
				{
					APIGroups: []string{"certificates.k8s.io"},
					Resources: []string{"certificatesigningrequests"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"certificates.k8s.io"},
					Resources: []string{"certificatesigningrequests/approval"},
					Verbs:     []string{"update"},
				},
				{
					APIGroups:     []string{"certificates.k8s.io"},
					Resources:     []string{"signers"},
					ResourceNames: []string{"kubernetes.io/kubelet-serving"},
					Verbs:         []string{"approve"},
				},
			}
			return cr, nil
		}
	}
}
