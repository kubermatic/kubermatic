package kubestatemetrics

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingCreator returns a func to create/update the ClusterRoleBinding for kube-state-metrics
func ClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return resources.KubeStateMetricsClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(Name, nil)

			crb.RoleRef = rbacv1.RoleRef{
				Name:     resources.KubeStateMetricsClusterRoleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.KubeStateMetricsCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil
		}
	}
}
