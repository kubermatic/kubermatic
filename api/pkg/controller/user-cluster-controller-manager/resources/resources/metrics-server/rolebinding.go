package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RolebindingAuthReaderCreator returns a func to create/update the RoleBinding used by the metrics-server to get access to the token subject review API
func RolebindingAuthReaderCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.MetricsServerAuthReaderRoleName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(Name, nil)

			rb.RoleRef = rbacv1.RoleRef{
				Name:     "extension-apiserver-authentication-reader",
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.MetricsServerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}
