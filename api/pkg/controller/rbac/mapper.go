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

func generateRBACRole(resource, resourceKind, groupName, policyAPIGroups, policyResourceName string, oRef metav1.OwnerReference) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbs(groupName, resourceKind)
	if err != nil {
		return nil, err
	}
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s:%s:%s", rbacPrefix, strings.ToLower(resourceKind), groupName),
			OwnerReferences: []metav1.OwnerReference{oRef},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{policyAPIGroups},
				Resources:     []string{resource},
				ResourceNames: []string{policyResourceName},
				Verbs:         verbs,
			},
		},
	}
	return role, nil
}

func generateRBACRoleBinding(resourceKind, groupName string, oRef metav1.OwnerReference) *rbacv1.ClusterRoleBinding {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s:%s:%s", rbacPrefix, strings.ToLower(resourceKind), groupName),
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
			Name:     fmt.Sprintf("%s:%s:%s", rbacPrefix, strings.ToLower(resourceKind), groupName),
		},
	}
	return binding
}

func generateVerbs(groupName, resourceKind string) ([]string, error) {
	// owners of a project
	if strings.HasPrefix(groupName, ownerGroupName) && resourceKind == "Project" {
		return []string{"get", "update", "delete"}, nil
	}
	// admins of a project
	if strings.HasPrefix(groupName, adminGroupName) && resourceKind == "Project" {
		return []string{"get", "update"}, nil
	}
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}
