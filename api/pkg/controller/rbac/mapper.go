package rbac

import (
	"fmt"

	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ownerGroupName = "owners"

	rbacPrefix = "kubermatic"
)

func generateOwnersGroupName(projectName string) string {
	return fmt.Sprintf("%s-%s", ownerGroupName, projectName)
}

func generateRBACRole(resource, resourceKind, groupName, policyAPIGroups, policyResourceName string, oRef metav1.OwnerReference) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbs(groupName)
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

func generateVerbs(groupName string) ([]string, error) {
	if strings.HasPrefix(groupName, ownerGroupName) {
		return []string{"get", "update", "delete"}, nil
	}
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}
