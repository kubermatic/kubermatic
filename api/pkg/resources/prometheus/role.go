package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleCreator returns the func to create/update the role for Prometheus
func RoleCreator() resources.NamedRoleCreatorGetter {
	return func() (string, resources.RoleCreator) {
		return resources.PrometheusRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = resources.BaseAppLabel(name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"nodes",
						"services",
						"endpoints",
						"pods",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get"},
				},
			}
			return r, nil
		}
	}
}
