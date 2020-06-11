package resources

import (
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterRoleBindingName = "container-linux-update-operator"
)

func ClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return ClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     ClusterRoleName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Namespace: metav1.NamespaceSystem,
					Name:      ServiceAccountName,
				},
			}

			return crb, nil
		}
	}
}
