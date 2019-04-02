package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingCreator returns a func to create/update the ClusterRoleBinding for Prometheus
func ClusterRoleBindingCreator() resources.NamedClusterRoleBindingCreatorGetter {
	return func() (string, resources.ClusterRoleBindingCreator) {
		return resources.PrometheusClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabel(Name, nil)

			crb.RoleRef = rbacv1.RoleRef{
				Name:     resources.PrometheusClusterRoleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.PrometheusCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil

		}
	}
}
