package resources

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

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
					Resources: []string{"nodes"},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"update",
					},
				},
				{
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
					Resources: []string{"events"},
					Verbs: []string{
						"create",
						"watch",
					},
				},
				{
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
