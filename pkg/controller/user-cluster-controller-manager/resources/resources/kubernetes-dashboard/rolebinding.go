package kubernetesdashboard

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBindingCreator creates the role binding for the Kubernetes Dashboard
func RoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.KubernetesDashboardRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(AppName, nil)
			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.KubernetesDashboardRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.KubernetesDashboardCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}
