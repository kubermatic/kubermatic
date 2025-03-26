/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbac

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OwnerGroupNamePrefix represents owners group prefix.
	OwnerGroupNamePrefix = "owners"

	// EditorGroupNamePrefix represents editors group prefix.
	EditorGroupNamePrefix = "editors"

	// ViewerGroupNamePrefix represents viewers group prefix.
	ViewerGroupNamePrefix = "viewers"

	// ProjectManagerGroupNamePrefix represents project managers group prefix.
	// Can create, update and delete projects and add/remove members & service accounts.
	ProjectManagerGroupNamePrefix = "projectmanagers"

	// RBACResourcesNamePrefix represents kubermatic group prefix.
	RBACResourcesNamePrefix = "kubermatic"
)

const (
	saSecretsNamespaceName              = "kubermatic"
	alertmanagerName                    = "alertmanager"
	defaultAlertmanagerConfigSecretName = "alertmanager"

	secretV1Kind = "Secret"
)

// AllGroupsPrefixes holds a list of groups with prefixes that we will generate RBAC Roles/Binding for.
//
// Note:
// adding a new group also requires updating generateVerbsForNamedResource method.
// the actual names of groups are different see generateActualGroupNameFor function.
var AllGroupsPrefixes = []string{
	OwnerGroupNamePrefix,
	EditorGroupNamePrefix,
	ViewerGroupNamePrefix,
	ProjectManagerGroupNamePrefix,
}

// GenerateActualGroupNameFor generates a group name for the given project and group prefix.
func GenerateActualGroupNameFor(projectName, groupName string) string {
	return fmt.Sprintf("%s-%s", groupName, projectName)
}

// ExtractGroupPrefix extracts only group prefix from the given group name.
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

func generateRBACRoleNameForNamedResourceWithServiceAccount(kind, resourceName, serviceAccount string) string {
	return fmt.Sprintf("%s:%s-%s:%s", RBACResourcesNamePrefix, strings.ToLower(kind), resourceName, serviceAccount)
}

func generateRBACRoleNameForResources(resourceName, groupName string) string {
	groupPrefix := ExtractGroupPrefix(groupName)
	return fmt.Sprintf("%s:%s:%s", RBACResourcesNamePrefix, resourceName, groupPrefix)
}

func generateRBACRoleNameForClusterNamespaceResource(kind, groupName string) string {
	return fmt.Sprintf("%s:%s:%s", RBACResourcesNamePrefix, strings.ToLower(kind), ExtractGroupPrefix(groupName))
}

func generateRBACRoleNameForClusterNamespaceResourceAndServiceAccount(kind, serviceAccountName string) string {
	return fmt.Sprintf("%s:%s:sa-%s", RBACResourcesNamePrefix, strings.ToLower(kind), serviceAccountName)
}

func generateRBACRoleNameForClusterNamespaceNamedResource(kind, resourceName, groupName string) string {
	return fmt.Sprintf("%s:%s-%s:%s", RBACResourcesNamePrefix, strings.ToLower(kind), resourceName, ExtractGroupPrefix(groupName))
}

// generateClusterRBACRoleNamedResource generates ClusterRole for a named resource.
// named resources have its rules set to a resource with the given name for example:
// the following rule allows reading a ConfigMap named "my-config"
//
//	rules:
//	 - apiGroups: [""]
//	   resources: ["configmaps"]
//	   resourceNames: ["my-config"]
//	   verbs: ["get"]
//
// Note that for some kinds we don't want to generate ClusterRole in that case a nil cluster resource will be returned without an error.
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
			Labels: map[string]string{
				kubermaticv1.AuthZRoleLabel: groupName,
			},
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
// that will be bound to the corresponding ClusterRole.
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
// Note that for some groups we don't want to generate ClusterRole in that case a nil will be returned.
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
			Labels: map[string]string{
				kubermaticv1.AuthZRoleLabel: groupName,
			},
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

