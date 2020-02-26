package kubernetesdashboard

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingCreator returns the ClusterRoleBinding required by the dashboard metrics scraper to
// read all required resources from the metrics server.
func ClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return resources.MetricsScraperClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(scraperName, nil)

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Name:     resources.MetricsScraperClusterRoleName,
				Kind:     "ClusterRole",
			}
			crb.Subjects = []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      resources.MetricsScraperServiceAccountUsername,
				Namespace: Namespace,
			}}
			return crb, nil
		}
	}

}
