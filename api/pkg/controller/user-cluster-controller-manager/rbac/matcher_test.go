package rbacusercluster_test

import (
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testName = "test"

func TestRolesMatches(t *testing.T) {

	tests := []struct {
		name          string
		existingRole  *rbacv1.ClusterRole
		requestedRole *rbacv1.ClusterRole
		expected      bool
	}{
		{
			name:          "scenario 1: roles are equal",
			existingRole:  generateRBACClusterRole(testName, testName, []string{"create", "delete"}, genFullOwnerReference(testName, testName, testName)),
			requestedRole: generateRBACClusterRole(testName, testName, []string{"create", "delete"}, genFullOwnerReference(testName, testName, testName)),
			expected:      true,
		},
		{
			name:          "scenario 3: roles have different rules (verbs)",
			existingRole:  generateRBACClusterRole(testName, testName, []string{"create", "delete"}, genFullOwnerReference(testName, testName, testName)),
			requestedRole: generateRBACClusterRole(testName, testName, []string{"get", "list"}, genFullOwnerReference(testName, testName, testName)),
			expected:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := rbacusercluster.ClusterRoleMatches(test.existingRole, test.requestedRole)
			if result != test.expected {
				t.Fatalf("role comparison failed expected %v got %v", test.expected, result)
			}

		})
	}
}

func TestRoleBindingsMatches(t *testing.T) {

	tests := []struct {
		name                string
		existingRoleBinding *rbacv1.ClusterRoleBinding
		requestedRoleBiding *rbacv1.ClusterRoleBinding
		expected            bool
	}{
		{
			name:                "scenario 1: role bindings are equal",
			existingRoleBinding: generateRBACClusterRoleBinding(testName, testName, genFullOwnerReference(testName, testName, testName)),
			requestedRoleBiding: generateRBACClusterRoleBinding(testName, testName, genFullOwnerReference(testName, testName, testName)),
			expected:            true,
		},
		{
			name:                "scenario 3: role bindings have different RoleRef",
			existingRoleBinding: generateRBACClusterRoleBinding(testName, testName, genFullOwnerReference(testName, testName, testName)),
			requestedRoleBiding: generateRBACClusterRoleBinding("viper", testName, genFullOwnerReference(testName, testName, testName)),
			expected:            false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := rbacusercluster.ClusterRoleBindingMatches(test.existingRoleBinding, test.requestedRoleBiding)
			if result != test.expected {
				t.Fatalf("role binding comparison failed expected %v got %v", test.expected, result)
			}

		})
	}
}

func generateRBACClusterRoleBinding(resourceName, groupName string, oRef metav1.OwnerReference) *rbacv1.ClusterRoleBinding {

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            resourceName,
			OwnerReferences: []metav1.OwnerReference{oRef},
			Namespace:       metav1.NamespaceSystem,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     groupName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     resourceName,
		},
	}
	return binding
}

func generateRBACClusterRole(resourceName, policyAPIGroups string, verbs []string, oRef metav1.OwnerReference) *rbacv1.ClusterRole {

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            resourceName,
			OwnerReferences: []metav1.OwnerReference{oRef},
			Namespace:       metav1.NamespaceSystem,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{policyAPIGroups},
				Resources: []string{resourceName},
				Verbs:     verbs,
			},
		},
	}
	return role
}

func genFullOwnerReference(apiVersion, kind, name string) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
	}

}
