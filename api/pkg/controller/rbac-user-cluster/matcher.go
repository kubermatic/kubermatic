package rbacusercluster

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

// RolesMatches compares roles OwnerReferences and Rules
func RolesMatches(existingRole, requestedRole *rbacv1.ClusterRole) bool {
	return equality.Semantic.DeepEqual(existingRole.Rules, requestedRole.Rules)
}

// ClusterRoleBindingsMatches checks if cluster role bindings have the same OwnerReferences, Subjects and RoleRef
func ClusterRoleBindingsMatches(existingRoleBindings, requestedRoleBindings *rbacv1.ClusterRoleBinding) bool {
	if !equality.Semantic.DeepEqual(existingRoleBindings.Subjects, requestedRoleBindings.Subjects) {
		return false
	}
	if !equality.Semantic.DeepEqual(existingRoleBindings.RoleRef, requestedRoleBindings.RoleRef) {
		return false
	}

	return true
}
