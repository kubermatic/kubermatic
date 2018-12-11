package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for the metrics server
func ClusterRole(_ resources.ClusterRoleDataProvider, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	r := existing
	if r == nil {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.MetricsServerClusterRoleName
	r.Labels = resources.BaseAppLabel(name, nil)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"pods",
				"nodes",
				"nodes/stats",
				"namespaces",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
		{
			APIGroups: []string{"extensions"},
			Resources: []string{
				"deployments",
			},
			Verbs: []string{"get", "list", "watch"},
		},
	}
	return r, nil
}
