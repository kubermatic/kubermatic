package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	Name = "prometheus"
)

// ClusterRoleCreator returns the func to create/update the ClusterRole for Prometheus
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.PrometheusClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabels(Name, nil)

			cr.Rules = []rbacv1.PolicyRule{
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
			return cr, nil
		}
	}
}
