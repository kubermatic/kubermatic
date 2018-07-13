package rbac

import (
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ownerGroupNamePrefix  = "owners"
	editorGroupNamePrefix = "editors"
	viewerGroupNamePrefix = "viewers"
	rbacResourcesNamePrefix = "kubermatic"
)

// AllGroupsPrefixes holds a list of groups with prefixes that we will generate RBAC Roles/Binding for.
//
// Note:
// adding a new group also requires updating generateVerbs method.
// the actual names of groups are different see generateActualGroupNameFor function
var allGroupsPrefixes = []string{
	ownerGroupNamePrefix,
	editorGroupNamePrefix,
	viewerGroupNamePrefix,
}

// GenerateActualGroupNameFor generates a group name for the given project and group prefix.
func GenerateActualGroupNameFor(projectName, groupName string) string {
	return fmt.Sprintf("%s-%s", groupName, projectName)
}

// ExtractGroupPrefix extracts only group prefix from the given group name
func ExtractGroupPrefix(groupName string) string {
	ret := strings.Split(groupName, "-")
	if len(ret) > 0 {
		return ret[0]
	}
	return groupName
}

func generateRBACRoleNameForNamedResource(kind, resourceName, groupName string) string {
	return fmt.Sprintf("%s:%s-%s:%s", rbacResourcesNamePrefix, strings.ToLower(kind), resourceName, groupName)
}

func generateRBACRoleNameForResources(resourceName, groupName string) string {
	groupPrefix := ExtractGroupPrefix(groupName)
	return fmt.Sprintf("%s:%s:%s", rbacResourcesNamePrefix, resourceName, groupPrefix)
}

func generateClusterRBACRoleNamedResource(kind, groupName, policyResource, policyAPIGroups, policyResourceName string, oRef metav1.OwnerReference) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbs(groupName, kind)
	if err != nil {
		return nil, err
	}
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRBACRoleNameForNamedResource(kind, policyResourceName, groupName),
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

func generateClusterRBACRoleBindingNamedResource(kind, resourceName, groupName string, oRef metav1.OwnerReference) *rbacv1.ClusterRoleBinding {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRBACRoleNameForNamedResource(kind, resourceName, groupName),
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
			Name:     generateRBACRoleNameForNamedResource(kind, resourceName, groupName),
		},
	}
	return binding
}

func generateClusterRBACRoleForResource(kind, groupName, policyResource, policyAPIGroups string) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbs(groupName, kind)
	if err != nil {
		return nil, err
	}
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: generateRBACRoleNameForResources(policyResource, groupName),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{policyAPIGroups},
				Resources: []string{policyResource},
				Verbs:     verbs,
			},
		},
	}
	return role, nil
}

func generateClusterRBACRoleBindingForResource(resourceName, groupName string) *rbacv1.ClusterRoleBinding {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: generateRBACRoleNameForResources(resourceName, groupName),
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
			Name:     generateRBACRoleNameForResources(resourceName, groupName),
		},
	}
	return binding
}

func generateVerbs(groupName, resourceKind string) ([]string, error) {
	// verbs for owners
	//
	// owners of a resource
	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) {
		return []string{"create", "get", "update", "delete"}, nil
	}

	// verbs for editors
	//
	// editors of a project
	// special case - editors are not allowed to delete a project
	if strings.HasPrefix(groupName, EditorGroupNamePrefix) && resourceKind == "Project" {
		return []string{"create", "get", "update"}, nil
	}

	// editors of a resource
	if strings.HasPrefix(groupName, EditorGroupNamePrefix) {
		return []string{"create", "get", "update", "delete"}, nil
	}

	// verbs for editors
	//
	// viewers of a resource
	if strings.HasPrefix(groupName, viewerGroupNamePrefix) {
		return []string{"get"}, nil
	}
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}
