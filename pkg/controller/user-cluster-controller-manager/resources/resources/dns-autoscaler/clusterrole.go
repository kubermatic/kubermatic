package dnsautoscaler

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRolereator returns the func to create/update the ClusterRole for dns autoscaler.
func ClusterRolereator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.DNSAutoscalerClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"watch", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get", "create"},
				},
				{
					APIGroups: []string{"extensions", "apps"},
					Resources: []string{"deployments/scale"},
					Verbs:     []string{"get", "update"},
				},
			}
			return cr, nil
		}
	}
}
