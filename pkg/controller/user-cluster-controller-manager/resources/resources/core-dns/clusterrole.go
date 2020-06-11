package coredns

import (
	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleCreator returns the func to create/update the ClusterRole for CoreDNS
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.CoreDNSClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{
						"endpoints",
						"services",
						"pods",
						"namespaces",
					},
					Verbs: []string{"list", "watch"},
				},
			}
			return cr, nil
		}
	}
}
