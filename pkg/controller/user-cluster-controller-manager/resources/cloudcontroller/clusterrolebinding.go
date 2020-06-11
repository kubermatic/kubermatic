package cloudcontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingCreator returns a func to create/update the ClusterRoleBinding for external cloud controllers.
func ClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return resources.CloudControllerManagerRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(resources.CloudControllerManagerRoleBindingName, nil)

			crb.RoleRef = rbacv1.RoleRef{
				// Can probably be tightened up a bit but for now I'm following the documentation.
				// https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/
				Name:     "cluster-admin",
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.CloudControllerManagerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil
		}
	}
}
