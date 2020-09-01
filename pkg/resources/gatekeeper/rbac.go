package gatekeeper

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

const (
	roleName               = "gatekeeper-manager-role"
	roleBindingName        = "gatekeeper-manager-rolebinding"
	clusterRoleName        = "gatekeeper-manager-role"
	clusterRoleBindingName = "gatekeeper-manager-rolebinding"
)

// ServiceAccountCreator returns a func to create/update the ServiceAccount used by gatekeeper.
func ServiceAccountCreator() (string, reconciling.ServiceAccountCreator) {
	return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		sa.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
		return sa, nil
	}
}

func RoleCreator() (string, reconciling.RoleCreator) {
	return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
		r.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
			},
		}
		return r, nil
	}
}

func RoleBindingCreator() (string, reconciling.RoleBindingCreator) {
	return roleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
		rb.RoleRef = rbacv1.RoleRef{
			Name:     roleName,
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: serviceAccountName,
			},
		}
		return rb, nil
	}
}

// gatekeeperClusterRoleBindingData is the data needed to construct the Gatekeeper clusterRoleBinding
type gatekeeperClusterRoleBindingData interface {
	Cluster() *kubermaticv1.Cluster
}

func ClusterRoleBindingCreator(data gatekeeperClusterRoleBindingData) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := fmt.Sprintf("%s.%s", data.Cluster().Name, clusterRoleBindingName)
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: data.Cluster().Name,
				},
			}

			return crb, nil
		}
	}
}
