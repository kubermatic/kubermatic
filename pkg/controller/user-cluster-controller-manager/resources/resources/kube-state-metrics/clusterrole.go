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

package kubestatemetrics

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	Name = "kube-state-metrics"
)

// ClusterRoleReconciler returns the func to create/update the ClusterRole for kube-state-metrics.
func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return resources.KubeStateMetricsClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabels(Name, nil)

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"configmaps",
						"secrets",
						"nodes",
						"pods",
						"services",
						"resourcequotas",
						"replicationcontrollers",
						"limitranges",
						"persistentvolumeclaims",
						"persistentvolumes",
						"namespaces",
						"endpoints",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"discovery.k8s.io"},
					Resources: []string{"endpointslices"},
					Verbs:     []string{"list", "watch"},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{
						"daemonsets",
						"deployments",
						"replicasets",
						"statefulsets",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"batch"},
					Resources: []string{
						"cronjobs",
						"jobs",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"autoscaling"},
					Resources: []string{
						"horizontalpodautoscalers",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"authentication.k8s.io"},
					Resources: []string{
						"tokenreviews",
					},
					Verbs: []string{"create"},
				},
				{
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{
						"subjectaccessreviews",
					},
					Verbs: []string{"create"},
				},
				{
					APIGroups: []string{"policy"},
					Resources: []string{
						"poddisruptionbudgets",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"certificates.k8s.io"},
					Resources: []string{
						"certificatesigningrequests",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"storage.k8s.io"},
					Resources: []string{
						"storageclasses",
						"volumeattachments",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{
						"networkpolicies",
						"ingresses",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{
						"mutatingwebhookconfigurations",
						"validatingwebhookconfigurations",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{
						"leases",
					},
					Verbs: []string{"list", "watch"},
				},
			}
			return cr, nil
		}
	}
}
