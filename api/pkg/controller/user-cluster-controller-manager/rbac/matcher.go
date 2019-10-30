package rbacusercluster

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

// ClusterRoleMatches compares cluster role Rules
func ClusterRoleMatches(existingRole, requestedRole *rbacv1.ClusterRole) bool {
	return equality.Semantic.DeepEqual(existingRole.Rules, requestedRole.Rules)
}

// ClusterRoleBindingMatches checks if cluster role bindings have the same Subjects and RoleRefs
func ClusterRoleBindingMatches(existingClusterRoleBinding, requestedClusterRoleBinding *rbacv1.ClusterRoleBinding) bool {
	if !equality.Semantic.DeepEqual(existingClusterRoleBinding.Subjects, requestedClusterRoleBinding.Subjects) {
		return false
	}
	if !equality.Semantic.DeepEqual(existingClusterRoleBinding.RoleRef, requestedClusterRoleBinding.RoleRef) {
		return false
	}

	return true
}
