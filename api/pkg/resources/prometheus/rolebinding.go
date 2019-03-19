package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBindingCreator returns the func to create/update the RoleBinding for Prometheus
func RoleBindingCreator(clusterNamespace string) resources.NamedRoleBindingCreatorGetter {
	return func() (string, resources.RoleBindingCreator) {
		return resources.PrometheusRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabel(name, nil)

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.PrometheusRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      resources.PrometheusServiceAccountName,
					Namespace: clusterNamespace,
				},
			}
			return rb, nil
		}
	}

}