// generateEtcdRBACRoleBindingForResourceWithServiceAccount creates a ClusterRoleBinding with etcd-launcher ServiceAccounts as a subject.
func generateEtcdRBACRoleBindingForResourceWithServiceAccount(resourceName, kind, groupName, clusterName, sa, namespace string, oRef metav1.OwnerReference) *rbacv1.ClusterRoleBinding {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            generateRBACRoleNameForNamedResourceWithServiceAccount(kind, resourceName, sa),
			OwnerReferences: []metav1.OwnerReference{oRef},
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      rbacv1.ServiceAccountKind,
				Name:      sa,
				Namespace: namespace,
			},
			{
				APIGroup:  "",
				Kind:      rbacv1.ServiceAccountKind,
				Name:      fmt.Sprintf("%s-%s", sa, clusterName),
				Namespace: metav1.NamespaceSystem,
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
// Note that for some groups we don't want to generate Role in that case a nil will be returned.
func generateRBACRoleForResource(groupName, policyResource, policyAPIGroups, kind, namespace string) (*rbacv1.Role, error) {
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
			Labels: map[string]string{
				kubermaticv1.AuthZRoleLabel: groupName,
			},
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
// the following rule allows reading a ConfigMap named "my-config"
//
//	rules:
//	 - apiGroups: [""]
//	   resources: ["configmaps"]
//	   resourceNames: ["my-config"]
//	   verbs: ["get"]
//
// Note that for some kinds we don't want to generate Role in that case a nil cluster resource will be returned without an error.
func generateRBACRoleNamedResource(kind, groupName, policyResource, policyAPIGroups, policyResourceName, namespace string, oRef metav1.OwnerReference) (*rbacv1.Role, error) {
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
			Labels: map[string]string{
				kubermaticv1.AuthZRoleLabel: groupName,
			},
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
// that will be bound to the corresponding Role.
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
// Note that for some groups we don't want to generate Role in that case a nil will be returned.
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
			Labels: map[string]string{
				kubermaticv1.AuthZRoleLabel: groupName,
			},
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

// generateRBACRoleBindingForClusterNamespaceResource generates per-cluster RoleBinding for the given cluster in the cluster namespace.
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

// generateRBACRoleForClusterNamespaceNamedResource generates per-cluster Role of named resource for the given cluster in the cluster namespace
// Note that for some groups we don't want to generate Role in that case a nil will be returned.
func generateRBACRoleForClusterNamespaceNamedResource(cluster *kubermaticv1.Cluster, groupName, policyAPIGroups, policyResource, kind, resourceName string) (*rbacv1.Role, error) {
	verbs, err := generateVerbsForClusterNamespaceNamedResource(cluster, groupName, kind, resourceName)
	if err != nil {
		return nil, err
	}
	if len(verbs) == 0 {
		return nil, nil
	}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForClusterNamespaceNamedResource(kind, resourceName, groupName),
			Namespace: cluster.Status.NamespaceName,
			Labels: map[string]string{
				kubermaticv1.AuthZRoleLabel: groupName,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{policyAPIGroups},
				Resources:     []string{policyResource},
				ResourceNames: []string{resourceName},
				Verbs:         verbs,
			},
		},
	}
	return role, nil
}

// generateRBACRoleBindingForClusterNamespaceNamedResource generates per-cluster RoleBinding for the given cluster in the cluster namespace.
func generateRBACRoleBindingForClusterNamespaceNamedResource(cluster *kubermaticv1.Cluster, groupName, kind, resourceName string) *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForClusterNamespaceNamedResource(kind, resourceName, groupName),
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
			Name:     generateRBACRoleNameForClusterNamespaceNamedResource(kind, resourceName, groupName),
		},
	}
	return binding
}

// generateRBACRoleForClusterNamespaceResourceAndServiceAccount generates per-cluster Role for the given cluster and service account in the cluster namespace.
func generateRBACRoleForClusterNamespaceResourceAndServiceAccount(cluster *kubermaticv1.Cluster, verbs []string, serviceAccountName, policyResource, policyAPIGroups, kind string) (*rbacv1.Role, error) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForClusterNamespaceResourceAndServiceAccount(kind, serviceAccountName),
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

// generateRBACRoleBindingForEtcdLauncherServiceAccount generates per-cluster RoleBinding for the given cluster and service account in the cluster namespace.
func generateRBACRoleBindingForEtcdLauncherServiceAccount(cluster *kubermaticv1.Cluster, serviceAccountName, kind string) *rbacv1.RoleBinding {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRBACRoleNameForClusterNamespaceResourceAndServiceAccount(kind, serviceAccountName),
			Namespace: cluster.Status.NamespaceName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: cluster.Status.NamespaceName,
			},
			{
				APIGroup:  "",
				Kind:      rbacv1.ServiceAccountKind,
				Name:      fmt.Sprintf("%s-%s", serviceAccountName, cluster.Name),
				Namespace: metav1.NamespaceSystem,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     generateRBACRoleNameForClusterNamespaceResourceAndServiceAccount(kind, serviceAccountName),
		},
	}
	return binding
}

