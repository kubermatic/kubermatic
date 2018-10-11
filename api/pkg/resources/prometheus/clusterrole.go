package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for prometheus
func ClusterRole(_ resources.ClusterRoleDataProvider, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	r := existing
	if r == nil {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.PrometheusClusterRoleName
	r.Labels = resources.BaseAppLabel(name, nil)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"nodes",
				"pods",
				"services",
				"namespaces",
				"endpoints",
			},
			Verbs: []string{"list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{
				"services/proxy",
				"nodes/proxy",
				"pods/proxy",
			},
			Verbs: []string{"get"},
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
			NonResourceURLs: []string{
				"/metrics",
			},
			Verbs: []string{"get"},
		},
	}
	return r, nil
}
