package dnatcontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingCreator returns the func to create/update the ClusterRoleBinding for the DNAT controller.
func ClusterRoleBindingCreator() resources.NamedClusterRoleBindingCreatorGetter {
	return func() (string, resources.ClusterRoleBindingCreator) {
		return resources.KubeletDnatControllerClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				Name:     resources.KubeletDnatControllerClusterRoleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.KubeletDnatControllerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return crb, nil
		}
	}
}
