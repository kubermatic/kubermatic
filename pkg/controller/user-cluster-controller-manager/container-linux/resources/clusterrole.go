package resources

import (
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	ClusterRoleName = "container-linux-update-operator"
)

func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return ClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"update",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs: []string{
						"create",
						"get",
						"update",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs: []string{
						"create",
						"watch",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs: []string{
						"get",
						"list",
						"delete",
					},
				},
				{
					APIGroups: []string{"extensions"},
					Resources: []string{"daemonsets"},
					Verbs: []string{
						"get",
					},
				},
			}

			return cr, nil
		}
	}
}
