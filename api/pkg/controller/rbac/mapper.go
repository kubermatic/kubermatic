package rbac

import (
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ownerGroupName = "owners"
	adminGroupName = "admins"

	rbacPrefix = "kubermatic"
)

// allGroups holds a list of groups that we will generate RBAC Roles/Binding for.
//
// Note:
// adding a new group also requires updating generateVerbs method.
var allGroups = []string{
	ownerGroupName,
	adminGroupName,
}

func generateGroupNameFor(projectName, groupName string) string {
	return fmt.Sprintf("%s-%s", groupName, projectName)
}

func generateRBACRoleName(kind, resourceName, groupName string) string {
	return fmt.Sprintf("%s:%s-%s:%s", rbacPrefix, strings.ToLower(kind), resourceName, groupName)
}

func generateRBACRole(kind, groupName, policyResource, policyAPIGroups, policyResourceName string, oRef metav1.OwnerReference) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbs(groupName, kind)
	if err != nil {
		return nil, err
	}
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRBACRoleName(kind, policyResourceName, groupName),
			OwnerReferences: []metav1.OwnerReference{oRef},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{policyAPIGroups},
				Resources:     []string{policyResource},
				ResourceNames: []string{policyResourceName},
				Verbs:         verbs,
			},
		},
	}
	return role, nil
}

func generateRBACRoleBinding(kind, resourceName, groupName string, oRef metav1.OwnerReference) *rbacv1.ClusterRoleBinding {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRBACRoleName(kind, resourceName, groupName),
			OwnerReferences: []metav1.OwnerReference{oRef},
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
			Name:     generateRBACRoleName(kind, resourceName, groupName),
		},
	}
	return binding
}

func generateVerbs(groupName, resourceKind string) ([]string, error) {
	// verbs for owners
	//
	// owners of a resource
	if strings.HasPrefix(groupName, ownerGroupName) {
		return []string{"get", "update", "delete"}, nil
	}

	// verbs for admins
	//
	// admins of a project
	// special case - admins are not allowed to delete a project
	if strings.HasPrefix(groupName, adminGroupName) && resourceKind == "Project" {
		return []string{"get", "update"}, nil
	}

	// admins of a resource
	if strings.HasPrefix(groupName, adminGroupName) {
		return []string{"get", "update", "delete"}, nil
	}
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}
