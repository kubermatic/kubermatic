package kubestatemetrics

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	Name = "kube-state-metrics"
)

// ClusterRoleCreator returns the func to create/update the ClusterRole for kube-state-metrics
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
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
					APIGroups: []string{"extensions"},
					Resources: []string{
						"daemonsets",
						"deployments",
						"replicasets",
						"ingresses",
					},
					Verbs: []string{"list", "watch"},
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
					Resources: []string{"storageclasses"},
					Verbs:     []string{"list", "watch"},
				},
			}
			return cr, nil
		}
	}
}