// generateVerbsForNamedResource generates a set of verbs for a named resource
// for example a "cluster" named "beefy-john".
func generateVerbsForNamedResource(groupName, resourceKind string) ([]string, error) {
	if resourceKind == kubermaticv1.ResourceQuotaKindName {
		return []string{"get"}, nil
	}
	// verbs for owners
	//
	// owners of a named resource
	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) {
		return []string{"get", "update", "patch", "delete"}, nil
	}

	// verbs for editors
	//
	if strings.HasPrefix(groupName, EditorGroupNamePrefix) {
		// editors of a project
		// special case - editors are not allowed to delete a project
		if resourceKind == kubermaticv1.ProjectKindName {
			return []string{"get", "update", "patch"}, nil
		}

		// special case - editors are not allowed to interact with members of a project (UserProjectBinding, GroupProjectBinding)
		if resourceKind == kubermaticv1.UserProjectBindingKind || resourceKind == kubermaticv1.GroupProjectBindingKind {
			return nil, nil
		}

		// special case - editors are not allowed to interact with service accounts (User)
		if resourceKind == kubermaticv1.UserKindName {
			return nil, nil
		}

		// editors of a named resource
		return []string{"get", "update", "patch", "delete"}, nil
	}

	// verbs for viewers
	//
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) {
		// viewers of a named resource
		// special case - viewers are not allowed to interact with members of a project (UserProjectBinding, GroupProjectBinding)
		if resourceKind == kubermaticv1.UserProjectBindingKind || resourceKind == kubermaticv1.GroupProjectBindingKind {
			return nil, nil
		}

		// special case - viewers are not allowed to interact with service accounts (User)
		if resourceKind == kubermaticv1.UserKindName {
			return nil, nil
		}

		return []string{"get"}, nil
	}

	// verbs for projectmanagers
	//
	if strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix) {
		// special cases - projectmanagers are not allowed to interact with clusters
		if resourceKind == kubermaticv1.ClusterKindName {
			return nil, nil
		}

		if resourceKind == kubermaticv1.ExternalClusterKind {
			return nil, nil
		}

		if resourceKind == kubermaticv1.ClusterTemplateInstanceKindName {
			return nil, nil
		}

		return []string{"get", "update", "patch", "delete"}, nil
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs, unknown group name %q passed in", groupName)
}

// generateVerbsForResource generates verbs for a resource for example "cluster"
// to make it even more concrete, if there is "create" verb returned for owners group, that means that the owners can create "cluster" resources.
func generateVerbsForResource(groupName, resourceKind string) ([]string, error) {
	// special case - only the owners and project managers of a project can manipulate members
	//
	switch {
	case strings.HasPrefix(groupName, OwnerGroupNamePrefix) && resourceKind == kubermaticv1.UserProjectBindingKind:
		return []string{"create"}, nil
	case strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix) && resourceKind == kubermaticv1.UserProjectBindingKind:
		return []string{"create"}, nil
	case resourceKind == kubermaticv1.UserProjectBindingKind:
		return nil, nil
	}

	// special case - only the owners and project managers of a project can create service account (aka. users)
	//
	switch {
	case strings.HasPrefix(groupName, OwnerGroupNamePrefix) && resourceKind == kubermaticv1.UserKindName:
		return []string{"create"}, nil
	case strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix) && resourceKind == kubermaticv1.UserKindName:
		return []string{"create"}, nil
	case resourceKind == kubermaticv1.UserKindName:
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

	// verbs for project managers
	//
	// project managers cannot create other resources
	if strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix) {
		return nil, nil
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs, unknown group name %q given", groupName)
}

