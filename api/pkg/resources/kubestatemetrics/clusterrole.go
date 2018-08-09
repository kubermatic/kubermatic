package kubestatemetrics

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for kube-state-metrics
func ClusterRole(_ *resources.TemplateData, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	var r *rbacv1.ClusterRole
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.KubeStateMetricsClusterRoleName
	r.Labels = resources.GetLabels(name)

	r.Rules = []rbacv1.PolicyRule{
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
			},
			Verbs: []string{"list", "watch"},
		},
		{
			APIGroups: []string{"apps"},
			Resources: []string{
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
	}
	return r, nil
}
