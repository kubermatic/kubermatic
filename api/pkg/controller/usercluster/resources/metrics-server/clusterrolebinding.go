package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingResourceReaderCreator returns the ClusterRoleBinding required for the metrics server to read all required resources
func ClusterRoleBindingResourceReaderCreator() resources.NamedClusterRoleBindingCreatorGetter {
	return func() (string, resources.ClusterRoleBindingCreator) {
		return resources.MetricsServerResourceReaderClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabel(Name, nil)

			crb.RoleRef = rbacv1.RoleRef{
				Name:     resources.MetricsServerClusterRoleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.MetricsServerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil
		}
	}

}

// ClusterRoleBindingAuthDelegatorCreator returns the ClusterRoleBinding required for the metrics server to create token review requests
func ClusterRoleBindingAuthDelegatorCreator() resources.NamedClusterRoleBindingCreatorGetter {
	return resources.ClusterRoleBindingAuthDelegatorCreator(resources.MetricsServerCertUsername)
}