func generateVerbsForNamespacedResource(groupName, resourceKind, namespace string) ([]string, error) {
	// special case - only the owners of a project and project managers can create secrets in "kubermatic" namespace
	//
	if !isAcceptedNamespace(namespace) {
		return nil, fmt.Errorf("unable to generate verbs, unsupported namespace %q given", namespace)
	}
	switch resourceKind {
	case secretV1Kind:
		return generateVerbsForNamespacedSecretKind(groupName)

	case kubermaticv1.ClusterBackupStorageLocationKind:
		return generateVerbsForNamespacedCBSLKind(groupName)
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs, unknown group name %q, namespace %q given", groupName, namespace)
}

// generateVerbsForNamedResourceInNamespace generates a set of verbs for a named resource in a given namespace
// for example a "cluster" named "beefy-john".
func generateVerbsForNamedResourceInNamespace(groupName, resourceKind, namespace string) ([]string, error) {
	// special case - only the owners of a project can manipulate secrets in "ssaSecretsNamespaceName" namespace
	//
	if !isAcceptedNamespace(namespace) {
		return nil, fmt.Errorf("unable to generate verbs, unsupported namespace %q given", namespace)
	}
	switch resourceKind {
	case secretV1Kind:
		return generateVerbsForNamedSecretKindInNamespace(groupName)

	case kubermaticv1.ClusterBackupStorageLocationKind:
		return generateVerbsForNamedCBSLKindInNamespace(groupName)
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs for group %q, kind %q and namespace %q", groupName, resourceKind, namespace)
}

func isAcceptedNamespace(namespace string) bool {
	return namespace == resources.KubermaticNamespace || strings.HasPrefix(namespace, resources.KubeOneNamespacePrefix)
}

func generateVerbsForNamespacedSecretKind(groupName string) ([]string, error) {
	switch {
	case strings.HasPrefix(groupName, OwnerGroupNamePrefix):
		return []string{"create"}, nil
	case strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix):
		return []string{"create"}, nil
	default:
		return nil, nil
	}
}

func generateVerbsForNamespacedCBSLKind(groupName string) ([]string, error) {
	switch {
	case strings.HasPrefix(groupName, OwnerGroupNamePrefix):
		return []string{"create"}, nil
	case strings.HasPrefix(groupName, EditorGroupNamePrefix):
		return []string{"create"}, nil
	case strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix):
		return []string{"create"}, nil
	default:
		return nil, nil
	}
}

func generateVerbsForNamedSecretKindInNamespace(groupName string) ([]string, error) {
	switch {
	case strings.HasPrefix(groupName, OwnerGroupNamePrefix):
		return []string{"get", "update", "delete"}, nil
	case strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix):
		return []string{"get", "update", "delete"}, nil
	default:
		return nil, nil
	}
}

func generateVerbsForNamedCBSLKindInNamespace(groupName string) ([]string, error) {
	switch {
	case strings.HasPrefix(groupName, OwnerGroupNamePrefix):
		return []string{"get", "list", "create", "patch", "update", "delete"}, nil
	case strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix):
		return []string{"get", "list", "create", "patch", "update", "delete"}, nil
	case strings.HasPrefix(groupName, EditorGroupNamePrefix):
		return []string{"get", "list", "create", "patch", "update", "delete"}, nil
	case strings.HasPrefix(groupName, ViewerGroupNamePrefix):
		return []string{"get", "list"}, nil
	default:
		return nil, nil
	}
}

func generateVerbsForClusterNamespaceResource(cluster *kubermaticv1.Cluster, groupName, kind string) ([]string, error) {
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) &&
		(kind == kubermaticv1.AddonKindName || kind == kubermaticv1.ConstraintKind || kind == kubermaticv1.RuleGroupKindName ||
			kind == kubermaticv1.EtcdBackupConfigKindName || kind == kubermaticv1.EtcdRestoreKindName) {
		return []string{"get", "list"}, nil
	}

	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) || strings.HasPrefix(groupName, EditorGroupNamePrefix) {
		return []string{"get", "list", "create", "update", "delete"}, nil
	}

	if strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix) {
		return nil, nil
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs for cluster namespace resource cluster %q, group %q and kind %q", cluster.Name, groupName, kind)
}

func generateVerbsForClusterNamespaceNamedResource(cluster *kubermaticv1.Cluster, groupName, kind, name string) ([]string, error) {
	if strings.HasPrefix(groupName, ViewerGroupNamePrefix) {
		if (kind == kubermaticv1.AlertmanagerKindName && name == alertmanagerName) ||
			(kind == secretV1Kind && name == defaultAlertmanagerConfigSecretName) {
			return []string{"get"}, nil
		}
	}

	if strings.HasPrefix(groupName, OwnerGroupNamePrefix) || strings.HasPrefix(groupName, EditorGroupNamePrefix) {
		if kind == kubermaticv1.AlertmanagerKindName && name == alertmanagerName {
			return []string{"get", "update"}, nil
		}
		if kind == secretV1Kind && name == defaultAlertmanagerConfigSecretName {
			return []string{"get", "update", "delete"}, nil
		}
	}

	if strings.HasPrefix(groupName, ProjectManagerGroupNamePrefix) {
		return nil, nil
	}

	// unknown group passed
	return nil, fmt.Errorf("unable to generate verbs for cluster namespace resource cluster %q, group %q, kind %q and name %q", cluster.Name, groupName, kind, name)
}

func formatMapping(rmapping *meta.RESTMapping) string {
	return rmapping.GroupVersionKind.Kind
}
