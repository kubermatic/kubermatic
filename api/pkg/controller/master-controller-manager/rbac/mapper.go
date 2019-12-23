package rbac

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OwnerGroupNamePrefix represents owners group prefix
	OwnerGroupNamePrefix = "owners"

	// EditorGroupNamePrefix represents editors group prefix
	EditorGroupNamePrefix = "editors"

	// ViewerGroupNamePrefix represents viewers group prefix
	ViewerGroupNamePrefix = "viewers"

	// RBACResourcesNamePrefix represents kubermatic group prefix
	RBACResourcesNamePrefix = "kubermatic"
)

const (
	saSecretsNamespaceName = "kubermatic"
)

// AllGroupsPrefixes holds a list of groups with prefixes that we will generate RBAC Roles/Binding for.
//
// Note:
// adding a new group also requires updating generateVerbsForNamedResource method.
// the actual names of groups are different see generateActualGroupNameFor function
var AllGroupsPrefixes = []string{
	OwnerGroupNamePrefix,
	EditorGroupNamePrefix,
	ViewerGroupNamePrefix,
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
	return fmt.Sprintf("%s:%s-%s:%s", RBACResourcesNamePrefix, strings.ToLower(kind), resourceName, groupName)
}

func generateRBACRoleNameForResources(resourceName, groupName string) string {
	groupPrefix := ExtractGroupPrefix(groupName)
	return fmt.Sprintf("%s:%s:%s", RBACResourcesNamePrefix, resourceName, groupPrefix)
}

func generateRBACRoleNameForClusterNamespaceResource(kind, groupName string) string {
	return fmt.Sprintf("%s:%s:%s", RBACResourcesNamePrefix, strings.ToLower(kind), ExtractGroupPrefix(groupName))
}

// generateClusterRBACRoleNamedResource generates ClusterRole for a named resource.
// named resources have its rules set to a resource with the given name for example:
// the following rule allows reading a ConfigMap named “my-config”
//  rules:
//   - apiGroups: [""]
//   resources: ["configmaps"]
//   resourceNames: ["my-config"]
//   verbs: ["get"]
//
// Note that for some kinds we don't want to generate ClusterRole in that case a nil cluster resource will be returned without an error
func generateClusterRBACRoleNamedResource(kind, groupName, policyResource, policyAPIGroups, policyResourceName string, oRef metav1.OwnerReference) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbsForNamedResource(groupName, kind)
	if err != nil {
		return nil, err
	}
	if len(verbs) == 0 {
		return nil, nil
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
func generateClusterRBACRoleForResource(groupName, policyResource, policyAPIGroups, kind string) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbsForResource(groupName, kind)
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

func generateRBACRoleBindingForResource(resourceName, groupName, namespace string) *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForResources(resourceName, groupName),
			Namespace: namespace,
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
			Kind:     "Role",
			Name:     generateRBACRoleNameForResources(resourceName, groupName),
		},
	}
	return binding
}

// generateRBACRoleForResource generates Role for the given resource in the given namespace
// Note that for some groups we don't want to generate Role in that case a nil will be returned
func generateRBACRoleForResource(groupName, policyResource, policyAPIGroups, kind string, namespace string) (*rbacv1.Role, error) {
	verbs, err := generateVerbsForNamespacedResource(groupName, kind, namespace)
	if err != nil {
		return nil, err
	}
	if len(verbs) == 0 {
		return nil, nil
	}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForResources(policyResource, groupName),
			Namespace: namespace,
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

// generateRBACRoleNamedResource generates Role for a named resource.
// named resources have its rules set to a resource with the given name for example:
// the following rule allows reading a ConfigMap named “my-config”
//  rules:
//   - apiGroups: [""]
//   resources: ["configmaps"]
//   resourceNames: ["my-config"]
//   verbs: ["get"]
//
// Note that for some kinds we don't want to generate Role in that case a nil cluster resource will be returned without an error
func generateRBACRoleNamedResource(kind, groupName, policyResource, policyAPIGroups, policyResourceName string, namespace string, oRef metav1.OwnerReference) (*rbacv1.Role, error) {
	verbs, err := generateVerbsForNamedResourceInNamespace(groupName, kind, namespace)
	if err != nil {
		return nil, err
	}
	if len(verbs) == 0 {
		return nil, nil
	}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRBACRoleNameForNamedResource(kind, policyResourceName, groupName),
			OwnerReferences: []metav1.OwnerReference{oRef},
			Namespace:       namespace,
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

// generateRBACRoleBindingNamedResource generates RoleBiding for the given group
// that will be bound to the corresponding Role
func generateRBACRoleBindingNamedResource(kind, resourceName, groupName, namespace string, oRef metav1.OwnerReference) *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRBACRoleNameForNamedResource(kind, resourceName, groupName),
			OwnerReferences: []metav1.OwnerReference{oRef},
			Namespace:       namespace,
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
			Kind:     "Role",
			Name:     generateRBACRoleNameForNamedResource(kind, resourceName, groupName),
		},
	}
	return binding
}

