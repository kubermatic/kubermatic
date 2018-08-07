package rbac

import (
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OwnerGroupNamePrefix represents owners group prefix
	OwnerGroupNamePrefix = "owners"

	// editorGroupNamePrefix represents editors group prefix
	editorGroupNamePrefix = "editors"

	// viewerGroupNamePrefix represents viewers group prefix
	viewerGroupNamePrefix = "viewers"

	// rbacResourcesNamePrefix represents kubermatic group prefix
	rbacResourcesNamePrefix = "kubermatic"
)

// AllGroupsPrefixes holds a list of groups with prefixes that we will generate RBAC Roles/Binding for.
//
// Note:
// adding a new group also requires updating generateVerbsForNamedResource method.
// the actual names of groups are different see generateActualGroupNameFor function
var AllGroupsPrefixes = []string{
	OwnerGroupNamePrefix,
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

// generateClusterRBACRoleNamedResource generates ClusterRole for a named resource.
// named resources have its rules set to a resource with the given name for example:
// the following rule allows reading a ConfigMap named “my-config”
//  rules:
//   - apiGroups: [""]
//   resources: ["configmaps"]
//   resourceNames: ["my-config"]
//   verbs: ["get"]
func generateClusterRBACRoleNamedResource(kind, groupName, policyResource, policyAPIGroups, policyResourceName string, oRef metav1.OwnerReference) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbsForNamedResource(groupName, kind)
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

// generateClusterRBACRoleBindingNamedResource generates ClusterRoleBiding for the given group
// that will be bound to the corresponding ClusterRole
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

// generateClusterRBACRoleForResource generates ClusterRole for the given resource
// Note that for some groups we don't want to generate ClusterRole in that case a nil will be returned
func generateClusterRBACRoleForResource(groupName, policyResource, policyAPIGroups string) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbsForResource(groupName)
	if err != nil {
		return nil, err
	}
	if len(verbs) == 0 {
		return nil, nil
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

// generateVerbsForNamedResource generates a set of verbs for a named resource
// for example a "cluster" named "beefy-john"
func generateVerbsForNamedResource(groupName, resourceKind string) ([]string, error) {
	// verbs for owners
	//
	// owners of a named resource
	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) {
		return []string{"get", "update", "delete"}, nil
	}

	// verbs for editors
	//
	// editors of a project
	// special case - editors are not allowed to delete a project
	if strings.HasPrefix(groupName, editorGroupNamePrefix) && resourceKind == "Project" {
		return []string{"get", "update"}, nil
	}

	// editors of a named resource
	if strings.HasPrefix(groupName, editorGroupNamePrefix) {
		return []string{"get", "update", "delete"}, nil
	}

	// verbs for editors
	//
	// viewers of a named resource
	if strings.HasPrefix(groupName, viewerGroupNamePrefix) {
		return []string{"get"}, nil
	}

	// unknown group passed
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}

// generateVerbsForResource generates verbs for a resource for example "cluster"
// to make it even more concrete, if there is "create" verb returned for owners group, that means that the owners can create "cluster" resources.
func generateVerbsForResource(groupName string) ([]string, error) {
	// verbs for owners and editors
	//
	// owners and editors can create resources
	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) || strings.HasPrefix(groupName, editorGroupNamePrefix) {
		return []string{"create"}, nil
	}

	// verbs for readers
	//
	// viewers cannot create resources
	if strings.HasPrefix(groupName, viewerGroupNamePrefix) {
		return []string{}, nil
	}

	// unknown group passed
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}
