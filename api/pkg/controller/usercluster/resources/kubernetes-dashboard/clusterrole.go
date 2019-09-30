package kubernetesdashboard

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for the Kubernetes Dashboard
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.KubernetesDashboardClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabel(Name, nil)
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"metrics.k8s.io"},
					Resources: []string{
						"pods",
						"nodes",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
			}
			return cr, nil
		}
	}
}
