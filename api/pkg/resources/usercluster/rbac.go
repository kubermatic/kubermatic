package usercluster

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

const (
	serviceAccountName = "kubermatic-usercluster-controller-manager"
	roleName           = "kubermatic:usercluster-controller-manager"
	roleBindingName    = "kubermatic:usercluster-controller-manager"
)

func ServiceAccountCreator() (string, reconciling.ServiceAccountCreator) {
	return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func RoleCreator() (string, reconciling.RoleCreator) {
	return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Rules = []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		}}
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