// generateRBACRoleForClusterNamespaceResource generates per-cluster Role for the given cluster in the cluster namespace
// Note that for some groups we don't want to generate Role in that case a nil will be returned
func generateRBACRoleForClusterNamespaceResource(cluster *kubermaticv1.Cluster, groupName, policyResource, policyAPIGroups, kind string) (*rbacv1.Role, error) {
	verbs, err := generateVerbsForClusterNamespaceResource(cluster, groupName, kind)
	if err != nil {
		return nil, err
	}
	if len(verbs) == 0 {
		return nil, nil
	}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForClusterNamespaceResource(kind, groupName),
			Namespace: cluster.Status.NamespaceName,
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

// generateRBACRoleBindingForClusterNamespaceResource generates per-cluster RoleBinding for the given cluster in the cluster namespace
// Note that for some groups we don't want to generate RoleBinding in that case a nil will be returned
func generateRBACRoleBindingForClusterNamespaceResource(cluster *kubermaticv1.Cluster, groupName, kind string) *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForClusterNamespaceResource(kind, groupName),
			Namespace: cluster.Status.NamespaceName,
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
			Kind:     "Role",
			Name:     generateRBACRoleNameForClusterNamespaceResource(kind, groupName),
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
	if strings.HasPrefix(groupName, EditorGroupNamePrefix) && resourceKind == kubermaticv1.ProjectKindName {
		return []string{"get", "update"}, nil
	}
	// special case - editors are not allowed to interact with members of a project (UserProjectBinding)
	if strings.HasPrefix(groupName, EditorGroupNamePrefix) && resourceKind == kubermaticv1.UserProjectBindingKind {
		return nil, nil
	}
	// special case - editors are not allowed to interact with service accounts (User)
	if strings.HasPrefix(groupName, EditorGroupNamePrefix) && resourceKind == kubermaticv1.UserKindName {
		return nil, nil
	}

	// editors of a named resource
	if strings.HasPrefix(groupName, EditorGroupNamePrefix) {
		return []string{"get", "update", "delete"}, nil
	}

	// verbs for editors
	//
	// viewers of a named resource
	// special case - viewers are not allowed to interact with members of a project (UserProjectBinding)
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) && resourceKind == kubermaticv1.UserProjectBindingKind {
		return nil, nil
	}
	// special case - viewers are not allowed to interact with service accounts (User)
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) && resourceKind == kubermaticv1.UserKindName {
		return nil, nil
	}
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) {
		return []string{"get"}, nil
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}

// generateVerbsForResource generates verbs for a resource for example "cluster"
// to make it even more concrete, if there is "create" verb returned for owners group, that means that the owners can create "cluster" resources.
func generateVerbsForResource(groupName, resourceKind string) ([]string, error) {
	// special case - only the owners of a project can manipulate members
	//
	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) && resourceKind == kubermaticv1.UserProjectBindingKind {
		return []string{"create"}, nil
	} else if resourceKind == kubermaticv1.UserProjectBindingKind {
		return nil, nil
	}

	// special case - only the owners of a project can create service account (aka. users)
	//
	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) && resourceKind == kubermaticv1.UserKindName {
		return []string{"create"}, nil
	} else if resourceKind == kubermaticv1.UserKindName {
		return nil, nil
	}

	// verbs for owners and editors
	//
	// owners and editors can create resources
	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) || strings.HasPrefix(groupName, EditorGroupNamePrefix) {
		return []string{"create"}, nil
	}

	// verbs for readers
	//
	// viewers cannot create resources
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) {
		return nil, nil
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}

func generateVerbsForNamespacedResource(groupName, resourceKind, namespace string) ([]string, error) {
	// special case - only the owners of a project can create secrets in "saSecretsNamespaceName" namespace
	//
	if namespace == saSecretsNamespaceName {
		secretV1Kind := "Secret"
		if strings.HasPrefix(groupName, OwnerGroupNamePrefix) && resourceKind == secretV1Kind {
			return []string{"create"}, nil
		} else if resourceKind == secretV1Kind {
			return nil, nil
		}
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s, namespace = %s", groupName, namespace)
}

// generateVerbsForNamedResourceInNamespace generates a set of verbs for a named resource in a given namespace
// for example a "cluster" named "beefy-john"
func generateVerbsForNamedResourceInNamespace(groupName, resourceKind, namespace string) ([]string, error) {
	// special case - only the owners of a project can manipulate secrets in "ssaSecretsNamespaceNam" namespace
	//
	if namespace == saSecretsNamespaceName {
		secretV1Kind := "Secret"
		if strings.HasPrefix(groupName, OwnerGroupNamePrefix) && resourceKind == secretV1Kind {
			return []string{"get", "update", "delete"}, nil
		} else if resourceKind == secretV1Kind {
			return nil, nil
		}
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs for group = %s, kind = %s, namespace = %s", groupName, resourceKind, namespace)
}

func generateVerbsForClusterNamespaceResource(cluster *kubermaticv1.Cluster, groupName, kind string) ([]string, error) {
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) && kind == kubermaticv1.AddonKindName {
		return []string{"get", "list"}, nil
	}

	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) || strings.HasPrefix(groupName, EditorGroupNamePrefix) {
		return []string{"get", "list", "create", "update", "delete"}, nil
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs for cluster namespace resource cluster = %s, group = %s, kind = %s", cluster.Name, groupName, kind)
}
