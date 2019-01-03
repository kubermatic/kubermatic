package resources

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBindingAuthenticationReaderCreator returns a function to create the RoleBinding which is needed for extension apiserver which do auth delegation
func RoleBindingAuthenticationReaderCreator(username string) RoleBindingCreator {
	return func(data RoleBindingDataProvider, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
		var rb *rbacv1.RoleBinding
		if existing != nil {
			rb = existing
		} else {
			rb = &rbacv1.RoleBinding{}
		}

		rb.Name = username + "-authentication-reader"
		rb.Namespace = "kube-system"

		rb.RoleRef = rbacv1.RoleRef{
			Name:     "extension-apiserver-authentication-reader",
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:     "User",
				Name:     username,
				APIGroup: rbacv1.GroupName,
			},
		}
		return rb, nil
	}
}

// ClusterRoleBindingAuthDelegatorCreator returns a function to create the ClusterRoleBinding which is needed for extension apiserver which do auth delegation
func ClusterRoleBindingAuthDelegatorCreator(username string) ClusterRoleBindingCreator {
	return func(data ClusterRoleBindingDataProvider, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
		var crb *rbacv1.ClusterRoleBinding
		if existing != nil {
			crb = existing
		} else {
			crb = &rbacv1.ClusterRoleBinding{}
		}

		crb.Name = username + "-auth-delegator"

		crb.RoleRef = rbacv1.RoleRef{
			Name:     "system:auth-delegator",
			Kind:     "ClusterRole",
			APIGroup: rbacv1.GroupName,
		}
		crb.Subjects = []rbacv1.Subject{
			{
				Kind:     "User",
				Name:     username,
				APIGroup: rbacv1.GroupName,
			},
		}
		return crb, nil
	}
}
